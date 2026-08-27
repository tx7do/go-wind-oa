package service

import (
	"context"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/tx7do/go-crud/viewer"

	"go-wind-oa/app/core/service/internal/data"
	"go-wind-oa/app/core/service/internal/data/ent"
	"go-wind-oa/app/core/service/internal/data/ent/workflowtask"

	internalMessageV1 "go-wind-oa/api/gen/go/internal_message/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// ===================== 图模型常量 =====================

// 节点类型。
const (
	nodeTypeStart                = "START"
	nodeTypeEnd                  = "END"
	nodeTypeTask                 = "TASK"
	nodeTypeExclusiveGateway     = "EXCLUSIVE_GATEWAY"
	nodeTypeParallelGatewayFork  = "PARALLEL_GATEWAY_FORK"
	nodeTypeParallelGatewayJoin  = "PARALLEL_GATEWAY_JOIN"
	nodeTypeSubprocess           = "SUBPROCESS"
)

// 审批人类型（node_config.approvers[].type）。
const (
	approverTypeUser     = "USER"     // 显式指定用户
	approverTypeLeader   = "LEADER"   // 申请人主组织单元的负责人（org_unit.leader_id）
	approverTypePosition = "POSITION" // 指定居位的在职持有者（可解析出多人）
)

// 节点审批策略（node_config.strategy）。
const (
	strategyAll = "ALL" // 会签：全员通过才推进，任一驳回即驳回（默认）
	strategyAny = "ANY" // 或签：任一通过即推进，全员驳回才驳回
)

const conditionDefault = "default"

// ===================== 图模型数据结构 =====================

// workflowApprover 单个审批人规格。type=LEADER 时无需 ID（按申请人动态解析）。
type workflowApprover struct {
	Type string `json:"type"`
	ID   uint32 `json:"id,omitempty"`
}

// taskNodeApproverConfig TASK 节点的审批人配置（新格式）。
type taskNodeApproverConfig struct {
	Approvers []workflowApprover `json:"approvers,omitempty"`
	Strategy  string             `json:"strategy,omitempty"`

	// 旧格式字段（向后兼容，仅 approver_type=="USER" 被接受）。
	ApproverType string `json:"approver_type,omitempty"`
	Approver     uint32 `json:"approver,omitempty"`
}

// normalizedApprovers 返回归一化后的审批人规格列表。
func (n *taskNodeApproverConfig) normalizedApprovers() []workflowApprover {
	if len(n.Approvers) > 0 {
		return n.Approvers
	}
	if n.ApproverType == approverTypeUser && n.Approver != 0 {
		return []workflowApprover{{Type: approverTypeUser, ID: n.Approver}}
	}
	return nil
}

// isAnyStrategy 是否或签。缺省/未知值按会签处理。
func (n *taskNodeApproverConfig) isAnyStrategy() bool {
	return strings.EqualFold(n.Strategy, strategyAny)
}

// graphNode 图节点（解析后内存表示）。
type graphNode struct {
	ID                   string
	Type                 string
	Task                 *taskNodeApproverConfig // 仅 TASK 节点非 nil
	SubprocessDefinition string                  // 仅 SUBPROCESS 节点非空（子流程引用 "code:version"）
	InDegree             int                     // 入度（边数）
	OutEdges             []graphEdge
}

// graphEdge 图边。
type graphEdge struct {
	From      string
	To        string
	Condition string // 仅 EXCLUSIVE_GATEWAY 的出边非空（"default" 或表达式）
}

// workflowGraph 图的内存表示。
type workflowGraph struct {
	Nodes map[string]*graphNode // id→节点
}

// ===================== 图格式 JSON 结构（定义侧） =====================

type graphNodeJSON struct {
	ID                   string             `json:"id"`
	Type                 string             `json:"type"`
	Approvers            []workflowApprover `json:"approvers,omitempty"`
	Strategy             string             `json:"strategy,omitempty"`
	SubprocessDefinition string             `json:"subprocessDefinition,omitempty"`
	// 旧格式（仅 TASK 旧数组兼容路径用）
	ApproverType string `json:"approver_type,omitempty"`
	Approver     uint32 `json:"approver,omitempty"`
}

type graphEdgeJSON struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Condition string `json:"condition,omitempty"`
}

type workflowGraphJSON struct {
	Version int             `json:"version"`
	Nodes   []graphNodeJSON `json:"nodes"`
	Edges   []graphEdgeJSON `json:"edges"`
}

// ===================== Service =====================

type WorkflowService struct {
	oaV1.UnimplementedWorkflowServiceServer

	log *log.Helper

	definitionRepo *data.WorkflowDefinitionRepo
	instanceRepo   *data.WorkflowInstanceRepo
	taskRepo       *data.WorkflowTaskRepo
	logRepo        *data.WorkflowLogRepo
	resolverRepo   *data.WorkflowResolverRepo
	joinRepo       *data.WorkflowInstanceJoinRepo
	parentLinkRepo *data.WorkflowInstanceParentLinkRepo
	delegationRepo *data.WorkflowDelegationRepo

	eventRegistry *WorkflowEventRegistry

	notificationService *InternalMessageService
}

