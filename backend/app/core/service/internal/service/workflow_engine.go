package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"github.com/tx7do/go-wind-flow/contrib/exprlang"
	"github.com/tx7do/go-wind-flow/flow"
	"github.com/tx7do/go-wind-flow/format/windjson"

	"github.com/tx7do/go-utils/trans"

	internalMessageV1 "go-wind-oa/api/gen/go/internal_message/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
	"go-wind-oa/app/core/service/internal/data"
)

// ===================== 引擎装配 =====================
//
// 工作流引擎内核自本文件起由 github.com/tx7do/go-wind-flow 库承担（Token
// 状态机、图遍历、网关/子流程/超时语义、表达式求值）。本文件只做三件事：
//
//  1. 把引擎装配起来：ent 存储适配（FlowStore）+ 身份解析（本组织负责人/
//     职位持有人/委托代办）+ expr-lang 求值器 + 监听器；
//  2. 实现 flow.IdentityProvider（旧引擎的 workflow_resolver_repo 寻人 +
//     applyDelegation 委托叠加在此对位）；
//  3. 实现 flow.Listener（旧引擎的「申请人自审自动通过」「待办站内信通知」
//     「业务单据终结回调」三个副作用在此对位）。
//
// 申请人剔除语义对齐旧引擎：解析出的办理人剔除申请人本人；剔除后为空则
// 保留申请人（监听器检测到申请人的待办任务后立即代其通过，日志与旧引擎
// 一致：审批人为申请人本人，自动通过）。

// WorkflowNotifier 站内信通知出口（引擎监听器与服务层共用）。
type WorkflowNotifier struct {
	log *log.Helper
	svc *InternalMessageService
}

func NewWorkflowNotifier(ctx *bootstrap.Context, svc *InternalMessageService) *WorkflowNotifier {
	return &WorkflowNotifier{
		log: ctx.NewLoggerHelper("workflow-notify/core-service"),
		svc: svc,
	}
}

// newNotifyMessage 构造站内信通知请求。
func newNotifyMessage(recipientUserIDs []int64, title, content string) *internalMessageV1.SendMessageRequest {
	return &internalMessageV1.SendMessageRequest{
		Type:          internalMessageV1.InternalMessage_NOTIFICATION,
		TargetUserIds: recipientUserIDs,
		Title:         trans.Ptr(title),
		Content:       content,
	}
}

// NotifyManyAsync 异步发送站内信通知（可多接收人）。fire-and-forget，失败不影响状态机。
func (n *WorkflowNotifier) NotifyManyAsync(ctx context.Context, recipientUserIDs []int64, title, content string) {
	if n == nil || n.svc == nil || len(recipientUserIDs) == 0 {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				n.log.Errorf("notify panic: %v", r)
			}
		}()
		// WithoutCancel 保留原 ctx 的 viewer（SendMessage 从 viewer 推导发送者，
		// 防伪造），同时脱离已返回的 gRPC 请求生命周期。
		notifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if _, err := n.svc.SendMessage(notifyCtx, newNotifyMessage(recipientUserIDs, title, content)); err != nil {
			n.log.Errorf("notify send failed: %s", err.Error())
		}
	}()
}

// ===================== flow.Logger 适配 =====================

type flowLogger struct{ h *log.Helper }

func (l flowLogger) Printf(format string, args ...any) { l.h.Infof(format, args...) }

// ===================== flow.IdentityProvider 适配 =====================

// workflowIdentityProvider 办理人解析：LEADER=申请人主组织负责人，
// POSITION=职位在职持有人；委托代办叠加由引擎在解析结果上调
// ResolveDelegate 完成（与旧引擎 applyDelegation 语义一致）。
type workflowIdentityProvider struct {
	resolver   *data.WorkflowResolverRepo
	delegation *data.WorkflowDelegationRepo
	log        *log.Helper
}

