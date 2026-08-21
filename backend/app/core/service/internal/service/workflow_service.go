package service

import (
	"context"
	"encoding/json"
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

// workflowApprover 单个审批人规格。type=LEADER 时无需 ID（按申请人动态解析）。
type workflowApprover struct {
	Type string `json:"type"`
	ID   uint32 `json:"id,omitempty"`
}

// workflowNode 定义流程节点配置中的单个节点结构。node_config 字段是这些节点的 JSON 数组。
// 新格式：approvers（列表）+ strategy（ALL/ANY）；旧格式（approver_type+approver 单人）
// 解析时归一化为单元素 approvers 列表。
type workflowNode struct {
	Approvers []workflowApprover `json:"approvers,omitempty"`
	Strategy  string             `json:"strategy,omitempty"`

	// 旧格式字段（向后兼容，仅 approver_type=="USER" 被接受）。
	ApproverType string `json:"approver_type,omitempty"`
	Approver     uint32 `json:"approver,omitempty"`
}

// normalizedApprovers 返回归一化后的审批人规格列表。
func (n *workflowNode) normalizedApprovers() []workflowApprover {
	if len(n.Approvers) > 0 {
		return n.Approvers
	}
	if n.ApproverType == approverTypeUser && n.Approver != 0 {
		return []workflowApprover{{Type: approverTypeUser, ID: n.Approver}}
	}
	return nil
}

// isAnyStrategy 是否或签。缺省/未知值按会签处理。
func (n *workflowNode) isAnyStrategy() bool {
	return strings.EqualFold(n.Strategy, strategyAny)
}

type WorkflowService struct {
	oaV1.UnimplementedWorkflowServiceServer

	log *log.Helper

	definitionRepo *data.WorkflowDefinitionRepo
	instanceRepo   *data.WorkflowInstanceRepo
	taskRepo       *data.WorkflowTaskRepo
	logRepo        *data.WorkflowLogRepo
	resolverRepo   *data.WorkflowResolverRepo

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
	eventRegistry *WorkflowEventRegistry,
	notificationService *InternalMessageService,
) *WorkflowService {
	return &WorkflowService{
		log:                 ctx.NewLoggerHelper("workflow/service/core-service"),
		definitionRepo:      definitionRepo,
		instanceRepo:        instanceRepo,
		taskRepo:            taskRepo,
		logRepo:             logRepo,
		resolverRepo:        resolverRepo,
		eventRegistry:       eventRegistry,
		notificationService: notificationService,
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
	if tid == 0 || uid == 0 {
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

	// 2. 解析节点配置。
	nodes, err := parseWorkflowNodes(def.GetNodeConfig())
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, oaV1.ErrorBadRequest("definition has no nodes")
	}

	// 3+4. 创建实例 + 写 SUBMIT 日志：原子提交。建单链的跨 repo 两步写包入
	// 单一事务，任一步失败整体回滚，避免孤儿实例或日志指向不存在的实例。
	var instanceID uint32
	if err := s.instanceRepo.Txn(ctx, func(tx *ent.Tx) error {
		id, err := s.instanceRepo.CreateWithTx(ctx, tx, tid, uid, def.GetId(), req.GetFormData(), req.GetBusinessType(), req.GetBusinessId())
		if err != nil {
			return err
		}
		instanceID = id
		if _, err := s.logRepo.CreateWithTx(ctx, tx, tid, uid, instanceID, 0, oaV1.WorkflowLog_SUBMIT.Enum(), ""); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// 5. 从首节点启动（申请人为审批人的节点自动通过）。
	if err := s.launchFromNode(ctx, tid, uid, instanceID, 0); err != nil {
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
	assignee, taskPending, instanceID, currentNodeIndex, err := s.taskRepo.GetState(ctx, req.GetTaskId(), tid)
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
		return s.handleApprove(ctx, tid, uid, req.GetTaskId(), instanceID, currentNodeIndex, req.GetComment())
	case oaV1.AuditAction_REJECT:
		return s.handleReject(ctx, tid, uid, req.GetTaskId(), instanceID, currentNodeIndex, req.GetComment())
	case oaV1.AuditAction_FORWARD:
		return s.handleForward(ctx, tid, uid, req.GetTaskId(), instanceID, currentNodeIndex, req.GetForwardTo())
	default:
		return nil, oaV1.ErrorBadRequest("invalid audit action")
	}
}

func (s *WorkflowService) handleApprove(
	ctx context.Context, tid, uid uint32, taskID, instanceID uint32,
	currentNodeIndex *int, comment string,
) (*emptypb.Empty, error) {
	nodeIdx := -1
	if currentNodeIndex != nil {
		nodeIdx = *currentNodeIndex
	}
	ni := nodeIdx
	// 关闭当前任务 + 写 APPROVE 日志：原子提交。
	if err := s.instanceRepo.Txn(ctx, func(tx *ent.Tx) error {
		if err := s.taskRepo.UpdateStatusWithTx(ctx, tx, taskID, tid, oaV1.WorkflowTask_APPROVED.Enum()); err != nil {
			return err
		}
		if _, err := s.logRepo.CreateWithTx(ctx, tx, tid, uid, instanceID, ni, oaV1.WorkflowLog_APPROVE.Enum(), comment); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// 取定义节点配置与当前节点策略。
	nodes, err := s.nodesOfInstance(ctx, instanceID, tid)
	if err != nil {
		return nil, err
	}
	if nodeIdx < 0 || nodeIdx >= len(nodes) {
		return nil, oaV1.ErrorConflict("instance state corrupt")
	}

	if nodes[nodeIdx].isAnyStrategy() {
		// 或签：一人通过即推进，取消其余待办任务。
		if err := s.taskRepo.CancelPendingByInstanceNode(ctx, tid, instanceID, nodeIdx, taskID); err != nil {
			return nil, err
		}
		return s.advanceInstance(ctx, tid, uid, instanceID, nodeIdx)
	}

	// 会签：全部任务通过才推进；仍有待办则等待其他审批人。
	statuses, err := s.taskRepo.ListNodeTaskStatuses(ctx, tid, instanceID, nodeIdx)
	if err != nil {
		return nil, err
	}
	for _, st := range statuses {
		if st == workflowtask.TaskStatusPending {
			return &emptypb.Empty{}, nil
		}
	}
	return s.advanceInstance(ctx, tid, uid, instanceID, nodeIdx)
}

func (s *WorkflowService) handleReject(
	ctx context.Context, tid, uid uint32, taskID, instanceID uint32,
	currentNodeIndex *int, comment string,
) (*emptypb.Empty, error) {
	nodeIdx := -1
	if currentNodeIndex != nil {
		nodeIdx = *currentNodeIndex
	}

	// 关闭任务（REJECTED）+ 写 REJECT 日志：原子提交。
	if err := s.instanceRepo.Txn(ctx, func(tx *ent.Tx) error {
		if err := s.taskRepo.UpdateStatusWithTx(ctx, tx, taskID, tid, oaV1.WorkflowTask_REJECTED.Enum()); err != nil {
			return err
		}
		if _, err := s.logRepo.CreateWithTx(ctx, tx, tid, uid, instanceID, nodeIdx, oaV1.WorkflowLog_REJECT.Enum(), comment); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// 或签：仍有待办或已有人通过则等待他人，不终结实例。
	nodes, err := s.nodesOfInstance(ctx, instanceID, tid)
	if err != nil {
		return nil, err
	}
	if nodeIdx >= 0 && nodeIdx < len(nodes) && nodes[nodeIdx].isAnyStrategy() {
		statuses, err := s.taskRepo.ListNodeTaskStatuses(ctx, tid, instanceID, nodeIdx)
		if err != nil {
			return nil, err
		}
		hasPending, hasApproved := false, false
		for _, st := range statuses {
			if st == workflowtask.TaskStatusPending {
				hasPending = true
			}
			if st == workflowtask.TaskStatusApproved {
				hasApproved = true
			}
		}
		if hasPending || hasApproved {
			// 或签部分驳回停留：实例未终结，但已有审批人驳回。通知申请人知晓。
			applicant, _, _, _ := s.instanceRepo.GetMeta(ctx, instanceID, tid)
			if applicant != 0 {
				s.notifyManyAsync(ctx, []uint32{applicant}, "您的申请收到驳回意见", "流程仍在进行中，请登录系统查看")
			}
			return &emptypb.Empty{}, nil
		}
	}

	// 会签任一驳回即驳回；或签全员驳回亦驳回。取消其余待办 + 实例终结：原子提交。
	if err := s.instanceRepo.Txn(ctx, func(tx *ent.Tx) error {
		if nodeIdx >= 0 {
			if err := s.taskRepo.CancelPendingByInstanceNodeWithTx(ctx, tx, tid, instanceID, nodeIdx, taskID); err != nil {
				return err
			}
		}
		if err := s.instanceRepo.UpdateStatusWithTx(ctx, tx, instanceID, tid, oaV1.WorkflowInstance_REJECTED.Enum(), nil); err != nil {
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
	ctx context.Context, tid, uid uint32, taskID, instanceID uint32, currentNodeIndex *int, forwardTo uint32,
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
	nodeIdx := -1
	if currentNodeIndex != nil {
		nodeIdx = *currentNodeIndex
	}
	ni := nodeIdx
	if err := s.instanceRepo.Txn(ctx, func(tx *ent.Tx) error {
		if err := s.taskRepo.UpdateAssigneeWithTx(ctx, tx, taskID, tid, forwardTo); err != nil {
			return err
		}
		if _, err := s.logRepo.CreateWithTx(ctx, tx, tid, uid, instanceID, ni, oaV1.WorkflowLog_FORWARD.Enum(), ""); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	s.notifyManyAsync(ctx, []uint32{forwardTo}, "您有被转办的任务", "请登录系统查看待审批事项")
	return &emptypb.Empty{}, nil
}

// advanceInstance 当前节点已全部通过，推进到下一节点。
func (s *WorkflowService) advanceInstance(
	ctx context.Context, tid, uid uint32, instanceID uint32, nodeIdx int,
) (*emptypb.Empty, error) {
	if err := s.launchFromNode(ctx, tid, uid, instanceID, nodeIdx+1); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// launchFromNode 从指定节点启动流程：解析审批人并排除申请人本人（申请人对自己的
// 申请视为自动同意）；全部审批人都是申请人的节点整节点自动通过（写 APPROVE 日志
// 留痕）并继续推进，直到出现有效审批人的节点（建任务+通知）或越界终结 APPROVED。
// 推进路径的跨 repo 多步写（instance 状态 + task 创建 + 日志）经 WorkflowInstanceRepo.Txn
// 包入单一事务原子提交；事务内任一步失败则整体回滚，返回错误。单步写路径不包事务。
func (s *WorkflowService) launchFromNode(
	ctx context.Context, tid, uid uint32, instanceID uint32, startIdx int,
) error {
	applicant, businessType, businessID, err := s.instanceRepo.GetMeta(ctx, instanceID, tid)
	if err != nil {
		return err
	}
	nodes, err := s.nodesOfInstance(ctx, instanceID, tid)
	if err != nil {
		return err
	}

	for idx := startIdx; ; idx++ {
		if idx >= len(nodes) {
			// 终结：实例 APPROVED，清空指针。
			if err := s.instanceRepo.UpdateStatus(ctx, instanceID, tid, oaV1.WorkflowInstance_APPROVED.Enum(), nil); err != nil {
				return err
			}
			s.notifyManyAsync(ctx, []uint32{applicant}, "您的申请已通过", "全部审批节点已通过")
			s.fireBusinessEvent(ctx, tid, instanceID, businessID, businessType, oaV1.WorkflowInstance_APPROVED)
			return nil
		}

		approvers, err := s.resolveApprovers(ctx, tid, applicant, nodes[idx])
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
			// 申请人是该节点唯一审批人：自动通过，推进下一节点。
			_, _ = s.logRepo.Create(ctx, tid, uid, instanceID, idx, oaV1.WorkflowLog_APPROVE.Enum(), "审批人为申请人本人，自动通过")
			continue
		}

		ni := idx
		if err := s.instanceRepo.Txn(ctx, func(tx *ent.Tx) error {
			if err := s.instanceRepo.UpdateStatusWithTx(ctx, tx, instanceID, tid, oaV1.WorkflowInstance_PENDING.Enum(), &ni); err != nil {
				return err
			}
			for _, approver := range others {
				if _, err := s.taskRepo.CreateWithTx(ctx, tx, tid, uid, instanceID, idx, approver); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		s.notifyManyAsync(ctx, others, "您有新的待办任务", "请登录系统查看待审批事项")
		return nil
	}
}

// ===================== 撤回 =====================

// WithdrawApply 申请人撤回自己的进行中申请：实例转 WITHDRAWN，全部待办任务取消，
// 写 WITHDRAW 日志，通知原待办审批人。
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
		if err := s.instanceRepo.UpdateStatusWithTx(ctx, tx, instanceID, tid, oaV1.WorkflowInstance_WITHDRAWN.Enum(), nil); err != nil {
			return err
		}
		if _, err := s.logRepo.CreateWithTx(ctx, tx, tid, uid, instanceID, -1, oaV1.WorkflowLog_WITHDRAW.Enum(), ""); err != nil {
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

// parseWorkflowNodes 解析 node_config JSON 文本为节点数组。
func parseWorkflowNodes(nodeConfig string) ([]workflowNode, error) {
	if nodeConfig == "" {
		return nil, oaV1.ErrorBadRequest("empty node config")
	}
	var nodes []workflowNode
	if err := json.Unmarshal([]byte(nodeConfig), &nodes); err != nil {
		return nil, oaV1.ErrorBadRequest("invalid node config json")
	}
	return nodes, nil
}

// nodesOfInstance 读取实例所属定义并解析节点配置。
func (s *WorkflowService) nodesOfInstance(ctx context.Context, instanceID, tenantID uint32) ([]workflowNode, error) {
	nodeConfig, err := s.instanceRepo.GetDefinitionNodeConfig(ctx, instanceID, tenantID)
	if err != nil {
		return nil, err
	}
	return parseWorkflowNodes(nodeConfig)
}

// resolveApprovers 解析节点审批人列表。USER 直取；LEADER 解析申请人主组织负责人；
// POSITION 解析职位在职持有者（可展开多人）。结果去重且保序。
func (s *WorkflowService) resolveApprovers(
	ctx context.Context, tenantID, applicantUserID uint32, node workflowNode,
) ([]uint32, error) {
	specs := node.normalizedApprovers()
	if len(specs) == 0 {
		return nil, oaV1.ErrorBadRequest("node has no approver")
	}

	seen := make(map[uint32]struct{}, len(specs))
	approvers := make([]uint32, 0, len(specs))
	appendUserID := func(id uint32) {
		if id == 0 {
			return
		}
		if _, dup := seen[id]; dup {
			return
		}
		seen[id] = struct{}{}
		approvers = append(approvers, id)
	}

	for _, spec := range specs {
		switch spec.Type {
		case approverTypeUser:
			appendUserID(spec.ID)
		case approverTypeLeader:
			leaderID, err := s.resolverRepo.ResolveOrgLeader(ctx, tenantID, applicantUserID)
			if err != nil {
				return nil, err
			}
			appendUserID(leaderID)
		case approverTypePosition:
			holders, err := s.resolverRepo.ResolvePositionHolders(ctx, tenantID, spec.ID)
			if err != nil {
				return nil, err
			}
			for _, holder := range holders {
				appendUserID(holder)
			}
		default:
			return nil, oaV1.ErrorBadRequest("unsupported approver type")
		}
	}
	if len(approvers) == 0 {
		return nil, oaV1.ErrorBadRequest("node has no approver")
	}
	return approvers, nil
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
