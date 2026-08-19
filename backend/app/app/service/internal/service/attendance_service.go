package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
	appV1 "go-wind-oa/api/gen/go/app/service/v1"
)

// AttendanceService app-service 側的移動打卡 HTTP 邊端轉發層。
//
// 與 WorkflowService 同構：持有 HTTP 邊端（由 buf 生成的
// appV1.AttendanceServiceHTTPServer），收到 CheckIn 請求後原樣轉發到
// core-service 的 AttendanceService gRPC 實現。本層不做業務邏輯，僅做
// 協議邊界轉換。
//
// app 邊端僅暴露 CheckIn（移動端打卡）；圍欄 / Wi-Fi 指紋庫的 CRUD 不經
// app 邊端，由 admin-service 暴露給管理端維護。
//
// 鑑權 / 租戶隔離由 app-service 的 REST 中間件鏈在請求進入本層前完成。
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

func (s *AttendanceService) CheckIn(ctx context.Context, req *oaV1.CheckInRequest) (*oaV1.CheckInResponse, error) {
	return s.attendanceServiceClient.CheckIn(ctx, req)
}
