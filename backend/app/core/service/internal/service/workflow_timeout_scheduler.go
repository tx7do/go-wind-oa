package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/go-utils/trans"

	internalMessageV1 "go-wind-oa/api/gen/go/internal_message/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
	"go-wind-oa/app/core/service/internal/data"
	"go-wind-oa/pkg/task"
)

// WorkflowTimeoutScheduler 工作流待办超时催办与自动升级。
//
// 由 asynq periodic 任务 oa:workflow:timeout-scan（每小时整点）触发，
// 扫描全部 PENDING 待办任务，按创建时间判定超时等级：
//   - 催办阈值（默认 24h）：向审批人发送催办站内信
//   - 升级阈值（默认 72h）：将待办转办给审批人的组织负责人（上级），并通知双方
//
// 催办/升级各执行一次：任务的 reminded_at / escalated_at 非空即跳过，
// 避免重复催办、以及升级转办后按新审批人沿组织树连环上转。
// 调度由 asynq 周期任务保证多实例下每小时只执行一次。
type WorkflowTimeoutScheduler struct {
	log                 *log.Helper
	taskRepo            *data.WorkflowTaskRepo
	instanceRepo        *data.WorkflowInstanceRepo
	resolverRepo        *data.WorkflowResolverRepo
	logRepo             *data.WorkflowLogRepo
	notificationService *InternalMessageService
}

func NewWorkflowTimeoutScheduler(
	ctx *bootstrap.Context,
	taskRepo *data.WorkflowTaskRepo,
	instanceRepo *data.WorkflowInstanceRepo,
	resolverRepo *data.WorkflowResolverRepo,
	logRepo *data.WorkflowLogRepo,
	notificationService *InternalMessageService,
) *WorkflowTimeoutScheduler {
	return &WorkflowTimeoutScheduler{
		log:                 ctx.NewLoggerHelper("workflow-timeout/scheduler/core-service"),
		taskRepo:            taskRepo,
		instanceRepo:        instanceRepo,
		resolverRepo:        resolverRepo,
		logRepo:             logRepo,
		notificationService: notificationService,
	}
}

const (
	reminderThreshold   = 24 * time.Hour
	escalationThreshold = 72 * time.Hour
)

// AsyncTimeoutScan asynq 任务入口：执行一次超时扫描。
func (s *WorkflowTimeoutScheduler) AsyncTimeoutScan(taskType string, _ *task.WorkflowTimeoutScanData) error {
	s.log.Infof("workflow timeout scan triggered [%s]", taskType)
	s.scanTimeouts(time.Now())
	return nil
}

func (s *WorkflowTimeoutScheduler) scanTimeouts(now time.Time) {
	defer func() {
		if r := recover(); r != nil {
			s.log.Errorf("workflow timeout scan panic: %v", r)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	tasks, err := s.taskRepo.ListAllPendingWithAge(ctx)
	if err != nil {
		s.log.Errorf("scan list pending failed: %s", err.Error())
		return
	}

	for _, task := range tasks {
		age := now.Sub(task.CreatedAt)

		if age >= escalationThreshold {
			// 已升级过的不再处理：升级通知已覆盖双方，也不重复催办。
			if task.EscalatedAt == nil {
				s.escalateTask(ctx, task)
			}
		} else if age >= reminderThreshold && task.RemindedAt == nil {
			s.remindTask(ctx, task)
		}
	}
}

func (s *WorkflowTimeoutScheduler) remindTask(ctx context.Context, task data.PendingTaskInfo) {
	// 催办：向审批人发送站内信提醒。
	s.sendNotify(ctx, []uint32{task.Assignee}, "您有待审批任务超时未处理", "请尽快登录系统处理待审批事项，超时未处理将自动升级至您的上级。")
	if err := s.taskRepo.MarkReminded(ctx, task.TaskID, task.TenantID); err != nil {
		s.log.Errorf("mark reminded for task %d failed: %s", task.TaskID, err.Error())
	}
}

func (s *WorkflowTimeoutScheduler) escalateTask(ctx context.Context, task data.PendingTaskInfo) {
	// 升级：将待办转办给审批人的组织负责人。
	leaderID, err := s.resolverRepo.ResolveOrgLeader(ctx, task.TenantID, task.Assignee)
	if err != nil {
		s.log.Errorf("escalation resolve leader for assignee %d failed: %s", task.Assignee, err.Error())
		// 无上级可升级：发催办但不转办。记 reminded_at 防止每小时重发。
		s.sendNotify(ctx, []uint32{task.Assignee}, "您有待审批任务严重超时", "该任务已超时较久但仍未处理，且系统无法找到您的上级进行升级，请立即处理。")
		s.markReminded(ctx, task)
		return
	}
	if leaderID == task.Assignee {
		// 审批人即自己的上级：无法升级，仅催办。记 reminded_at 防止每小时重发。
		s.sendNotify(ctx, []uint32{task.Assignee}, "您有待审批任务严重超时", "该任务已超时较久，请立即处理。")
		s.markReminded(ctx, task)
		return
	}

	// 转办任务给上级。
	if err := s.taskRepo.UpdateAssignee(ctx, task.TaskID, task.TenantID, leaderID); err != nil {
		s.log.Errorf("escalation reassign task %d to leader %d failed: %s", task.TaskID, leaderID, err.Error())
		return
	}

	// 转办成功后记 escalated_at：后续扫描对该待办不再升级、不再催办。
	if err := s.taskRepo.MarkEscalated(ctx, task.TaskID, task.TenantID); err != nil {
		s.log.Errorf("mark escalated for task %d failed: %s", task.TaskID, err.Error())
	}

	// 写转办日志。
	if _, err := s.logRepo.Create(ctx, task.TenantID, 0, task.InstanceID, task.NodeID, oaV1.WorkflowLog_FORWARD.Enum(), "超时自动升级转办"); err != nil {
		s.log.Errorf("escalation log forward failed: %s", err.Error())
	}

	// 通知上级有新待办，通知原审批人任务已升级。
	s.sendNotify(ctx, []uint32{leaderID}, "您有被转办的任务（超时升级）", "因原审批人超时未处理，该任务已自动转办给您，请登录系统处理。")
	s.sendNotify(ctx, []uint32{task.Assignee}, "您的待审批任务已自动升级", "因您超时未处理，该任务已自动转办给您的上级。")
}

func (s *WorkflowTimeoutScheduler) markReminded(ctx context.Context, task data.PendingTaskInfo) {
	if err := s.taskRepo.MarkReminded(ctx, task.TaskID, task.TenantID); err != nil {
		s.log.Errorf("mark reminded for task %d failed: %s", task.TaskID, err.Error())
	}
}

func (s *WorkflowTimeoutScheduler) sendNotify(ctx context.Context, recipients []uint32, title, content string) {
	if len(recipients) == 0 {
		return
	}
	notifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_, err := s.notificationService.SendMessage(notifyCtx, &internalMessageV1.SendMessageRequest{
		Type:          internalMessageV1.InternalMessage_NOTIFICATION,
		TargetUserIds: recipients,
		Title:         trans.Ptr(title),
		Content:       content,
	})
	if err != nil {
		s.log.Errorf("timeout notify send failed: %s", err.Error())
	}
}
