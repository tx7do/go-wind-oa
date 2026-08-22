package server

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport/http"

	authzEngine "github.com/tx7do/kratos-authz/engine"
	authz "github.com/tx7do/kratos-authz/middleware"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/kratos-bootstrap/rpc"

	swaggerUI "github.com/tx7do/kratos-swagger-ui"

	"go-wind-oa/app/app/service/cmd/server/assets"
	"go-wind-oa/app/app/service/internal/service"

	appV1 "go-wind-oa/api/gen/go/app/service/v1"
	auditV1 "go-wind-oa/api/gen/go/audit/service/v1"

	"go-wind-oa/pkg/middleware/auth"
	entmiddleware "go-wind-oa/pkg/middleware/ent"
	applogging "go-wind-oa/pkg/middleware/logging"
)

// NewRestMiddleware 创建中间件
func NewRestMiddleware(
	ctx *bootstrap.Context,
	accessTokenChecker auth.AccessTokenChecker,
	authorizer authzEngine.Engine,
) []middleware.Middleware {
	var ms []middleware.Middleware
	ms = append(ms, logging.Server(ctx.GetLogger()))

	// add white list for authentication.
	rpc.AddWhiteList(
		appV1.OperationAuthenticationServiceLogin,
	)

	ms = append(ms, applogging.Server(
		applogging.WithWriteApiLogFunc(func(ctx context.Context, data *auditV1.ApiAuditLog) error {
			return nil
		}),
		applogging.WithWriteLoginLogFunc(func(ctx context.Context, data *auditV1.LoginAuditLog) error {
			return nil
		}),
	))

	// 鉴权必须在 ent.Server() 之前执行：auth.Server 对非白名单请求注入
	// OperatorMetadata，随后 ent.Server() 才能据此构建带租户作用域的 UserViewer。
	// 若顺序颠倒，ent.Server() 总以 md==nil 兜底为 SystemViewer，导致租户隔离失效。
	ms = append(ms, selector.Server(
		auth.Server(
			auth.WithAccessTokenChecker(accessTokenChecker),
			auth.WithInjectMetadata(true),
			auth.WithInjectEnt(true),
		),
		authz.Server(authorizer),
	).Match(rpc.NewRestWhiteListMatcher()).Build())

	// ent.Server() 必须在 auth.Server 之后：此时非白名单请求已注入 OperatorMetadata，
	// 可构建 UserViewer；白名单请求（登录/验证码）md==nil 但在白名单内，兜底 SystemViewer。
	ms = append(ms, entmiddleware.Server())

	return ms
}

// NewRestServer new an REST server.
func NewRestServer(
	ctx *bootstrap.Context,

	middlewares []middleware.Middleware,

	authenticationService *service.AuthenticationService,
	workflowService *service.WorkflowService,
	leaveService *service.LeaveService,
	expenseService *service.ExpenseService,
	businessTripService *service.BusinessTripService,
	overtimeService *service.OvertimeService,
	sealApplicationService *service.SealApplicationService,
	outingService *service.OutingService,
	attendanceService *service.AttendanceService,
	internalMessageService *service.InternalMessageService,
	fileTransferService *service.FileTransferService,
	userProfileService *service.UserProfileService,
	orgUnitService *service.OrgUnitService,
	userService *service.UserService,
) *http.Server {
	cfg := ctx.GetConfig()

	if cfg == nil || cfg.Server == nil || cfg.Server.Rest == nil {
		return nil
	}

	srv, err := rpc.CreateRestServer(cfg, middlewares...)
	if err != nil {
		panic(err)
	}

	appV1.RegisterAuthenticationServiceHTTPServer(srv, authenticationService)
	appV1.RegisterWorkflowServiceHTTPServer(srv, workflowService)
	appV1.RegisterLeaveServiceHTTPServer(srv, leaveService)
	appV1.RegisterExpenseServiceHTTPServer(srv, expenseService)
	appV1.RegisterBusinessTripServiceHTTPServer(srv, businessTripService)
	appV1.RegisterOvertimeServiceHTTPServer(srv, overtimeService)
	appV1.RegisterSealApplicationServiceHTTPServer(srv, sealApplicationService)
	appV1.RegisterOutingServiceHTTPServer(srv, outingService)
	appV1.RegisterAttendanceServiceHTTPServer(srv, attendanceService)
	appV1.RegisterInternalMessageServiceHTTPServer(srv, internalMessageService)
	// 文件上传为 multipart 流式处理，走手改的 registerFileTransferServiceHandler
	// （同路径的生成版会对 multipart 做 ctx.Bind 报 CODEC 400）。
	registerFileTransferServiceHandler(srv, fileTransferService)
	appV1.RegisterUserProfileServiceHTTPServer(srv, userProfileService)
	appV1.RegisterOrgUnitServiceHTTPServer(srv, orgUnitService)
	appV1.RegisterUserServiceHTTPServer(srv, userService)

	if cfg.GetServer().GetRest().GetEnableSwagger() {
		swaggerUI.RegisterSwaggerUIServerWithOption(
			srv,
			swaggerUI.WithTitle("GoWind OA App API"),
			swaggerUI.WithMemoryData(assets.OpenApiData, "yaml"),
		)
	}

	return srv
}
