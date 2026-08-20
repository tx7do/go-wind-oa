package service

import (
	"context"
	"sync"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// WorkflowBusinessHook 业务单据审批终结回调。status 为实例终态（APPROVED/REJECTED/WITHDRAWN）。
// 回调实现须自行校验业务单据与实例的关联（business_id 对应单据的 instance_id 应等于
// instanceID），防止客户端伪造 business 字段触发他人单据的副作用；失败在回调内自行记日志。
type WorkflowBusinessHook func(ctx context.Context, tenantID, instanceID, businessID uint32, status oaV1.WorkflowInstance_InstanceStatus)

// WorkflowEventRegistry 按业务单据类型注册的审批终结回调表。core 进程内解耦
// WorkflowService 与业务模块（请假/报销）：业务 service 构造时注册，引擎终结时同步回调。
type WorkflowEventRegistry struct {
	mu    sync.RWMutex
	hooks map[string]WorkflowBusinessHook
}

func NewWorkflowEventRegistry() *WorkflowEventRegistry {
	return &WorkflowEventRegistry{hooks: make(map[string]WorkflowBusinessHook)}
}

// Register 注册某业务类型的终结回调。同名类型重复注册以后者覆盖。
func (r *WorkflowEventRegistry) Register(businessType string, hook WorkflowBusinessHook) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks[businessType] = hook
}

// Fire 触发业务单据的终结回调（同步执行）。businessType 为空或未注册时静默跳过。
// 调用方（引擎侧）负责 recover 保护与失败日志，回调 panic/失败不影响状态机结果。
func (r *WorkflowEventRegistry) Fire(
	ctx context.Context,
	tenantID, instanceID, businessID uint32,
	businessType string,
	status oaV1.WorkflowInstance_InstanceStatus,
) {
	if businessType == "" || businessID == 0 {
		return
	}
	r.mu.RLock()
	hook, ok := r.hooks[businessType]
	r.mu.RUnlock()
	if !ok {
		return
	}
	hook(ctx, tenantID, instanceID, businessID, status)
}
