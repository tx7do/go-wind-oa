package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
)

// AttendanceScheduler 考勤每日定时结算。
//
// 每 30 分钟唤醒一次，在本地时间 00:30 之后为「昨日」执行全租户结算（物化旷工/
// 请假、补结算漏结算记录；周末跳过）。经 wire 注入即随服务启动，周期任务在服务
// 生命周期内常驻。
//
// 说明：结算走 repo 层显式租户过滤，不依赖请求级 viewer 上下文；若 ent 隐私层
// 对无 viewer 的写入有额外限制，错误会被记录且下一周期重试（幂等）。
type AttendanceScheduler struct {
	log         *log.Helper
	attendance  *AttendanceService
	lastRunDate string
}

func NewAttendanceScheduler(
	ctx *bootstrap.Context,
	attendance *AttendanceService,
) *AttendanceScheduler {
	s := &AttendanceScheduler{
		log:        ctx.NewLoggerHelper("attendance/scheduler/core-service"),
		attendance: attendance,
	}
	go s.loop()
	return s
}

func (s *AttendanceScheduler) loop() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		if now.Hour() != 0 || now.Minute() < 30 {
			continue
		}
		today := truncateDate(now).Format("2006-01-02")
		if s.lastRunDate == today {
			continue
		}
		s.lastRunDate = today
		s.runSettlementForYesterday(now)
	}
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
