package server

import (
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/transport/grpc"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/kratos-bootstrap/rpc"

	"go-wind-oa/app/core/service/internal/service"

	internalMessageV1 "go-wind-oa/api/gen/go/internal_message/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"

	"go-wind-oa/pkg/middleware/ent"
)

func NewGrpcMiddleware(ctx *bootstrap.Context) []middleware.Middleware {
	var ms []middleware.Middleware
	ms = append(ms, logging.Server(ctx.GetLogger()))
	ms = append(ms, ent.Server())
	return ms
}

// NewGrpcServer new a gRPC server.
//
// OA core-service 為 gRPC-only，僅註冊三類服務：
//   - internal_message（站內信，工作流通知落庫用）
//   - workflow（工作流引擎狀態機）
//   - attendance（移動打卡判定 + 圍欄/Wi-Fi 指紋庫 CRUD）
//
// 中間件鏈為 logging + ent：前者記錄調用，後者按 auth 中間件注入的
// viewer context 做租戶/操作人隔離（core 本身不驗 token，由上游 admin
// 轉發時帶身份）。
func NewGrpcServer(
	ctx *bootstrap.Context,
	middlewares []middleware.Middleware,

	internalMessageService *service.InternalMessageService,
	internalMessageCategoryService *service.InternalMessageCategoryService,
	internalMessageRecipientService *service.InternalMessageRecipientService,

	workflowService *service.WorkflowService,

	attendanceService *service.AttendanceService,
) (*grpc.Server, error) {
	cfg := ctx.GetConfig()

	if cfg == nil || cfg.Server == nil || cfg.Server.Grpc == nil {
		return nil, nil
	}

	srv, err := rpc.CreateGrpcServer(cfg, middlewares...)
	if err != nil {
		return nil, err
	}

	internalMessageV1.RegisterInternalMessageServiceServer(srv, internalMessageService)
	internalMessageV1.RegisterInternalMessageCategoryServiceServer(srv, internalMessageCategoryService)
	internalMessageV1.RegisterInternalMessageRecipientServiceServer(srv, internalMessageRecipientService)

	oaV1.RegisterWorkflowServiceServer(srv, workflowService)

	oaV1.RegisterAttendanceServiceServer(srv, attendanceService)

	return srv, nil
}
