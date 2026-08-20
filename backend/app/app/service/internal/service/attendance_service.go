package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	appV1 "go-wind-oa/api/gen/go/app/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// AttendanceService 是 app 边端的考勤转发层（移动端）。打卡与本人记录查询，
// 用户身份由 core 侧 viewer context 决定。
type AttendanceService struct {
	appV1.AttendanceServiceHTTPServer

	log *log.Helper

	attendanceServiceClient oaV1.AttendanceServiceClient
}

func NewAttendanceService(
	ctx *bootstrap.Context,
	attendanceServiceClient oaV1.AttendanceServiceClient,
) *AttendanceService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "attendance/service/app-service"))
	return &AttendanceService{
		log:                     l,
		attendanceServiceClient: attendanceServiceClient,
	}
}

func (s *AttendanceService) CheckIn(ctx context.Context, req *oaV1.CheckInRequest) (*oaV1.AttendanceRecord, error) {
	return s.attendanceServiceClient.CheckIn(ctx, req)
}

func (s *AttendanceService) GetMyAttendanceRecords(ctx context.Context, req *oaV1.GetMyAttendanceRecordsRequest) (*oaV1.ListAttendanceRecordsResponse, error) {
	return s.attendanceServiceClient.GetMyAttendanceRecords(ctx, req)
}
