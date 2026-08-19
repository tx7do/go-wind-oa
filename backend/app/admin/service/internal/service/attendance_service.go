package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	adminV1 "go-wind-oa/api/gen/go/admin/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// AttendanceService admin-service 側的考勤庫管理 HTTP 邊端轉發層。
//
// 對齊 WorkflowService 的轉發模式：admin-service 持有由
// admin/service/v1/i_attendance.proto 生成的 HTTP server interface，
// 收到圍欄 / Wi-Fi 指紋庫的 CRUD 請求後原樣轉發到 core-service 的
// AttendanceService gRPC 實現。本層不做業務邏輯，僅做協議邊界轉換。
//
// 鑑權 / 租戶隔離由 admin-service 的 REST 中間件鏈
// （auth.Server + ent.Server）在請求進入本層前完成：auth 注入
// OperatorMetadata，ent 據此構建帶租戶作用域的 viewer，core-service
// 的 TenantPrivacy 策略再按 viewer 做行級隔離。圍欄 / 指紋庫的寫入
// 租戶由 core-service 的 AttendanceService 從 viewer 取值強制賦值，
// 客戶端不可越權指定他租戶。
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

// ---- 围栏库 CRUD ----

func (s *AttendanceService) CreateAttendanceFence(ctx context.Context, req *oaV1.CreateAttendanceFenceRequest) (*oaV1.AttendanceFence, error) {
	return s.attendanceServiceClient.CreateAttendanceFence(ctx, req)
}

func (s *AttendanceService) ListAttendanceFence(ctx context.Context, req *oaV1.ListAttendanceFenceRequest) (*oaV1.ListAttendanceFenceResponse, error) {
	return s.attendanceServiceClient.ListAttendanceFence(ctx, req)
}

func (s *AttendanceService) UpdateAttendanceFence(ctx context.Context, req *oaV1.UpdateAttendanceFenceRequest) (*oaV1.AttendanceFence, error) {
	return s.attendanceServiceClient.UpdateAttendanceFence(ctx, req)
}

func (s *AttendanceService) DeleteAttendanceFence(ctx context.Context, req *oaV1.DeleteAttendanceFenceRequest) (*emptypb.Empty, error) {
	return s.attendanceServiceClient.DeleteAttendanceFence(ctx, req)
}

// ---- Wi-Fi 指纹库 CRUD ----

func (s *AttendanceService) CreateAttendanceWifi(ctx context.Context, req *oaV1.CreateAttendanceWifiRequest) (*oaV1.AttendanceWifi, error) {
	return s.attendanceServiceClient.CreateAttendanceWifi(ctx, req)
}

func (s *AttendanceService) ListAttendanceWifi(ctx context.Context, req *oaV1.ListAttendanceWifiRequest) (*oaV1.ListAttendanceWifiResponse, error) {
	return s.attendanceServiceClient.ListAttendanceWifi(ctx, req)
}

func (s *AttendanceService) DeleteAttendanceWifi(ctx context.Context, req *oaV1.DeleteAttendanceWifiRequest) (*emptypb.Empty, error) {
	return s.attendanceServiceClient.DeleteAttendanceWifi(ctx, req)
}
