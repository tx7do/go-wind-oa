package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-oa/pkg/task"
)

// AttendanceScheduler 考勤每日定时结算。
//
// 由 asynq periodic 任务 oa:attendance:settle（每日 00:30）触发，
// 为「昨日」执行全租户结算（物化旷工/请假、补结算漏结算记录；周末跳过）。
// 调度由 asynq 周期任务保证多实例下每日只执行一次。
//
// 说明：结算走 repo 层显式租户过滤，不依赖请求级 viewer 上下文；若 ent 隐私层
// 对无 viewer 的写入有额外限制，错误会被记录且下一周期重试（幂等）。
type AttendanceScheduler struct {
	log        *log.Helper
	attendance *AttendanceService
}

func NewAttendanceScheduler(
	ctx *bootstrap.Context,
	attendance *AttendanceService,
) *AttendanceScheduler {
	return &AttendanceScheduler{
		log:        ctx.NewLoggerHelper("attendance/scheduler/core-service"),
		attendance: attendance,
	}
}

// AsyncAttendanceSettle asynq 任务入口：执行一次昨日考勤结算。
func (s *AttendanceScheduler) AsyncAttendanceSettle(taskType string, _ *task.AttendanceSettleData) error {
	s.log.Infof("attendance settlement triggered [%s]", taskType)
	s.runSettlementForYesterday(time.Now())
	return nil
}

// runSettlementForYesterday 对全部租户结算昨日考勤。
func (s *AttendanceScheduler) runSettlementForYesterday(now time.Time) {
	defer func() {
		if r := recover(); r != nil {
			s.log.Errorf("attendance settlement panic: %v", r)
		}
	}()

	yesterday := truncateDate(now.AddDate(0, 0, -1))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	tenantIDs, err := s.attendance.repo.ListTenantIDs(ctx)
	if err != nil {
		s.log.Errorf("settlement list tenants failed: %s", err.Error())
		return
	}
	for _, tid := range tenantIDs {
		// 跳过休息日（节假日表优先于周末；调休 WORKDAY 的周末照常结算）。
		if rest, err := s.attendance.isRestDay(ctx, tid, yesterday); err != nil {
			s.log.Errorf("settlement check rest day (tenant %d) failed: %s", tid, err.Error())
			continue
		} else if rest {
			continue
		}
		settled, err := s.attendance.settleDateForTenant(ctx, tid, 0, yesterday)
		if err != nil {
			s.log.Errorf("settlement tenant %d failed: %s", tid, err.Error())
			continue
		}
		s.log.Infof("settlement tenant %d %s: %d records", tid, yesterday.Format("2006-01-02"), settled)
	}
}