func NewWorkflowService(
	ctx *bootstrap.Context,
	definitionRepo *data.WorkflowDefinitionRepo,
	instanceRepo *data.WorkflowInstanceRepo,
	taskRepo *data.WorkflowTaskRepo,
	logRepo *data.WorkflowLogRepo,
	resolverRepo *data.WorkflowResolverRepo,
	joinRepo *data.WorkflowInstanceJoinRepo,
	parentLinkRepo *data.WorkflowInstanceParentLinkRepo,
	delegationRepo *data.WorkflowDelegationRepo,
	eventRegistry *WorkflowEventRegistry,
	notificationService *InternalMessageService,
) *WorkflowService {
	return &WorkflowService{
		log:                  ctx.NewLoggerHelper("workflow/service/core-service"),
		definitionRepo:       definitionRepo,
		instanceRepo:         instanceRepo,
		taskRepo:             taskRepo,
		logRepo:              logRepo,
		resolverRepo:         resolverRepo,
		joinRepo:             joinRepo,
		parentLinkRepo:       parentLinkRepo,
		delegationRepo:       delegationRepo,
		eventRegistry:        eventRegistry,
		notificationService:  notificationService,
	}
}

// callerFromContext 从 viewer context 取 (tenantID, userID)，二者任一为 0 即 fail-closed。
func callerFromContext(ctx context.Context) (uint32, uint32, bool) {
	vc, exist := viewer.FromContext(ctx)
	if !exist || vc == nil {
		return 0, 0, false
	}
	tid := uint32(vc.TenantID())
	uid := uint32(vc.UserID())
	if uid == 0 {
		return 0, 0, false
	}
	return tid, uid, true
}

// ===================== 定义管理 =====================

func (s *WorkflowService) CreateWorkflowDefinition(ctx context.Context, req *oaV1.CreateWorkflowDefinitionRequest) (*oaV1.WorkflowDefinition, error) {
	tid, uid, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}

	// 强制落 DRAFT，忽略客户端传入的 definition_status。
	if req.Data != nil {
		req.Data.DefinitionStatus = oaV1.WorkflowDefinition_DRAFT.Enum()
		req.Data.TenantId = trans.Ptr(tid)
		req.Data.CreatedBy = trans.Ptr(uid)
	}

	return s.definitionRepo.Create(ctx, req)
}

func (s *WorkflowService) ListWorkflowDefinition(ctx context.Context, req *paginationV1.PagingRequest) (*oaV1.ListWorkflowDefinitionResponse, error) {
	return s.definitionRepo.List(ctx, req)
}

func (s *WorkflowService) GetWorkflowDefinition(ctx context.Context, req *oaV1.GetWorkflowDefinitionRequest) (*oaV1.WorkflowDefinition, error) {
	return s.definitionRepo.Get(ctx, req)
}

// UpdateWorkflowDefinition 仅允许切换 definition_status。校验 update_mask。
func (s *WorkflowService) UpdateWorkflowDefinition(ctx context.Context, req *oaV1.UpdateWorkflowDefinitionRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, oaV1.ErrorBadRequest("invalid parameter")
	}
	mask := req.GetUpdateMask()
	paths := mask.GetPaths()
	if len(paths) != 1 || paths[0] != "definition_status" {
		return nil, oaV1.ErrorBadRequest("only definition_status can be updated")
	}

	if err := s.definitionRepo.UpdateStatus(ctx, req.GetId(), req.Data.DefinitionStatus); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// ===================== 申请提交 =====================

func (s *WorkflowService) SubmitApply(ctx context.Context, req *oaV1.SubmitApplyRequest) (*oaV1.SubmitApplyResponse, error) {
	tid, uid, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}

	// 1. 取定义，校验 ENABLED。
	def, err := s.definitionRepo.GetByCodeVersion(ctx, req.GetCode(), req.GetVersion())
	if err != nil {
		return nil, err
	}
	if def.DefinitionStatus == nil || *def.DefinitionStatus != oaV1.WorkflowDefinition_ENABLED {
		return nil, oaV1.ErrorBadRequest("definition not enabled")
	}

	// 2. 解析图配置。
	graph, err := parseWorkflowGraph(def.GetNodeConfig())
	if err != nil {
		return nil, err
	}
	if err := validateGraph(graph); err != nil {
		return nil, err
	}
	startNode := graph.startNode()
	if startNode == nil {
		return nil, oaV1.ErrorBadRequest("definition has no start node")
	}

	// 3+4. 创建实例 + 写 SUBMIT 日志：原子提交。
	var instanceID uint32
	if err := s.instanceRepo.Txn(ctx, func(tx *ent.Tx) error {
		id, err := s.instanceRepo.CreateWithTx(ctx, tx, tid, uid, def.GetId(), req.GetFormData(), req.GetBusinessType(), req.GetBusinessId())
		if err != nil {
			return err
		}
		instanceID = id
		if _, err := s.logRepo.CreateWithTx(ctx, tx, tid, uid, instanceID, "", oaV1.WorkflowLog_SUBMIT.Enum(), ""); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// 5. 从 START 节点启动图遍历。
	if err := s.advanceFromNode(ctx, tid, uid, instanceID, startNode.ID, newWalkerState()); err != nil {
		return nil, err
	}

	return &oaV1.SubmitApplyResponse{InstanceId: instanceID}, nil
}

// ===================== 审批 =====================

func (s *WorkflowService) AuditTask(ctx context.Context, req *oaV1.AuditTaskRequest) (*emptypb.Empty, error) {
	tid, uid, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}

	// 1. 取任务状态（直读 entity + WithInstance 边加载）。
	assignee, taskPending, instanceID, nodeID, err := s.taskRepo.GetState(ctx, req.GetTaskId(), tid)
	if err != nil {
		return nil, err
	}

	// 2. 校验：assignee==caller 且 task.PENDING 且 instance 活跃。
	if assignee != uid || !taskPending {
		return nil, oaV1.ErrorForbidden("not your pending task")
	}
	instanceActive, err := s.instanceRepo.GetState(ctx, instanceID, tid)
	if err != nil {
		return nil, err
	}
	if !instanceActive {
		return nil, oaV1.ErrorConflict("instance not active")
	}

	// 3. 分派。
	switch req.GetAction() {
	case oaV1.AuditAction_APPROVE:
		return s.handleApprove(ctx, tid, uid, req.GetTaskId(), instanceID, nodeID, req.GetComment())
	case oaV1.AuditAction_REJECT:
		return s.handleReject(ctx, tid, uid, req.GetTaskId(), instanceID, nodeID, req.GetComment())
	case oaV1.AuditAction_FORWARD:
		return s.handleForward(ctx, tid, uid, req.GetTaskId(), instanceID, nodeID, req.GetForwardTo())
	case oaV1.AuditAction_ADD_APPROVER:
		return s.handleAddApprover(ctx, tid, uid, req.GetTaskId(), instanceID, nodeID, req.GetAdditionalApprover())
	default:
		return nil, oaV1.ErrorBadRequest("invalid audit action")
	}
}

