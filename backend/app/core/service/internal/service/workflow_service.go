package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/tx7do/go-wind-flow/flow"

	"github.com/tx7do/go-crud/viewer"

	"go-wind-oa/app/core/service/internal/data"
	"go-wind-oa/app/core/service/internal/data/ent/workflowlog"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// WorkflowService OA 工作流服务。引擎内核由 go-wind-flow 承担（见
// workflow_engine.go 的装配与 SPI 适配），本文件只做 RPC 契约适配：
// 定义/委托 CRUD 与列表查询直读仓储，状态迁移（提交/审批/撤回）经引擎。
type WorkflowService struct {
	oaV1.UnimplementedWorkflowServiceServer

	log *log.Helper

	definitionRepo *data.WorkflowDefinitionRepo
	instanceRepo   *data.WorkflowInstanceRepo
	taskRepo       *data.WorkflowTaskRepo
	logRepo        *data.WorkflowLogRepo
	resolverRepo   *data.WorkflowResolverRepo
	delegationRepo *data.WorkflowDelegationRepo

	flowStore *data.FlowStore
	engine    *flow.Engine
	notifier  *WorkflowNotifier
}

func NewWorkflowService(
	ctx *bootstrap.Context,
	definitionRepo *data.WorkflowDefinitionRepo,
	instanceRepo *data.WorkflowInstanceRepo,
	taskRepo *data.WorkflowTaskRepo,
	logRepo *data.WorkflowLogRepo,
	resolverRepo *data.WorkflowResolverRepo,
	delegationRepo *data.WorkflowDelegationRepo,
	flowStore *data.FlowStore,
	engine *flow.Engine,
	notifier *WorkflowNotifier,
) *WorkflowService {
	return &WorkflowService{
		log:            ctx.NewLoggerHelper("workflow/service/core-service"),
		definitionRepo: definitionRepo,
		instanceRepo:   instanceRepo,
		taskRepo:       taskRepo,
		logRepo:        logRepo,
		resolverRepo:   resolverRepo,
		delegationRepo: delegationRepo,
		flowStore:      flowStore,
		engine:         engine,
		notifier:       notifier,
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

// mapFlowError 把引擎哨兵错误映射为 gRPC 业务错误。
func (s *WorkflowService) mapFlowError(err error) error {
	if err == nil {
		return nil
	}
	s.log.Errorf("workflow engine error: %s", err.Error())
	switch {
	case errors.Is(err, flow.ErrNotFound):
		return oaV1.ErrorNotFound("workflow object not found")
	case errors.Is(err, flow.ErrInvalidState):
		return oaV1.ErrorConflict("invalid workflow state")
	case errors.Is(err, flow.ErrForbidden):
		return oaV1.ErrorForbidden("not your pending task")
	case errors.Is(err, flow.ErrInvalidDefinition),
		errors.Is(err, flow.ErrExpression),
		errors.Is(err, flow.ErrScript):
		return oaV1.ErrorBadRequest("invalid workflow operation: %s", err.Error())
	default:
		return err
	}
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

	// 1. 取定义，校验 ENABLED（定义状态门禁在业务侧，引擎只认已存在的定义）。
	def, err := s.definitionRepo.GetByCodeVersion(ctx, req.GetCode(), req.GetVersion())
	if err != nil {
		return nil, err
	}
	if def.DefinitionStatus == nil || *def.DefinitionStatus != oaV1.WorkflowDefinition_ENABLED {
		return nil, oaV1.ErrorBadRequest("definition not enabled")
	}

	// 2. 表单数据 → 实例变量（网关条件与动态办理人共享该命名空间）。
	vars, err := decodeFormVars(req.GetFormData())
	if err != nil {
		return nil, oaV1.ErrorBadRequest("form data is not valid json")
	}

	// 3. 引擎启动实例：实例/令牌/任务/日志一次事务落库，推进到首个阻塞点。
	var instanceID uint32
	err = s.flowStore.WithTx(ctx, func(ctx context.Context) error {
		inst, _, err := s.engine.Start(ctx, flow.StartRequest{
			TenantID:     data.FlowIDOf(tid),
			Key:          req.GetCode(),
			Version:      int(req.GetVersion()),
			BusinessKey:  data.FlowIDOf(req.GetBusinessId()),
			BusinessType: req.GetBusinessType(),
			StarterID:    data.FlowIDOf(uid),
			Variables:    vars,
		})
		if err != nil {
			return err
		}
		instanceID = data.FromFlowIDOrZero(inst.ID)
		return nil
	})
	if err != nil {
		return nil, s.mapFlowError(err)
	}

	return &oaV1.SubmitApplyResponse{InstanceId: instanceID}, nil
}

// decodeFormVars 表单 JSON → 实例变量环境。
func decodeFormVars(formData string) (flow.Vars, error) {
	if formData == "" {
		return flow.Vars{}, nil
	}
	m := map[string]any{}
	if err := json.Unmarshal([]byte(formData), &m); err != nil {
		return nil, err
	}
	return flow.Vars(m), nil
}

// ===================== 审批 =====================

func (s *WorkflowService) AuditTask(ctx context.Context, req *oaV1.AuditTaskRequest) (*emptypb.Empty, error) {
	tid, uid, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}

	switch req.GetAction() {
	case oaV1.AuditAction_APPROVE, oaV1.AuditAction_REJECT:
		outcome := flow.OutcomeApprove
		if req.GetAction() == oaV1.AuditAction_REJECT {
			outcome = flow.OutcomeReject
		}
		err := s.flowStore.WithTx(ctx, func(ctx context.Context) error {
			return s.engine.CompleteTask(ctx, flow.CompleteTaskRequest{
				TaskID:  data.FlowIDOf(req.GetTaskId()),
				ActorID: data.FlowIDOf(uid),
				Outcome: outcome,
				Comment: req.GetComment(),
			})
		})
		return &emptypb.Empty{}, s.mapFlowError(err)

	case oaV1.AuditAction_FORWARD:
		forwardTo := req.GetForwardTo()
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
		// 取任务状态供日志（assignee/待办校验引擎会再做一次，这里先给出旧文案）。
		assignee, pending, fwInstanceID, nodeID, err := s.taskRepo.GetState(ctx, req.GetTaskId(), tid)
		if err != nil {
			return nil, err
		}
		if assignee != uid || !pending {
			return nil, oaV1.ErrorForbidden("not your pending task")
		}
		err = s.flowStore.WithTx(ctx, func(ctx context.Context) error {
			if ferr := s.engine.ForwardTask(ctx, data.FlowIDOf(req.GetTaskId()), data.FlowIDOf(uid), data.FlowIDOf(forwardTo)); ferr != nil {
				return ferr
			}
			return s.flowStore.AppendAuditLog(ctx, tid, uid, fwInstanceID, nodeID, workflowlog.LogActionForward, "")
		})
		if err != nil {
			return nil, s.mapFlowError(err)
		}
		s.notifier.NotifyManyAsync(ctx, []uint32{forwardTo}, "您有被转办的任务", "请登录系统查看待审批事项")
		return &emptypb.Empty{}, nil

	case oaV1.AuditAction_ADD_APPROVER:
		additional := req.GetAdditionalApprover()
		if additional == 0 || additional == uid {
			return nil, oaV1.ErrorBadRequest("invalid additional approver")
		}
		active, err := s.resolverRepo.UserIsActive(ctx, tid, additional)
		if err != nil {
			return nil, err
		}
		if !active {
			return nil, oaV1.ErrorBadRequest("additional approver user is unavailable")
		}
		_, pending, instanceID, nodeID, err := s.taskRepo.GetState(ctx, req.GetTaskId(), tid)
		if err != nil {
			return nil, err
		}
		if !pending {
			return nil, oaV1.ErrorForbidden("not your pending task")
		}
		var newAssignees []uint32
		err = s.flowStore.WithTx(ctx, func(ctx context.Context) error {
			created, aerr := s.engine.AddTaskApprovers(
				ctx, data.FlowIDOf(req.GetTaskId()), data.FlowIDOf(uid),
				[]string{data.FlowIDOf(additional)},
			)
			if aerr != nil {
				return aerr
			}
			// 加签日志（与旧引擎一致：FORWARD 动作 + 固定文案）。
			if lerr := s.flowStore.AppendAuditLog(ctx, tid, uid, instanceID, nodeID, workflowlog.LogActionForward, "加签：新增审批人"); lerr != nil {
				return lerr
			}
			for _, t := range created {
				if uid2 := data.FromFlowIDOrZero(t.Assignee); uid2 != 0 {
					newAssignees = append(newAssignees, uid2)
				}
			}
			return nil
		})
		if err != nil {
			return nil, s.mapFlowError(err)
		}
		if len(newAssignees) > 0 {
			s.notifier.NotifyManyAsync(ctx, newAssignees, "您有被加签的任务", "请登录系统查看待审批事项")
		}
		return &emptypb.Empty{}, nil

	default:
		return nil, oaV1.ErrorBadRequest("invalid audit action")
	}
}

// ===================== 撤回 =====================

// WithdrawApply 申请人撤回自己的进行中申请。引擎终结实例（WITHDRAWN）并
// 级联取消全部待办/令牌；对原待办人的通知在终结前取列表后补发。
func (s *WorkflowService) WithdrawApply(ctx context.Context, req *oaV1.WithdrawApplyRequest) (*emptypb.Empty, error) {
	tid, uid, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	instanceID := req.GetInstanceId()
	if instanceID == 0 {
		return nil, oaV1.ErrorBadRequest("invalid parameter")
	}

	// 仅申请人本人可撤回（引擎校验 StarterID）；仅进行中实例可撤回（引擎校验 Live）。
	pendingAssignees, err := s.taskRepo.ListPendingAssigneesByInstance(ctx, tid, instanceID)
	if err != nil {
		return nil, err
	}

	err = s.flowStore.WithTx(ctx, func(ctx context.Context) error {
		if werr := s.engine.Withdraw(ctx, data.FlowIDOf(instanceID), data.FlowIDOf(uid)); werr != nil {
			return werr
		}
		// WITHDRAW 日志（引擎的 process.ended 事件在 History 适配层不落日志）。
		return s.flowStore.AppendAuditLog(ctx, tid, uid, instanceID, "", workflowlog.LogActionWithdraw, "")
	})
	if err != nil {
		return nil, s.mapFlowError(err)
	}

	s.notifier.NotifyManyAsync(ctx, pendingAssignees, "待审批申请已被撤回", "申请人已撤回该申请，无需继续处理")
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
