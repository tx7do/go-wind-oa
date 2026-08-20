package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	adminV1 "go-wind-oa/api/gen/go/admin/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// AttendanceService 是 admin 边端的考勤管理转发层（记录查询/设置/结算）。
type AttendanceService struct {
	adminV1.AttendanceServiceHTTPServer

	log *log.Helper

	attendanceServiceClient oaV1.AttendanceServiceClient
}

func NewAttendanceService(
	ctx *bootstrap.Context,
	attendanceServiceClient oaV1.AttendanceServiceClient,
) *AttendanceService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "attendance/service/admin-service"))
	return &AttendanceService{
		log:                     l,
		attendanceServiceClient: attendanceServiceClient,
	}
}

func (s *AttendanceService) ListAttendanceRecords(ctx context.Context, req *oaV1.ListAttendanceRecordsRequest) (*oaV1.ListAttendanceRecordsResponse, error) {
	return s.attendanceServiceClient.ListAttendanceRecords(ctx, req)
}

func (s *AttendanceService) GetAttendanceSetting(ctx context.Context, e *emptypb.Empty) (*oaV1.AttendanceSetting, error) {
	return s.attendanceServiceClient.GetAttendanceSetting(ctx, e)
}

func (s *AttendanceService) UpdateAttendanceSetting(ctx context.Context, req *oaV1.AttendanceSetting) (*emptypb.Empty, error) {
	return s.attendanceServiceClient.UpdateAttendanceSetting(ctx, req)
}

func (s *AttendanceService) RunDailySettlement(ctx context.Context, req *oaV1.RunDailySettlementRequest) (*oaV1.RunDailySettlementResponse, error) {
	return s.attendanceServiceClient.RunDailySettlement(ctx, req)
}

func (s *AttendanceService) UpsertHoliday(ctx context.Context, req *oaV1.Holiday) (*emptypb.Empty, error) {
	return s.attendanceServiceClient.UpsertHoliday(ctx, req)
}

func (s *AttendanceService) DeleteHoliday(ctx context.Context, req *oaV1.DeleteHolidayRequest) (*emptypb.Empty, error) {
	return s.attendanceServiceClient.DeleteHoliday(ctx, req)
}

func (s *AttendanceService) ListHolidays(ctx context.Context, req *oaV1.ListHolidaysRequest) (*oaV1.ListHolidaysResponse, error) {
	return s.attendanceServiceClient.ListHolidays(ctx, req)
}