func (s *WorkflowService) handleApprove(
	ctx context.Context, tid, uid uint32, taskID, instanceID uint32,
	nodeID string, comment string,
) (*emptypb.Empty, error) {
	// 关闭当前任务 + 写 APPROVE 日志：原子提交。
	if err := s.instanceRepo.Txn(ctx, func(tx *ent.Tx) error {
		if err := s.taskRepo.UpdateStatusWithTx(ctx, tx, taskID, tid, oaV1.WorkflowTask_APPROVED.Enum()); err != nil {
			return err
		}
		if _, err := s.logRepo.CreateWithTx(ctx, tx, tid, uid, instanceID, nodeID, oaV1.WorkflowLog_APPROVE.Enum(), comment); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// 取图与当前节点。
	graph, err := s.graphOfInstance(ctx, instanceID, tid)
	if err != nil {
		return nil, err
	}
	node := graph.Nodes[nodeID]
	if node == nil {
		return nil, oaV1.ErrorConflict("instance state corrupt")
	}

	if node.Task.isAnyStrategy() {
		// 或签：一人通过即推进，取消其余待办任务。
		if err := s.taskRepo.CancelPendingByInstanceNode(ctx, tid, instanceID, nodeID, taskID); err != nil {
			return nil, err
		}
		if err := s.advanceFromNode(ctx, tid, uid, instanceID, nodeID, newWalkerState()); err != nil {
			return nil, err
		}
		return &emptypb.Empty{}, nil
	}

	// 会签：全部任务通过才推进；仍有待办则等待其他审批人。
	statuses, err := s.taskRepo.ListNodeTaskStatuses(ctx, tid, instanceID, nodeID)
	if err != nil {
		return nil, err
	}
	for _, st := range statuses {
		if st == workflowtask.TaskStatusPending {
			return &emptypb.Empty{}, nil
		}
	}
	if err := s.advanceFromNode(ctx, tid, uid, instanceID, nodeID, newWalkerState()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *WorkflowService) handleReject(
	ctx context.Context, tid, uid uint32, taskID, instanceID uint32,
	nodeID string, comment string,
) (*emptypb.Empty, error) {
	// 关闭任务（REJECTED）+ 写 REJECT 日志：原子提交。
	if err := s.instanceRepo.Txn(ctx, func(tx *ent.Tx) error {
		if err := s.taskRepo.UpdateStatusWithTx(ctx, tx, taskID, tid, oaV1.WorkflowTask_REJECTED.Enum()); err != nil {
			return err
		}
		if _, err := s.logRepo.CreateWithTx(ctx, tx, tid, uid, instanceID, nodeID, oaV1.WorkflowLog_REJECT.Enum(), comment); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// 图模型下拒绝统一为全实例终结：取消全部待办 + 删除 join 行 + 删除 parent_link + 实例 REJECTED。
	// 并行场景下任一分支拒绝，其他分支必须同步杀掉，否则残留半死分支。
	if err := s.instanceRepo.Txn(ctx, func(tx *ent.Tx) error {
		if err := s.taskRepo.CancelAllPendingByInstanceWithTx(ctx, tx, tid, instanceID); err != nil {
			return err
		}
		if err := s.joinRepo.DeleteByInstanceWithTx(ctx, tx, tid, instanceID); err != nil {
			return err
		}
		if err := s.parentLinkRepo.DeleteByParentInstanceWithTx(ctx, tx, tid, instanceID); err != nil {
			return err
		}
		if err := s.instanceRepo.UpdateStatusWithTx(ctx, tx, instanceID, tid, oaV1.WorkflowInstance_REJECTED.Enum()); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// 通知申请人（实例创建者，而非驳回操作人），并回调业务模块。
	applicant, businessType, businessID, err := s.instanceRepo.GetMeta(ctx, instanceID, tid)
	if err != nil {
		return nil, err
	}
	s.notifyManyAsync(ctx, []uint32{applicant}, "您的申请已被驳回", "请登录系统查看详情")
	s.fireBusinessEvent(ctx, tid, instanceID, businessID, businessType, oaV1.WorkflowInstance_REJECTED)
	return &emptypb.Empty{}, nil
}

func (s *WorkflowService) handleForward(
	ctx context.Context, tid, uid uint32, taskID, instanceID uint32, nodeID string, forwardTo uint32,
) (*emptypb.Empty, error) {
	if forwardTo == 0 || forwardTo == uid {
		return nil, oaV1.ErrorBadRequest("invalid forward target")
	}
	active, err := s.resolverRepo.UserIsActive(ctx, tid, forwardTo)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, oaV1.ErrorBadRequest("forward target user is unavailable")
	}
	if err := s.instanceRepo.Txn(ctx, func(tx *ent.Tx) error {
		if err := s.taskRepo.UpdateAssigneeWithTx(ctx, tx, taskID, tid, forwardTo); err != nil {
			return err
		}
		if _, err := s.logRepo.CreateWithTx(ctx, tx, tid, uid, instanceID, nodeID, oaV1.WorkflowLog_FORWARD.Enum(), ""); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	s.notifyManyAsync(ctx, []uint32{forwardTo}, "您有被转办的任务", "请登录系统查看待审批事项")
	return &emptypb.Empty{}, nil
}

// handleAddApprover 加签：当前审批人在审批过程中为该节点临时增加一名审批人。
// 新增的任务与原有任务并列存在，遵守该节点的会签/或签策略。
// 会签策略下，新审批人的任务也需通过才会签才推进。
func (s *WorkflowService) handleAddApprover(
	ctx context.Context, tid, uid uint32, taskID, instanceID uint32, nodeID string, additionalApprover uint32,
) (*emptypb.Empty, error) {
	if additionalApprover == 0 || additionalApprover == uid {
		return nil, oaV1.ErrorBadRequest("invalid additional approver")
	}
	active, err := s.resolverRepo.UserIsActive(ctx, tid, additionalApprover)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, oaV1.ErrorBadRequest("additional approver user is unavailable")
	}

	// 取图节点，确认当前节点仍是 TASK 且实例活跃（已在 AuditTask 入口校验，此处取节点用于日志）。
	graph, err := s.graphOfInstance(ctx, instanceID, tid)
	if err != nil {
		return nil, err
	}
	node := graph.Nodes[nodeID]
	if node == nil || node.Type != nodeTypeTask {
		return nil, oaV1.ErrorConflict("instance state corrupt")
	}

	// 在当前节点为新审批人建任务 + 写加签日志：原子提交。
	if err := s.instanceRepo.Txn(ctx, func(tx *ent.Tx) error {
		if _, err := s.taskRepo.CreateWithTx(ctx, tx, tid, uid, instanceID, nodeID, additionalApprover); err != nil {
			return err
		}
		if _, err := s.logRepo.CreateWithTx(ctx, tx, tid, uid, instanceID, nodeID, oaV1.WorkflowLog_FORWARD.Enum(), "加签：新增审批人"); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	s.notifyManyAsync(ctx, []uint32{additionalApprover}, "您有被加签的任务", "请登录系统查看待审批事项")
	return &emptypb.Empty{}, nil
}

// ===================== 图遍历 walker =====================

// walkerState 跟踪单次推进调用树的步数，防止失控（回退边允许环路，靠步数上限兜底）。
type walkerState struct {
	steps int
}

func newWalkerState() *walkerState {
	return &walkerState{}
}

const walkerStepLimit = 1000

// advanceFromNode 从指定节点开始图遍历。按节点类型分派：
//   - START/END/FORK：沿出边继续（END 先检查全局活跃节点）
//   - TASK：解析审批人，剔除申请人；全为申请人则 auto-skip 继续，否则建任务+通知并停
//   - EXCLUSIVE_GATEWAY：对出边条件求值选一条，无匹配走 default，无 default 则 fail-closed REJECTED
//   - PARALLEL_GATEWAY_JOIN：递增到达计数，满入度则合并继续，未满则停
func (s *WorkflowService) advanceFromNode(
	ctx context.Context, tid, uid uint32, instanceID uint32, fromNodeID string, ws *walkerState,
) error {
	// 步数守卫：防失控。回退边允许环路，靠步数上限兜底，不靠 visited 拒环。
	ws.steps++
	if ws.steps > walkerStepLimit {
		return s.failClosedReject(ctx, tid, uid, instanceID, "walker step limit exceeded")
	}

	graph, err := s.graphOfInstance(ctx, instanceID, tid)
	if err != nil {
		return err
	}
	node := graph.Nodes[fromNodeID]
	if node == nil {
		return oaV1.ErrorConflict("instance state corrupt")
	}

	switch node.Type {
	case nodeTypeStart:
		if len(node.OutEdges) != 1 {
			return s.failClosedReject(ctx, tid, uid, instanceID, "start node must have exactly one out edge")
		}
		return s.advanceFromNode(ctx, tid, uid, instanceID, node.OutEdges[0].To, ws)

	case nodeTypeTask:
		applicant, _, _, err := s.instanceRepo.GetMeta(ctx, instanceID, tid)
		if err != nil {
			return err
		}
		approvers, err := s.resolveApprovers(ctx, tid, applicant, node.Task)
		if err != nil {
			return err
		}
		others := make([]uint32, 0, len(approvers))
		for _, approver := range approvers {
			if approver != applicant {
				others = append(others, approver)
			}
		}
		if len(others) == 0 {
			// 申请人是该节点唯一审批人：清除该节点残留任务（防回退后旧 APPROVED 污染收敛），自动通过，推进下一节点。
			_ = s.taskRepo.CancelAllByInstanceNode(ctx, tid, instanceID, node.ID)
			_, _ = s.logRepo.Create(ctx, tid, uid, instanceID, node.ID, oaV1.WorkflowLog_APPROVE.Enum(), "审批人为申请人本人，自动通过")
			if len(node.OutEdges) != 1 {
				return s.failClosedReject(ctx, tid, uid, instanceID, "task node must have exactly one out edge")
			}
			return s.advanceFromNode(ctx, tid, uid, instanceID, node.OutEdges[0].To, ws)
		}
		// 清除该节点残留任务（防回退后旧 APPROVED 污染收敛判定）+ 建新任务 + 置实例 PENDING：原子提交。
		if err := s.instanceRepo.Txn(ctx, func(tx *ent.Tx) error {
			if err := s.taskRepo.CancelAllByInstanceNodeWithTx(ctx, tx, tid, instanceID, node.ID); err != nil {
				return err
			}
			if err := s.instanceRepo.UpdateStatusWithTx(ctx, tx, instanceID, tid, oaV1.WorkflowInstance_PENDING.Enum()); err != nil {
				return err
			}
			for _, approver := range others {
				if _, err := s.taskRepo.CreateWithTx(ctx, tx, tid, uid, instanceID, node.ID, approver); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		s.notifyManyAsync(ctx, others, "您有新的待办任务", "请登录系统查看待审批事项")
		return nil

	case nodeTypeExclusiveGateway:
		// 对出边条件求值；命中第一条 true 的边；全 false 走 default；无 default 则 fail-closed。
		formData, _ := s.instanceRepo.GetFormData(ctx, instanceID, tid)
		nextNode, defaultEdge := s.selectExclusiveEdge(ctx, node, formData)
		if nextNode != "" {
			return s.advanceFromNode(ctx, tid, uid, instanceID, nextNode, ws)
		}
		if defaultEdge != "" {
			return s.advanceFromNode(ctx, tid, uid, instanceID, defaultEdge, ws)
		}
		return s.failClosedReject(ctx, tid, uid, instanceID, "exclusive gateway no matching condition and no default")

	case nodeTypeParallelGatewayFork:
		// 对每条出边递归（token 分裂为多）。
		for _, e := range node.OutEdges {
			if err := s.advanceFromNode(ctx, tid, uid, instanceID, e.To, ws); err != nil {
				return err
			}
		}
		return nil

	case nodeTypeParallelGatewayJoin:
		newCount, err := s.joinRepo.Increment(ctx, tid, instanceID, node.ID)
		if err != nil {
			return err
		}
		if newCount >= node.InDegree {
			// 所有分支到达，合并为单 token 继续。
			if err := s.joinRepo.DeleteByInstance(ctx, tid, instanceID); err != nil {
				return err
			}
			if len(node.OutEdges) != 1 {
				return s.failClosedReject(ctx, tid, uid, instanceID, "join node must have exactly one out edge")
			}
			return s.advanceFromNode(ctx, tid, uid, instanceID, node.OutEdges[0].To, ws)
		}
		// 未满：token 停此 join，等待其余分支。
		return nil

	case nodeTypeEnd:
		// 本分支 token 消亡；检查全局是否还有活跃节点。
		return s.checkTermination(ctx, tid, uid, instanceID)

	case nodeTypeSubprocess:
		return s.handleSubprocess(ctx, tid, uid, instanceID, node, ws)

	default:
		return s.failClosedReject(ctx, tid, uid, instanceID, "unknown node type: "+node.Type)
	}
}

// handleSubprocess 处理 SUBPROCESS 节点：创建子流程实例，挂起父实例，启动子流程遍历。
// 子流程终结后由 maybeResumeParent 恢复父流程从 SUBPROCESS 出边继续推进。
func (s *WorkflowService) handleSubprocess(
	ctx context.Context, tid, uid uint32, instanceID uint32, node *graphNode, ws *walkerState,
) error {
	if node.SubprocessDefinition == "" {
		return s.failClosedReject(ctx, tid, uid, instanceID, "subprocess node missing subprocessDefinition")
	}

	// 解析 "code:version" 引用。
	parts := strings.SplitN(node.SubprocessDefinition, ":", 2)
	if len(parts) != 2 {
		return s.failClosedReject(ctx, tid, uid, instanceID, "subprocessDefinition format invalid")
	}
	subCode := parts[0]
	subVersion := int32(0)
	for _, c := range parts[1] {
		if c < '0' || c > '9' {
			return s.failClosedReject(ctx, tid, uid, instanceID, "subprocessDefinition version invalid")
		}
		subVersion = subVersion*10 + int32(c-'0')
	}
	if subCode == "" || subVersion == 0 {
		return s.failClosedReject(ctx, tid, uid, instanceID, "subprocessDefinition code or version empty")
	}

	// 取子定义，校验 ENABLED。
	subDef, err := s.definitionRepo.GetByCodeVersion(ctx, subCode, subVersion)
	if err != nil {
		return s.failClosedReject(ctx, tid, uid, instanceID, "subprocess definition not found: "+err.Error())
	}
	if subDef.DefinitionStatus == nil || *subDef.DefinitionStatus != oaV1.WorkflowDefinition_ENABLED {
		return s.failClosedReject(ctx, tid, uid, instanceID, "subprocess definition not enabled")
	}

	// 解析子图配置。
	subGraph, err := parseWorkflowGraph(subDef.GetNodeConfig())
	if err != nil {
		return s.failClosedReject(ctx, tid, uid, instanceID, "subprocess graph parse failed: "+err.Error())
	}
	if err := validateGraph(subGraph); err != nil {
		return s.failClosedReject(ctx, tid, uid, instanceID, "subprocess graph invalid: "+err.Error())
	}
	subStart := subGraph.startNode()
	if subStart == nil {
		return s.failClosedReject(ctx, tid, uid, instanceID, "subprocess graph has no start node")
	}

	// 创建子实例 + 父子关联 + 挂起父实例：原子提交。
	var childInstanceID uint32
	if err := s.instanceRepo.Txn(ctx, func(tx *ent.Tx) error {
		// 子实例：business_type/business_id 清空（子流程不直接关联业务单据）。
		cid, err := s.instanceRepo.CreateWithTx(ctx, tx, tid, uid, subDef.GetId(), "", "", 0)
		if err != nil {
			return err
		}
		childInstanceID = cid
		// 父子关联。
		if err := s.parentLinkRepo.Create(ctx, tid, instanceID, node.ID, childInstanceID); err != nil {
			return err
		}
		// 挂起父实例。
		if err := s.instanceRepo.UpdateStatusWithTx(ctx, tx, instanceID, tid, oaV1.WorkflowInstance_SUSPENDED.Enum()); err != nil {
			return err
		}
		// 子实例写 SUBMIT 日志。
		if _, err := s.logRepo.CreateWithTx(ctx, tx, tid, uid, childInstanceID, "", oaV1.WorkflowLog_SUBMIT.Enum(), ""); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	// 启动子流程遍历（从子图的 START 节点）。
	return s.advanceFromNode(ctx, tid, uid, childInstanceID, subStart.ID, newWalkerState())
}

// selectExclusiveEdge 对 EXCLUSIVE_GATEWAY 的出边条件求值，返回命中的 to 节点 id
// 和 default 边的 to 节点 id。无命中则 to 为空，此时若有 default 则返回 default。
func (s *WorkflowService) selectExclusiveEdge(
	ctx context.Context, node *graphNode, formDataJSON string,
) (hitTo, defaultTo string) {
	for _, e := range node.OutEdges {
		if e.Condition == conditionDefault {
			defaultTo = e.To
			continue
		}
		result, err := evalCondition(e.Condition, formDataJSON)
		if err != nil {
			s.log.Errorf("condition eval error on node %s: %s", node.ID, err.Error())
			continue
		}
		if result {
			return e.To, ""
		}
	}
	return "", defaultTo
}

// checkTermination 检查实例是否还有活跃节点（PENDING 任务、未满 join、或挂起的子流程）。
// 无活跃节点则终结 APPROVED + 回调业务模块。
// 若此实例是子流程，终结后检查是否有父流程挂起等待它，有则恢复父流程。
func (s *WorkflowService) checkTermination(
	ctx context.Context, tid, uid uint32, instanceID uint32,
) error {
	hasTasks, err := s.taskRepo.HasPendingByInstance(ctx, tid, instanceID)
	if err != nil {
		return err
	}
	hasJoins, err := s.joinRepo.HasByInstance(ctx, tid, instanceID)
	if err != nil {
		return err
	}
	hasChildSubprocess, err := s.parentLinkRepo.HasByParentInstance(ctx, tid, instanceID)
	if err != nil {
		return err
	}
	if hasTasks || hasJoins || hasChildSubprocess {
		// 其他分支或子流程仍在跑，本分支 token 消亡即可。
		return nil
	}
	// 全部分支走完：终结 APPROVED。
	if err := s.instanceRepo.UpdateStatus(ctx, instanceID, tid, oaV1.WorkflowInstance_APPROVED.Enum()); err != nil {
		return err
	}
	applicant, businessType, businessID, err := s.instanceRepo.GetMeta(ctx, instanceID, tid)
	if err != nil {
		return err
	}
	s.notifyManyAsync(ctx, []uint32{applicant}, "您的申请已通过", "全部审批节点已通过")
	s.fireBusinessEvent(ctx, tid, instanceID, businessID, businessType, oaV1.WorkflowInstance_APPROVED)

	// 子流程终结后检查是否有父流程挂起等待此子流程。
	s.maybeResumeParent(ctx, tid, uid, instanceID)
	return nil
}

// maybeResumeParent 检查此实例是否是某父流程的子流程。若是，删除关联记录、
// 恢复父流程从 SUBPROCESS 节点继续推进。
func (s *WorkflowService) maybeResumeParent(
	ctx context.Context, tid, uid uint32, childInstanceID uint32,
) {
	parentInstanceID, subprocessNodeID, err := s.parentLinkRepo.GetByChildInstance(ctx, tid, childInstanceID)
	if err != nil {
		// 无父关联：普通流程终结，正常。
		return
	}

	// 删除关联记录（子流程已完成，不再挂起父流程）。
	if err := s.parentLinkRepo.DeleteByChildInstance(ctx, tid, childInstanceID); err != nil {
		s.log.Errorf("delete parent link after child completion failed: %s", err.Error())
		return
	}

	// 恢复父流程：状态从 SUSPENDED 切回 PENDING，从 SUBPROCESS 节点的出边继续推进。
	if err := s.instanceRepo.UpdateStatus(ctx, parentInstanceID, tid, oaV1.WorkflowInstance_PENDING.Enum()); err != nil {
		s.log.Errorf("resume parent instance failed: %s", err.Error())
		return
	}
	if err := s.advanceFromNode(ctx, tid, uid, parentInstanceID, subprocessNodeID, newWalkerState()); err != nil {
		s.log.Errorf("resume parent walker failed: %s", err.Error())
	}
}

// failClosedReject 路由失败/定义异常的 fail-closed 终结：取消全部待办 + 删除 join + 删除 parent_link + REJECTED。
func (s *WorkflowService) failClosedReject(
	ctx context.Context, tid, uid uint32, instanceID uint32, reason string,
) error {
	s.log.Errorf("workflow fail-closed reject: instance=%d reason=%s", instanceID, reason)
	if err := s.instanceRepo.Txn(ctx, func(tx *ent.Tx) error {
		if err := s.taskRepo.CancelAllPendingByInstanceWithTx(ctx, tx, tid, instanceID); err != nil {
			return err
		}
		if err := s.joinRepo.DeleteByInstanceWithTx(ctx, tx, tid, instanceID); err != nil {
			return err
		}
		if err := s.parentLinkRepo.DeleteByParentInstanceWithTx(ctx, tx, tid, instanceID); err != nil {
			return err
		}
		if err := s.instanceRepo.UpdateStatusWithTx(ctx, tx, instanceID, tid, oaV1.WorkflowInstance_REJECTED.Enum()); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	applicant, businessType, businessID, err := s.instanceRepo.GetMeta(ctx, instanceID, tid)
	if err != nil {
		return err
	}
	s.notifyManyAsync(ctx, []uint32{applicant}, "您的申请已被驳回", "流程定义异常，请联系管理员")
	s.fireBusinessEvent(ctx, tid, instanceID, businessID, businessType, oaV1.WorkflowInstance_REJECTED)
	return nil
}

// ===================== 撤回 =====================

// WithdrawApply 申请人撤回自己的进行中申请：实例转 WITHDRAWN，全部待办任务取消，
// 删除 join 行，写 WITHDRAW 日志，通知原待办审批人。
func (s *WorkflowService) WithdrawApply(ctx context.Context, req *oaV1.WithdrawApplyRequest) (*emptypb.Empty, error) {
	tid, uid, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	instanceID := req.GetInstanceId()
	if instanceID == 0 {
		return nil, oaV1.ErrorBadRequest("invalid parameter")
	}

	// 仅申请人本人可撤回。
	creator, businessType, businessID, err := s.instanceRepo.GetMeta(ctx, instanceID, tid)
	if err != nil {
		return nil, err
	}
	if creator != uid {
		return nil, oaV1.ErrorForbidden("not your application")
	}

	// 仅进行中（PENDING）实例可撤回。
	instanceActive, err := s.instanceRepo.GetState(ctx, instanceID, tid)
	if err != nil {
		return nil, err
	}
	if !instanceActive {
		return nil, oaV1.ErrorConflict("instance not active")
	}

	pendingAssignees, err := s.taskRepo.ListPendingAssigneesByInstance(ctx, tid, instanceID)
	if err != nil {
		return nil, err
	}
	if err := s.instanceRepo.Txn(ctx, func(tx *ent.Tx) error {
		if err := s.taskRepo.CancelAllPendingByInstanceWithTx(ctx, tx, tid, instanceID); err != nil {
			return err
		}
		if err := s.joinRepo.DeleteByInstanceWithTx(ctx, tx, tid, instanceID); err != nil {
			return err
		}
		if err := s.parentLinkRepo.DeleteByParentInstanceWithTx(ctx, tx, tid, instanceID); err != nil {
			return err
		}
		if err := s.instanceRepo.UpdateStatusWithTx(ctx, tx, instanceID, tid, oaV1.WorkflowInstance_WITHDRAWN.Enum()); err != nil {
			return err
		}
		if _, err := s.logRepo.CreateWithTx(ctx, tx, tid, uid, instanceID, "", oaV1.WorkflowLog_WITHDRAW.Enum(), ""); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	s.notifyManyAsync(ctx, pendingAssignees, "待审批申请已被撤回", "申请人已撤回该申请，无需继续处理")
	s.fireBusinessEvent(ctx, tid, instanceID, businessID, businessType, oaV1.WorkflowInstance_WITHDRAWN)
	return &emptypb.Empty{}, nil
}

// GetApplyForm 获取申请表单定义（提交页动态渲染用）。仅 ENABLED 定义可取；
// form_schema 为空串表示该流程无表单定义，客户端回退自由 JSON 输入。
func (s *WorkflowService) GetApplyForm(ctx context.Context, req *oaV1.GetApplyFormRequest) (*oaV1.GetApplyFormResponse, error) {
	_, _, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	if req.GetCode() == "" || req.GetVersion() == 0 {
		return nil, oaV1.ErrorBadRequest("invalid parameter")
	}
	def, err := s.definitionRepo.GetByCodeVersion(ctx, req.GetCode(), req.GetVersion())
	if err != nil {
		return nil, err
	}
	if def.DefinitionStatus == nil || *def.DefinitionStatus != oaV1.WorkflowDefinition_ENABLED {
		return nil, oaV1.ErrorBadRequest("definition not enabled")
	}
	return &oaV1.GetApplyFormResponse{FormSchema: def.GetFormSchema()}, nil
}

// ===================== 列表 / 详情 =====================

func (s *WorkflowService) GetMyTasks(ctx context.Context, req *oaV1.GetMyTasksRequest) (*oaV1.GetMyTasksResponse, error) {
	tid, uid, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}

	var items []*oaV1.MyTaskItem
	var total int
	var err error
	switch req.GetListType() {
	case oaV1.ListType_PENDING:
		items, total, err = s.taskRepo.ListPendingByAssignee(ctx, tid, uid, req.GetPage(), req.GetPageSize())
	case oaV1.ListType_DONE:
		items, total, err = s.logRepo.ListByActor(ctx, tid, uid, req.GetPage(), req.GetPageSize())
	case oaV1.ListType_SUBMITTED:
		items, total, err = s.instanceRepo.ListByCreator(ctx, tid, uid, req.GetPage(), req.GetPageSize())
	default:
		return nil, oaV1.ErrorBadRequest("invalid list type")
	}
	if err != nil {
		return nil, err
	}
	return &oaV1.GetMyTasksResponse{Items: items, Total: uint64(total)}, nil
}

func (s *WorkflowService) GetTask(ctx context.Context, req *oaV1.GetTaskRequest) (*oaV1.GetTaskResponse, error) {
	tid, uid, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}

	// 防御性重校验：仅 assignee 本人且任务 PENDING 可查看详情。
	task, instanceID, err := s.taskRepo.GetDetailByAssignee(ctx, req.GetId(), tid, uid)
	if err != nil {
		return nil, err
	}

	// 审批历史。
	logs, err := s.logRepo.ListByInstance(ctx, tid, instanceID)
	if err != nil {
		return nil, err
	}

	// 申请表单数据（审批人查看申请内容）。
	formData, err := s.instanceRepo.GetFormData(ctx, instanceID, tid)
	if err != nil {
		return nil, err
	}

	return &oaV1.GetTaskResponse{Task: task, Logs: logs, FormData: trans.Ptr(formData)}, nil
}

// ===================== 审批委托 =====================

func (s *WorkflowService) SetWorkflowDelegation(
	ctx context.Context, req *oaV1.SetWorkflowDelegationRequest,
) (*oaV1.WorkflowDelegation, error) {
	tid, uid, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	data := req.GetData()
	if data == nil {
		return nil, oaV1.ErrorBadRequest("missing delegation data")
	}
	delegator := data.GetDelegatorUserId()
	delegate := data.GetDelegateUserId()
	if delegator == 0 || delegate == 0 {
		return nil, oaV1.ErrorBadRequest("delegator and delegate user id required")
	}
	if delegator == delegate {
		return nil, oaV1.ErrorBadRequest("cannot delegate to self")
	}
	// 仅允许设置自己的委托。
	if delegator != uid {
		return nil, oaV1.ErrorForbidden("can only set delegation for yourself")
	}
	// 被委托人必须在职。
	active, err := s.resolverRepo.UserIsActive(ctx, tid, delegate)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, oaV1.ErrorBadRequest("delegate user is not active")
	}
	return s.delegationRepo.Upsert(ctx, tid, uid, delegator, delegate)
}

func (s *WorkflowService) ListWorkflowDelegation(
	ctx context.Context, req *paginationV1.PagingRequest,
) (*oaV1.ListWorkflowDelegationResponse, error) {
	tid, _, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	return s.delegationRepo.List(ctx, tid, req)
}

func (s *WorkflowService) DeleteWorkflowDelegation(
	ctx context.Context, req *oaV1.DeleteWorkflowDelegationRequest,
) (*emptypb.Empty, error) {
	tid, _, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	if err := s.delegationRepo.DeleteByID(ctx, tid, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// ===================== 辅助 =====================

// fireBusinessEvent 触发业务单据终结回调，panic 隔离并记日志，失败不影响状态机。
func (s *WorkflowService) fireBusinessEvent(
	ctx context.Context, tenantID, instanceID, businessID uint32,
	businessType string, status oaV1.WorkflowInstance_InstanceStatus,
) {
	defer func() {
		if r := recover(); r != nil {
			s.log.Errorf("business hook panic (%s): %v", businessType, r)
		}
	}()
	s.eventRegistry.Fire(ctx, tenantID, instanceID, businessID, businessType, status)
}

// notifyManyAsync 异步发送站内信通知（可多接收人）。fire-and-forget，失败不回滚状态机。
func (s *WorkflowService) notifyManyAsync(ctx context.Context, recipientUserIDs []uint32, title, content string) {
	if len(recipientUserIDs) == 0 {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.log.Errorf("notify panic: %v", r)
			}
		}()
		// WithoutCancel 保留原 ctx 的 viewer（SendMessage 从 viewer 推导发送者，
		// 防伪造），同时脱离已返回的 gRPC 请求生命周期。
		notifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_, err := s.notificationService.SendMessage(notifyCtx, &internalMessageV1.SendMessageRequest{
			Type:          internalMessageV1.InternalMessage_NOTIFICATION,
			TargetUserIds: recipientUserIDs,
			Title:         trans.Ptr(title),
			Content:       content,
		})
		if err != nil {
			s.log.Errorf("notify send failed: %s", err.Error())
		}
	}()
}
