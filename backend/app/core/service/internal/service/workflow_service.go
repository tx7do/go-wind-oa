package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/tx7do/go-crud/viewer"

	internalMessageV1 "go-wind-oa/api/gen/go/internal_message/service/v1"

	oav1 "go-wind-oa/api/gen/go/oa/service/v1"
	"go-wind-oa/app/core/service/internal/data"
)

// WorkflowService OA 工作流审批服务的状态机驱动层。
//
// core-service 内单层 gRPC 服务：实现 kratos 生成的 WorkflowServiceServer，
// 持有本模块的 ent 仓库与同进程的 InternalMessageService。审批流转时经后者
// 投递待办/结果通知，复用 cms 既有的站内信通道，不在 OA 内重造。
//
// 状态机为线性模型：实例任一时刻至多一条 PENDING 任务；审批通过则推进到定义中的
// 下一节点并派生新任务，无下一节点则终结为 APPROVED；驳回则终结为 REJECTED；
// 转办则改写当前任务指派人、节点指针不变。每次迁移落一条 WorkflowLog。
type WorkflowService struct {
	oav1.WorkflowServiceServer
	log *log.Helper

	definitionRepo *data.WorkflowDefinitionRepo
	instanceRepo   *data.WorkflowInstanceRepo
	taskRepo       *data.WorkflowTaskRepo
	logRepo        *data.WorkflowLogRepo

	// notificationService 同进程站内信服务。审批流转时据此异步通知下一审批人 /
	// 申请人，复用既有消息通道，不在 OA 内重造。
	notificationService *InternalMessageService
}

func NewWorkflowService(
	ctx *bootstrap.Context,
	definitionRepo *data.WorkflowDefinitionRepo,
	instanceRepo *data.WorkflowInstanceRepo,
	taskRepo *data.WorkflowTaskRepo,
	logRepo *data.WorkflowLogRepo,
	notificationService *InternalMessageService,
) *WorkflowService {
	return &WorkflowService{
		log:                 ctx.NewLoggerHelper("workflow/service/core-service"),
		definitionRepo:      definitionRepo,
		instanceRepo:        instanceRepo,
		taskRepo:            taskRepo,
		logRepo:             logRepo,
		notificationService: notificationService,
	}
}

// ---- 上下文 / 通用 helper ----

// callerFromContext 从 viewer context 取调用人 (tenantID, userID)。
// OA viewer 中间件把 JWT 鉴权身份翻译为 viewer context，二者缺一即 fail-closed。
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

// ptr 取任意标量 / 枚举值的指针，用于填充 proto optional 字段。
func ptr[T any](v T) *T { return &v }

// notifyAsync 经同进程站内信服务异步通知某用户。
// 脱离请求 ctx（context.WithoutCancel）+ recover，fire-and-forget，不阻塞状态机。
// viewer context 作为 context value 跨 WithoutCancel 保留，站内信落库时由
// TenantPrivacy 按调用人租户做行级隔离。
func (s *WorkflowService) notifyAsync(ctx context.Context, recipientUserId uint32, title, content string) {
	if recipientUserId == 0 || s.notificationService == nil {
		return
	}
	go func() {
		defer func() { _ = recover() }()
		notifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_, _ = s.notificationService.SendMessage(notifyCtx, &internalMessageV1.SendMessageRequest{
			Type:          internalMessageV1.InternalMessage_NOTIFICATION,
			Title:         &title,
			Content:       content,
			TargetUserIds: []uint32{recipientUserId},
		})
	}()
}

// ---- 节点解析 ----

// workflowNode 定义中单个审批节点的最小结构。
// v1 仅支持 approver_type=="USER"（指定用户ID）。角色 / 部门主管等指派策略
// 需对接 identity 服务，留作后续扩展。
type workflowNode struct {
	ApproverType   string `json:"approver_type"`
	ApproverUserID uint32 `json:"approver_user_id"`
}

func parseWorkflowNodes(nodeConfigJSON string) ([]workflowNode, error) {
	if strings.TrimSpace(nodeConfigJSON) == "" {
		return nil, fmt.Errorf("empty node config")
	}
	var nodes []workflowNode
	if err := json.Unmarshal([]byte(nodeConfigJSON), &nodes); err != nil {
		return nil, fmt.Errorf("invalid node config: %w", err)
	}
	return nodes, nil
}

