package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-oa/app/core/service/internal/data"
	"go-wind-oa/app/core/service/internal/data/ent/attendancerecord"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

type AttendanceService struct {
	oaV1.UnimplementedAttendanceServiceServer

	log *log.Helper

	repo         *data.AttendanceRepo
	leaveApp     *data.LeaveApplicationRepo
	resolverRepo *data.WorkflowResolverRepo
}

func NewAttendanceService(
	ctx *bootstrap.Context,
	repo *data.AttendanceRepo,
	leaveApp *data.LeaveApplicationRepo,
	resolverRepo *data.WorkflowResolverRepo,
) *AttendanceService {
	return &AttendanceService{
		log:          ctx.NewLoggerHelper("attendance/service/core-service"),
		repo:         repo,
		leaveApp:     leaveApp,
		resolverRepo: resolverRepo,
	}
}

// fillUserNames 批量回填用户姓名（失败仅缺姓名，不影响列表）。
func (s *AttendanceService) fillUserNames(ctx context.Context, tid uint32, items []*oaV1.AttendanceRecord) {
	ids := make([]uint32, 0, len(items))
	for _, it := range items {
		if it.GetUserId() != 0 {
			ids = append(ids, it.GetUserId())
		}
	}
	names, err := s.resolverRepo.ResolveUsernames(ctx, tid, ids)
	if err != nil {
		return
	}
	for _, it := range items {
		if name, ok := names[it.GetUserId()]; ok {
			it.UserName = trans.Ptr(name)
		}
	}
}

// isWeekend 周六/周日（节假日表为未来扩展，当前仅按周末判断）。
func isWeekend(t time.Time) bool {
	switch t.Weekday() {
	case time.Saturday, time.Sunday:
		return true
	default:
		return false
	}
}

// truncateDate 截断到当日零点（服务器本地时区）。
func truncateDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// parseHHMM 解析 "HH:MM" 为 workDate 当日时刻。
func parseHHMM(workDate time.Time, hhmm string) (time.Time, bool) {
	layouts := []string{"15:04", "15:04:05"}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, hhmm, workDate.Location()); err == nil {
			return time.Date(workDate.Year(), workDate.Month(), workDate.Day(),
				parsed.Hour(), parsed.Minute(), 0, 0, workDate.Location()), true
		}
	}
	return time.Time{}, false
}

// computeDayResult 按考勤设置结算：请假覆盖优先，其次迟到、早退判定。
func (s *AttendanceService) computeDayResult(
	ctx context.Context, tid, userID uint32,
	workDate, checkInAt, checkOutAt time.Time,
) (oaV1.AttendanceRecord_DayResult, error) {
	setting, err := s.repo.GetSetting(ctx, tid)
	if err != nil {
		return oaV1.AttendanceRecord_PENDING, err
	}
	workStart, ok1 := parseHHMM(workDate, setting.GetWorkStartTime())
	workEnd, ok2 := parseHHMM(workDate, setting.GetWorkEndTime())
	if !ok1 || !ok2 {
		return oaV1.AttendanceRecord_PENDING, oaV1.ErrorConflict("invalid attendance setting time format")
	}

	if covered, err := s.leaveApp.HasApprovedLeaveCovering(ctx, tid, userID, workDate); err != nil {
		return oaV1.AttendanceRecord_PENDING, err
	} else if covered {
		return oaV1.AttendanceRecord_ON_LEAVE, nil
	}

	late := checkInAt.After(workStart)
	earlyLeave := checkOutAt.Before(workEnd)
	if late {
		return oaV1.AttendanceRecord_LATE, nil
	}
	if earlyLeave {
		return oaV1.AttendanceRecord_EARLY_LEAVE, nil
	}
	return oaV1.AttendanceRecord_NORMAL, nil
}

// CheckIn 打卡：当日首次=签到；已签到未签退=签退并结算；已签退=409。
func (s *AttendanceService) CheckIn(ctx context.Context, req *oaV1.CheckInRequest) (*oaV1.AttendanceRecord, error) {
	tid, uid, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}

	now := time.Now()
	workDate := truncateDate(now)

	record, err := s.repo.GetByUserDate(ctx, tid, uid, workDate)
	if err != nil {
		return nil, err
	}
	if record == nil {
		created, err := s.repo.CreateCheckIn(ctx, tid, uid, workDate, now, req.GetLatitude(), req.GetLongitude(), req.GetWifiBssid())
		if err != nil {
			return nil, err
		}
		return data.AttendanceRecordToDTO(created), nil
	}
	if record.CheckInAt != nil && record.CheckOutAt != nil {
		return nil, oaV1.ErrorConflict("already checked out today")
	}
	if record.CheckInAt == nil {
		// 理论不可达（结算物化的记录无签到时间），防御处理：补签到。
		created, err := s.repo.CreateCheckIn(ctx, tid, uid, workDate, now, req.GetLatitude(), req.GetLongitude(), req.GetWifiBssid())
		if err != nil {
			return nil, err
		}
		return data.AttendanceRecordToDTO(created), nil
	}

	// 第二次打卡：签退并结算。
	checkInAt := *record.CheckInAt
	result, err := s.computeDayResult(ctx, tid, uid, workDate, checkInAt, now)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetCheckOut(ctx, tid, record.ID, now, req.GetLatitude(), req.GetLongitude(), req.GetWifiBssid(), result); err != nil {
		return nil, err
	}
	updated, err := s.repo.GetByUserDate(ctx, tid, uid, workDate)
	if err != nil {
		return nil, err
	}
	return data.AttendanceRecordToDTO(updated), nil
}