func (p *workflowIdentityProvider) ResolveApprovers(ctx context.Context, req flow.ResolveRequest) ([]string, error) {
	tid := data.FromFlowIDOrZero(req.TenantID)
	starter := data.FromFlowIDOrZero(req.StarterID)

	var ids []int64
	seen := map[int64]bool{}
	appendID := func(uid int64) {
		if uid != 0 && !seen[uid] {
			seen[uid] = true
			ids = append(ids, uid)
		}
	}

	for _, rule := range req.Rules {
		switch rule.Kind {
		case flow.ApproverUser:
			appendID(data.FromFlowIDOrZero(rule.Value))
		case flow.ApproverStarter:
			appendID(starter)
		case flow.ApproverLeader:
			leader, err := p.resolver.ResolveOrgLeader(ctx, tid, starter)
			if err != nil {
				return nil, err
			}
			appendID(leader)
		case flow.ApproverPosition:
			posID := data.FromFlowIDOrZero(rule.Value)
			if posID == 0 {
				return nil, oaV1.ErrorBadRequest("POSITION approver missing position id")
			}
			holders, err := p.resolver.ResolvePositionHolders(ctx, tid, posID)
			if err != nil {
				return nil, err
			}
			for _, h := range holders {
				appendID(h)
			}
		default:
			// ROLE/EXPR 等规则 OA 定义体系不产出；fail-closed 拒绝。
			return nil, oaV1.ErrorBadRequest("unsupported approver rule: %s", string(rule.Kind))
		}
	}

	// 申请人剔除：办理人中的申请人本人自动跳过；剔除后为空则保留申请人
	//（监听器会代其立即通过，等价旧引擎的「自动通过」分支）。
	others := make([]int64, 0, len(ids))
	for _, uid := range ids {
		if uid != starter {
			others = append(others, uid)
		}
	}
	if len(others) == 0 && starter != 0 {
		others = []int64{starter}
	}

	out := make([]string, 0, len(others))
	for _, uid := range others {
		out = append(out, data.FlowIDOf(uid))
	}
	return out, nil
}

func (p *workflowIdentityProvider) ResolveDelegate(ctx context.Context, tenantID, userID string) (string, error) {
	delegate, err := p.delegation.ResolveDelegate(ctx, data.FromFlowIDOrZero(tenantID), data.FromFlowIDOrZero(userID))
	if err != nil {
		return "", err
	}
	if delegate == 0 {
		return userID, nil
	}
	return data.FlowIDOf(delegate), nil
}

func (p *workflowIdentityProvider) ResolveGroupsForUser(_ context.Context, _, _ string) ([]string, error) {
	// OA 无候选组体系；候选组任务（GROUP 规则）不在定义格式中产出。
	return nil, nil
}

// ===================== flow.Listener 适配 =====================

// workflowFlowListener 引擎事件监听器，承接旧引擎的三个副作用：
//   - 申请人自审自动通过（task.created 且办理人==申请人 → 代其通过）；
//   - 新待办站内信通知（task.created 且任务仍未办结）；
//   - 实例终结通知申请人 + 业务单据回调（process.ended）。
//
// 监听器按注册顺序执行：自动通过在通知之前——通知前重查任务状态，
// 已被自动通过/取消的任务不再打扰办理人。
type workflowFlowListener struct {
	log       *log.Helper
	engine    *flow.Engine
	store     *data.FlowStore
	registry  *WorkflowEventRegistry
	notifier  *WorkflowNotifier
	autoDepth int // 自审自动通过的级联深度（防自环定义递归失控）
}

// autoApproveMaxDepth 自审自动通过级联深度上限。
const autoApproveMaxDepth = 64

func (l *workflowFlowListener) OnFlowEvent(ctx context.Context, ev *flow.Event) {
	switch ev.Type {
	case flow.EventTaskCreated:
		l.autoApproveIfSelf(ctx, ev)
		l.notifyTaskCreated(ctx, ev)
	case flow.EventProcessEnded:
		l.onInstanceEnded(ctx, ev)
	default:
		// 令牌移动/网关判定/变量变更等流转事件无宿主副作用。
	}
}

// autoApproveIfSelf 办理人为申请人本人时自动通过（对齐旧引擎日志文案）。
// 自动通过沿图级联（下一节点办理人仍是申请人时继续），与旧引擎 walker 的
// 自动通过分支一致；autoDepth 限深防自环定义递归失控（超限时任务留给申请人手动处理）。
func (l *workflowFlowListener) autoApproveIfSelf(ctx context.Context, ev *flow.Event) {
	if ev.ActorID == "" || ev.TaskID == "" {
		return
	}
	if l.autoDepth >= autoApproveMaxDepth {
		l.log.Errorf("auto-approve depth limit exceeded at instance %s node %s", ev.InstanceID, ev.NodeID)
		return
	}
	inst, err := l.store.Instances().Get(ctx, ev.InstanceID)
	if err != nil {
		l.log.Errorf("auto-approve lookup instance %s failed: %s", ev.InstanceID, err.Error())
		return
	}
	if ev.ActorID != inst.StarterID {
		return
	}
	l.autoDepth++
	defer func() { l.autoDepth-- }()
	if err := l.engine.CompleteTask(ctx, flow.CompleteTaskRequest{
		TaskID:  ev.TaskID,
		ActorID: ev.ActorID,
		Outcome: flow.OutcomeApprove,
		Comment: "审批人为申请人本人，自动通过",
	}); err != nil {
		l.log.Errorf("auto-approve self task %s failed: %s", ev.TaskID, err.Error())
	}
}