func resolveApprover(n workflowNode) (uint32, error) {
	if strings.ToUpper(n.ApproverType) != "USER" || n.ApproverUserID == 0 {
		return 0, fmt.Errorf("unsupported approver in node")
	}
	return n.ApproverUserID, nil
}

// ---- 管理端：定义 ----

func (s *WorkflowService) CreateWorkflowDefinition(ctx context.Context, req *oav1.CreateWorkflowDefinitionRequest) (*oav1.WorkflowDefinition, error) {
	if req == nil || req.Data == nil {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	tenantID, userID, ok := callerFromContext(ctx)
	if !ok {
		return nil, oav1.ErrorBadRequest("missing viewer context")
	}
	// 新建定义一律落 DRAFT；启用/禁用经 UpdateWorkflowDefinition 切换。
	req.Data.TenantId = ptr(tenantID)
	req.Data.CreatedBy = ptr(userID)
	req.Data.DefinitionStatus = ptr(oav1.WorkflowDefinition_DRAFT)
	return s.definitionRepo.Create(ctx, req.Data)
}

func (s *WorkflowService) ListWorkflowDefinition(ctx context.Context, req *oav1.ListWorkflowDefinitionRequest) (*oav1.ListWorkflowDefinitionResponse, error) {
	if req == nil || req.Paging == nil {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	if _, _, ok := callerFromContext(ctx); !ok {
		return nil, oav1.ErrorBadRequest("missing viewer context")
	}
	return s.definitionRepo.List(ctx, req.Paging)
}

// UpdateWorkflowDefinition 切换定义状态（启用/禁用）。
//
// 仅允许 update_mask 含 "definition_status"——其余字段路径一律拒绝，
// 防止经此接口篡改 node_config / form_schema 等业务字段。状态转换合法性：
//   - DRAFT → ENABLED：启用，SubmitApply 后续可对 ENABLED 定义提交申请；
//   - ENABLED → DISABLED：禁用，已存在的 PENDING 实例不受影响，但新申请被拒；
//   - DISABLED → ENABLED：重新启用。
//
// 调用者须为同租户成员（tenant 由 TenantPrivacy + 仓库层 TenantIDEQ 双重隔离）。
func (s *WorkflowService) UpdateWorkflowDefinition(ctx context.Context, req *oav1.UpdateWorkflowDefinitionRequest) (*oav1.WorkflowDefinition, error) {
	if req == nil || req.Id == 0 {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	if _, _, ok := callerFromContext(ctx); !ok {
		return nil, oav1.ErrorBadRequest("missing viewer context")
	}
	// update_mask 严格限定为 definition_status，且须存在该字段路径。
	mask := req.GetUpdateMask()
	if mask == nil || len(mask.GetPaths()) != 1 || mask.GetPaths()[0] != "definition_status" {
		return nil, oav1.ErrorBadRequest("update_mask must contain only definition_status")
	}
	if req.Data == nil || req.Data.DefinitionStatus == nil {
		return nil, oav1.ErrorBadRequest("missing definition_status")
	}
	return s.definitionRepo.UpdateStatus(ctx, req.Id, *req.Data.DefinitionStatus)
}

// GetWorkflowDefinition 取单个定义详情（含 node_config / form_schema）。
// 调用者须为同租户成员（tenant 由 TenantPrivacy 隔离）。
func (s *WorkflowService) GetWorkflowDefinition(ctx context.Context, req *oav1.GetWorkflowDefinitionRequest) (*oav1.WorkflowDefinition, error) {
	if req == nil || req.Id == 0 {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	if _, _, ok := callerFromContext(ctx); !ok {
		return nil, oav1.ErrorBadRequest("missing viewer context")
	}
	return s.definitionRepo.GetByID(ctx, req.Id)
}

// ---- 业务端：提交申请 ----

func (s *WorkflowService) SubmitApply(ctx context.Context, req *oav1.SubmitApplyRequest) (*oav1.SubmitApplyResponse, error) {
	if req == nil || req.DefinitionCode == "" || req.Title == "" {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	tenantID, userID, ok := callerFromContext(ctx)
	if !ok {
		return nil, oav1.ErrorBadRequest("missing viewer context")
	}

	// 1) 取定义，校验已启用，解析首节点。
	def, err := s.definitionRepo.GetByCodeVersion(ctx, req.DefinitionCode, req.DefinitionVersion)
	if err != nil {
		return nil, err
	}
	if def.GetDefinitionStatus() != oav1.WorkflowDefinition_ENABLED {
		return nil, oav1.ErrorBadRequest("workflow definition not enabled")
	}
	nodes, err := parseWorkflowNodes(def.GetNodeConfig())
	if err != nil || len(nodes) == 0 {
		return nil, oav1.ErrorBadRequest("invalid workflow node config")
	}
	firstAssignee, err := resolveApprover(nodes[0])
	if err != nil {
		return nil, oav1.ErrorBadRequest("%s", err.Error())
	}

	// 2) 创建实例（PENDING，指针 = 0）。申请表单数据以 JSON 文本透传落盘。
	instReq := &oav1.WorkflowInstance{
		Title:            ptr(req.Title),
		FormData:         ptr(req.FormData),
		InstanceStatus:   ptr(oav1.WorkflowInstance_PENDING),
		CurrentNodeIndex: ptr(int32(0)),
		TenantId:         ptr(tenantID),
		CreatedBy:        ptr(userID),
		DefinitionId:     def.Id,
	}
	inst, err := s.instanceRepo.Create(ctx, instReq)
	if err != nil {
		return nil, err
	}

	// 3) 派生首条待办任务（node 0 → firstAssignee）。
	taskReq := &oav1.WorkflowTask{
		InstanceId:     inst.Id,
		NodeIndex:      ptr(int32(0)),
		AssigneeUserId: ptr(firstAssignee),
		TaskStatus:     ptr(oav1.WorkflowTask_PENDING),
		TenantId:       ptr(tenantID),
		CreatedBy:      ptr(userID),
	}
	if _, err := s.taskRepo.Create(ctx, taskReq); err != nil {
		return nil, err
	}

	// 4) 审计日志：SUBMIT。
	if _, err := s.logRepo.Create(ctx, &oav1.WorkflowLog{
		InstanceId: inst.Id,
		NodeIndex:  ptr(int32(0)),
		LogAction:  ptr(oav1.WorkflowLog_SUBMIT),
		TenantId:   ptr(tenantID),
		CreatedBy:  ptr(userID),
	}); err != nil {
		return nil, err
	}

	// 5) 异步通知首节点审批人。
	s.notifyAsync(ctx, firstAssignee, "您有新的审批待办", "您有一条新的工作流审批待办，请及时处理。")

	return &oav1.SubmitApplyResponse{InstanceId: inst.GetId()}, nil
}

// ---- 业务端：审批 / 驳回 / 转办 ----

func (s *WorkflowService) AuditTask(ctx context.Context, req *oav1.AuditTaskRequest) (*emptypb.Empty, error) {
	if req == nil || req.TaskId == 0 {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	_, userID, ok := callerFromContext(ctx)
	if !ok {
		return nil, oav1.ErrorBadRequest("missing viewer context")
	}

	// 仅当前指派审批人可处置其处于 PENDING 的任务。
	assignee, taskStatus, instanceID, nodeIndex, err := s.taskRepo.GetState(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}
	if assignee != userID || taskStatus != 0 {
		return nil, oav1.ErrorForbidden("not your pending task")
	}

	// 实例必须仍处于活跃态。currentNodeIndex 此处无需（推进逻辑在 handleApprove 内重新读取）。
	instStatus, _, applicant, err := s.instanceRepo.GetState(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if instStatus != 0 {
		return nil, oav1.ErrorBadRequest("instance not active")
	}
	_ = nodeIndex

	switch req.Action {
	case oav1.AuditTaskRequest_APPROVE:
		return s.handleApprove(ctx, req.TaskId, instanceID, applicant, req.Comment)
	case oav1.AuditTaskRequest_REJECT:
		return s.handleReject(ctx, req.TaskId, instanceID, applicant, req.Comment)
	case oav1.AuditTaskRequest_FORWARD:
		return s.handleForward(ctx, req.TaskId, instanceID, req.Comment, req.ForwardToUserId, userID)
	default:
		return nil, oav1.ErrorBadRequest("invalid audit action")
	}
}

// handleApprove 审批通过：关闭任务、落 APPROVE 日志，按定义节点序列推进或终结。
func (s *WorkflowService) handleApprove(ctx context.Context, taskID, instanceID, applicant uint32, comment *string) (*emptypb.Empty, error) {
	tenantID, userID, ok := callerFromContext(ctx)
	if !ok {
		return nil, oav1.ErrorBadRequest("missing viewer context")
	}
	if err := s.taskRepo.UpdateStatus(ctx, taskID, oav1.WorkflowTask_APPROVED); err != nil {
		return nil, err
	}

	// 审计日志：APPROVE。
	if _, err := s.logRepo.Create(ctx, &oav1.WorkflowLog{
		InstanceId: ptr(instanceID),
		LogAction:  ptr(oav1.WorkflowLog_APPROVE),
		Comment:    comment,
		TenantId:   ptr(tenantID),
		CreatedBy:  ptr(userID),
	}); err != nil {
		return nil, err
	}

	// 取实例当前指针 + 定义节点序列。applicant 已由参数传入，此处不再重复读取。
	_, currentNodeIndex, _, err := s.instanceRepo.GetState(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	nodeConfigJSON, err := s.instanceRepo.GetDefinitionNodeConfig(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	nodes, err := parseWorkflowNodes(nodeConfigJSON)
	if err != nil {
		return nil, oav1.ErrorInternalError("broken workflow node config")
	}

	nextIndex := currentNodeIndex + 1
	if int(nextIndex) >= len(nodes) {
		// 终结：APPROVED，清空节点指针，通知申请人。
		if err := s.instanceRepo.UpdateStatus(ctx, instanceID, oav1.WorkflowInstance_APPROVED, nil); err != nil {
			return nil, err
		}
		s.notifyAsync(ctx, applicant, "您的申请已通过", "您的工作流申请已审批通过。")
		return &emptypb.Empty{}, nil
	}

	// 推进：派生下一节点任务，通知下一审批人。
	nextAssignee, err := resolveApprover(nodes[nextIndex])
	if err != nil {
		return nil, oav1.ErrorBadRequest("%s", err.Error())
	}
	ni := nextIndex
	if err := s.instanceRepo.UpdateStatus(ctx, instanceID, oav1.WorkflowInstance_PENDING, &ni); err != nil {
		return nil, err
	}
	if _, err := s.taskRepo.Create(ctx, &oav1.WorkflowTask{
		InstanceId:     ptr(instanceID),
		NodeIndex:      ptr(nextIndex),
		AssigneeUserId: ptr(nextAssignee),
		TaskStatus:     ptr(oav1.WorkflowTask_PENDING),
		TenantId:       ptr(tenantID),
		CreatedBy:      ptr(userID),
	}); err != nil {
		return nil, err
	}
	s.notifyAsync(ctx, nextAssignee, "您有新的审批待办", "您有一条新的工作流审批待办，请及时处理。")
	return &emptypb.Empty{}, nil
}

// handleReject 驳回：关闭任务、落 REJECT 日志、实例终结为 REJECTED，通知申请人。
func (s *WorkflowService) handleReject(ctx context.Context, taskID, instanceID, applicant uint32, comment *string) (*emptypb.Empty, error) {
	tenantID, userID, ok := callerFromContext(ctx)
	if !ok {
		return nil, oav1.ErrorBadRequest("missing viewer context")
	}
	// 关闭当前任务 + 实例终结为 REJECTED。
	if err := s.taskRepo.UpdateStatus(ctx, taskID, oav1.WorkflowTask_REJECTED); err != nil {
		return nil, err
	}
	if err := s.instanceRepo.UpdateStatus(ctx, instanceID, oav1.WorkflowInstance_REJECTED, nil); err != nil {
		return nil, err
	}
	// 审计日志：REJECT。
	if _, err := s.logRepo.Create(ctx, &oav1.WorkflowLog{
		InstanceId: ptr(instanceID),
		LogAction:  ptr(oav1.WorkflowLog_REJECT),
		Comment:    comment,
		TenantId:   ptr(tenantID),
		CreatedBy:  ptr(userID),
	}); err != nil {
		return nil, err
	}
	// 通知申请人其申请已被驳回。
	s.notifyAsync(ctx, applicant, "您的申请已被驳回", "您的工作流申请已被审批人驳回。")
	return &emptypb.Empty{}, nil
}

// handleForward 转办：改写当前任务指派人、落 FORWARD 日志，节点指针不变，通知被转办人。
func (s *WorkflowService) handleForward(ctx context.Context, taskID, instanceID uint32, comment *string, forwardTo *uint32, currentUser uint32) (*emptypb.Empty, error) {
	tenantID, userID, ok := callerFromContext(ctx)
	if !ok {
		return nil, oav1.ErrorBadRequest("missing viewer context")
	}
	if forwardTo == nil || *forwardTo == 0 || *forwardTo == currentUser {
		return nil, oav1.ErrorBadRequest("invalid forward target")
	}
	if err := s.taskRepo.UpdateAssignee(ctx, taskID, *forwardTo); err != nil {
		return nil, err
	}
	if _, err := s.logRepo.Create(ctx, &oav1.WorkflowLog{
		InstanceId: ptr(instanceID),
		LogAction:  ptr(oav1.WorkflowLog_FORWARD),
		Comment:    comment,
		TenantId:   ptr(tenantID),
		CreatedBy:  ptr(userID),
	}); err != nil {
		return nil, err
	}
	s.notifyAsync(ctx, *forwardTo, "您有转办的审批待办", "有一条审批任务被转办给您，请及时处理。")
	return &emptypb.Empty{}, nil
}

// ---- 业务端：我的任务视图 ----

func (s *WorkflowService) GetMyTasks(ctx context.Context, req *oav1.GetMyTasksRequest) (*oav1.GetMyTasksResponse, error) {
	if req == nil || req.Paging == nil {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	_, userID, ok := callerFromContext(ctx)
	if !ok {
		return nil, oav1.ErrorBadRequest("missing viewer context")
	}
	switch req.ListType {
	case oav1.GetMyTasksRequest_PENDING:
		return s.taskRepo.ListPendingByAssignee(ctx, userID, req.Paging)
	case oav1.GetMyTasksRequest_DONE:
		return s.logRepo.ListByActor(ctx, userID, req.Paging)
	case oav1.GetMyTasksRequest_SUBMITTED:
		return s.instanceRepo.ListByCreator(ctx, userID, req.Paging)
	default:
		return nil, oav1.ErrorBadRequest("invalid list type")
	}
}

// GetTask 单任务详情：申请标题、申请表单数据、该实例的审批日志轨迹。
//
// 授权口径与 AuditTask 完全一致——调用 GetState 取 assignee / taskStatus，
// 校验 assignee == caller 且 task 处于 PENDING，否则 Forbidden。此为
// 纵深防御：GetDetailByAssignee 在 DB 层以 IDEQ + AssigneeUserIDEQ +
// TaskStatusEQ(PENDING) 三重谓词同样拒绝非归属任务，任一层失败即拒。
// tenant 由 TenantPrivacy 策略按 viewer 自动隔离。
//
// 授权通过后，投影分两步：
//  1. taskRepo.GetDetailByAssignee 投影 task + 关联实例的 Title / FormData
//     （FormData 按 GetDefinitionNodeConfig 同款 any→string 经 json.Marshal 落回）；
//  2. logRepo.ListByInstance 投影该实例的审批日志轨迹（APPROVE/REJECT/FORWARD，
//     与 ListByActor 同款口径，SUBMIT 排除）。
//
// 其余字段（definition_id / current_node_index / tenant_id 等内部字段）不投影，
// 对齐 MyTaskItem 的最小披露原则。
func (s *WorkflowService) GetTask(ctx context.Context, req *oav1.GetTaskRequest) (*oav1.GetTaskResponse, error) {
	if req == nil || req.TaskId == 0 {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	_, userID, ok := callerFromContext(ctx)
	if !ok {
		return nil, oav1.ErrorBadRequest("missing viewer context")
	}

	// 仅当前指派审批人可查其处于 PENDING 的任务（与 AuditTask 同款授权）。
	assignee, taskStatus, _, _, err := s.taskRepo.GetState(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}
	if assignee != userID || taskStatus != 0 {
		return nil, oav1.ErrorForbidden("not your pending task")
	}

	resp, err := s.taskRepo.GetDetailByAssignee(ctx, req.TaskId, userID)
	if err != nil {
		return nil, err
	}

	instanceID := resp.GetInstanceId()
	if instanceID != 0 {
		history, hErr := s.logRepo.ListByInstance(ctx, instanceID)
		if hErr != nil {
			return nil, hErr
		}
		resp.History = history
	}
	return resp, nil
}