func (s *AttendanceService) GetMyAttendanceRecords(ctx context.Context, req *oaV1.GetMyAttendanceRecordsRequest) (*oaV1.ListAttendanceRecordsResponse, error) {
	tid, uid, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	end := time.Now()
	start := end.AddDate(0, 0, -30)
	if req.GetStartDate() != nil {
		start = truncateDate(req.GetStartDate().AsTime().In(time.Local))
	}
	if req.GetEndDate() != nil {
		end = truncateDate(req.GetEndDate().AsTime().In(time.Local))
	}
	items, err := s.repo.ListByUserRange(ctx, tid, uid, start, end)
	if err != nil {
		return nil, err
	}
	s.fillUserNames(ctx, tid, items)
	return &oaV1.ListAttendanceRecordsResponse{Items: items, Total: uint64(len(items))}, nil
}

func (s *AttendanceService) ListAttendanceRecords(ctx context.Context, req *oaV1.ListAttendanceRecordsRequest) (*oaV1.ListAttendanceRecordsResponse, error) {
	tid, _, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	if req.GetWorkDate() == nil {
		return nil, oaV1.ErrorBadRequest("work date required")
	}
	items, err := s.repo.ListByDate(ctx, tid, req.GetUserId(), truncateDate(req.GetWorkDate().AsTime().In(time.Local)))
	if err != nil {
		return nil, err
	}
	s.fillUserNames(ctx, tid, items)
	return &oaV1.ListAttendanceRecordsResponse{Items: items, Total: uint64(len(items))}, nil
}

func (s *AttendanceService) GetAttendanceSetting(ctx context.Context, _ *emptypb.Empty) (*oaV1.AttendanceSetting, error) {
	tid, _, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	return s.repo.GetSetting(ctx, tid)
}

func (s *AttendanceService) UpdateAttendanceSetting(ctx context.Context, req *oaV1.AttendanceSetting) (*emptypb.Empty, error) {
	tid, uid, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	if _, ok1 := parseHHMM(time.Now(), req.GetWorkStartTime()); !ok1 {
		return nil, oaV1.ErrorBadRequest("invalid work start time")
	}
	if _, ok2 := parseHHMM(time.Now(), req.GetWorkEndTime()); !ok2 {
		return nil, oaV1.ErrorBadRequest("invalid work end time")
	}
	if err := s.repo.UpdateSetting(ctx, tid, uid, req.GetWorkStartTime(), req.GetWorkEndTime()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// RunDailySettlement 工作日结算（admin 手动触发，也可由外部定时任务调用）：
// 无记录用户按请假覆盖与否物化 ON_LEAVE/ABSENT；已签退未结算的记录补结算。
// 周末不物化旷工/请假记录（节假日表为未来扩展）。
func (s *AttendanceService) RunDailySettlement(ctx context.Context, req *oaV1.RunDailySettlementRequest) (*oaV1.RunDailySettlementResponse, error) {
	tid, uid, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	workDate := truncateDate(time.Now())
	if req.GetWorkDate() != nil {
		// 请求时间戳固定 UTC 位置（Timestamp.AsTime 语义），转本地后截断，
		// 与打卡记录的本地日期轴对齐。
		workDate = truncateDate(req.GetWorkDate().AsTime().In(time.Local))
	}
	if isWeekend(workDate) {
		return &oaV1.RunDailySettlementResponse{SettledCount: 0}, nil
	}
	settled, err := s.settleDateForTenant(ctx, tid, uid, workDate)
	if err != nil {
		return nil, err
	}
	return &oaV1.RunDailySettlementResponse{SettledCount: settled}, nil
}

// settleDateForTenant 单租户单日结算（定时任务与 RPC 共用）。operatorID 落
// updated_by/created_by 审计字段。
func (s *AttendanceService) settleDateForTenant(ctx context.Context, tid, operatorID uint32, workDate time.Time) (uint32, error) {
	userIDs, err := s.repo.ListUserIDs(ctx, tid)
	if err != nil {
		return 0, err
	}
	settled := uint32(0)
	for _, userID := range userIDs {
		record, err := s.repo.GetByUserDate(ctx, tid, userID, workDate)
		if err != nil {
			return 0, err
		}
		if record == nil {
			covered, err := s.leaveApp.HasApprovedLeaveCovering(ctx, tid, userID, workDate)
			if err != nil {
				return 0, err
			}
			result := oaV1.AttendanceRecord_ABSENT
			if covered {
				result = oaV1.AttendanceRecord_ON_LEAVE
			}
			if err := s.repo.SettleMaterialize(ctx, tid, userID, workDate, result, operatorID); err != nil {
				return 0, err
			}
			settled++
			continue
		}
		// 已签退但仍 PENDING 的补结算（签退时结算失败或旧数据）。
		if record.CheckInAt != nil && record.CheckOutAt != nil && record.DayResult != nil && *record.DayResult == attendancerecord.DayResultPending {
			result, err := s.computeDayResult(ctx, tid, userID, workDate, *record.CheckInAt, *record.CheckOutAt)
			if err != nil {
				return 0, err
			}
			if err := s.repo.SettlePending(ctx, tid, record.ID, result, operatorID); err != nil {
				return 0, err
			}
			settled++
		}
	}
	return settled, nil
}