// notifyTaskCreated 新待办通知（自动通过/或签取消的任务不再通知）。
func (l *workflowFlowListener) notifyTaskCreated(ctx context.Context, ev *flow.Event) {
	task, err := l.store.Tasks().Get(ctx, ev.TaskID)
	if err != nil || task.Status != flow.TaskPending || task.Assignee == "" {
		return
	}
	l.notifier.NotifyManyAsync(ctx, []int64{data.FromFlowIDOrZero(task.Assignee)}, "您有新的待办任务", "请登录系统查看待审批事项")
}

// onInstanceEnded 实例终结：通知申请人 + 触发业务单据回调。
// 子流程实例无 business 字段，回调由 registry 自行跳过；通知与旧引擎一致
// （子流程终结同样通知申请人，其 StarterID 继承自父实例）。
func (l *workflowFlowListener) onInstanceEnded(ctx context.Context, ev *flow.Event) {
	inst, err := l.store.Instances().Get(ctx, ev.InstanceID)
	if err != nil {
		l.log.Errorf("terminal event lookup instance %s failed: %s", ev.InstanceID, err.Error())
		return
	}
	tenantID := data.FromFlowIDOrZero(inst.TenantID)
	instanceID := data.FromFlowIDOrZero(inst.ID)
	businessID := data.FromFlowIDOrZero(inst.BusinessKey)
	starter := data.FromFlowIDOrZero(inst.StarterID)

	switch ev.Status {
	case flow.InstanceApproved:
		if starter != 0 {
			l.notifier.NotifyManyAsync(ctx, []int64{starter}, "您的申请已通过", "全部审批节点已通过")
		}
		l.fireBusiness(ctx, tenantID, instanceID, businessID, inst.BusinessType, oaV1.WorkflowInstance_APPROVED)
	case flow.InstanceRejected:
		if starter != 0 {
			l.notifier.NotifyManyAsync(ctx, []int64{starter}, "您的申请已被驳回", "请登录系统查看详情")
		}
		l.fireBusiness(ctx, tenantID, instanceID, businessID, inst.BusinessType, oaV1.WorkflowInstance_REJECTED)
	case flow.InstanceAborted:
		if starter != 0 {
			l.notifier.NotifyManyAsync(ctx, []int64{starter}, "您的申请已被系统终止", "流程定义异常，请联系管理员")
		}
		// ABORTED 按 fail-closed 对齐旧引擎：业务侧收到 REJECTED。
		l.fireBusiness(ctx, tenantID, instanceID, businessID, inst.BusinessType, oaV1.WorkflowInstance_REJECTED)
	case flow.InstanceWithdrawn:
		// 撤回的业务回调照常触发；对原待办人的通知由服务层处理
		//（撤回前的待办人列表在终结前查询）。
		l.fireBusiness(ctx, tenantID, instanceID, businessID, inst.BusinessType, oaV1.WorkflowInstance_WITHDRAWN)
	}
}

// fireBusinessEvent 触发业务单据终结回调，panic 隔离并记日志，失败不影响状态机。
func (l *workflowFlowListener) fireBusiness(
	ctx context.Context, tenantID, instanceID, businessID int64,
	businessType string, status oaV1.WorkflowInstance_InstanceStatus,
) {
	defer func() {
		if r := recover(); r != nil {
			l.log.Errorf("business hook panic (%s): %v", businessType, r)
		}
	}()
	l.registry.Fire(ctx, tenantID, instanceID, businessID, businessType, status)
}

// ===================== 装配入口 =====================

// NewWorkflowEngine 装配 go-wind-flow 引擎：ent 存储适配 + 身份解析 +
// expr-lang 求值器 + 事件监听器。定义解析经 windjson（存量 node_config 零迁移）。
func NewWorkflowEngine(
	ctx *bootstrap.Context,
	store *data.FlowStore,
	resolverRepo *data.WorkflowResolverRepo,
	delegationRepo *data.WorkflowDelegationRepo,
	eventRegistry *WorkflowEventRegistry,
	notifier *WorkflowNotifier,
) *flow.Engine {
	logger := flowLogger{h: ctx.NewLoggerHelper("workflow-engine/core-service")}
	identity := &workflowIdentityProvider{
		resolver:   resolverRepo,
		delegation: delegationRepo,
		log:        logger.h,
	}
	listener := &workflowFlowListener{
		log:      logger.h,
		store:    store,
		registry: eventRegistry,
		notifier: notifier,
	}
	engine := flow.New(
		flow.WithStore(store),
		flow.WithIdentity(identity),
		flow.WithEvaluator(exprlang.New()),
		flow.WithLogger(logger),
		flow.WithImporters(windjson.Importer{}),
		flow.WithListener(listener),
	)
	listener.engine = engine
	return engine
}
