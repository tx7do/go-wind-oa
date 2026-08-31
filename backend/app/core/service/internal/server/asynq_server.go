package server

import (
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/kratos-transport/transport/asynq"

	"go-wind-oa/app/core/service/internal/service"

	appViewer "go-wind-oa/pkg/entgo/viewer"
	"go-wind-oa/pkg/task"
)

// NewAsynqServer creates a new asynq server.
func NewAsynqServer(
	ctx *bootstrap.Context,
	taskService *service.TaskService,
	workflowTimeoutScheduler *service.WorkflowTimeoutScheduler,
	attendanceScheduler *service.AttendanceScheduler,
) *asynq.Server {
	cfg := ctx.GetConfig()

	if cfg == nil || cfg.Server == nil || cfg.Server.Asynq == nil {
		return nil
	}

	srv := asynq.NewServer(
		asynq.WithCodec(cfg.Server.Asynq.GetCodec()),
		asynq.WithRedisURI(cfg.Server.Asynq.GetUri()),
		asynq.WithLocation(cfg.Server.Asynq.GetLocation()),
		asynq.WithGracefullyShutdown(cfg.Server.Asynq.GetEnableGracefullyShutdown()),
		asynq.WithShutdownTimeout(cfg.Server.Asynq.GetShutdownTimeout().AsDuration()),
	)

	taskService.RegisterTaskScheduler(srv)

	var err error

	// 注册任务
	if err = asynq.RegisterSubscriber(srv, task.BackupTaskType, taskService.AsyncBackup); err != nil {
		log.Error(err)
	}
	if err = asynq.RegisterSubscriber(srv, task.WorkflowTimeoutScanTaskType, workflowTimeoutScheduler.AsyncTimeoutScan); err != nil {
		log.Error(err)
	}
	if err = asynq.RegisterSubscriber(srv, task.AttendanceSettleTaskType, attendanceScheduler.AsyncAttendanceSettle); err != nil {
		log.Error(err)
	}

	// 内置系统周期任务：asynq periodic scheduler 经 Redis 抢锁，
	// 多实例下整点只入队一次，取代原进程内 ticker。
	if _, err = srv.NewPeriodicTask("0 * * * *", task.WorkflowTimeoutScanTaskType, task.WorkflowTimeoutScanData{}); err != nil {
		log.Error(err)
	}
	if _, err = srv.NewPeriodicTask("30 0 * * *", task.AttendanceSettleTaskType, task.AttendanceSettleData{}); err != nil {
		log.Error(err)
	}

	// 启动所有的任务
	_, _ = taskService.StartAllTask(appViewer.NewSystemViewerContext(ctx.Context()), nil)

	return srv
}
