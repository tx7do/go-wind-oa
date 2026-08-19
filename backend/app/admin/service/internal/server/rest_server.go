package server

import (
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport/http"

	authzEngine "github.com/tx7do/kratos-authz/engine"
	authz "github.com/tx7do/kratos-authz/middleware"

	swaggerUI "github.com/tx7do/kratos-swagger-ui"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/kratos-bootstrap/rpc"

	"go-wind-oa/app/admin/service/cmd/server/assets"
	"go-wind-oa/app/admin/service/internal/service"

	adminV1 "go-wind-oa/api/gen/go/admin/service/v1"

	"go-wind-oa/pkg/middleware/auth"
	entmiddleware "go-wind-oa/pkg/middleware/ent"
)

// NewRestMiddleware 创建中间件
//
// OA admin-service 的 REST 中間件鏈：
//   - logging：記錄調用
//   - auth.Server + authz.Server（白名單匹配）：非白名單請求注入 OperatorMetadata
//   - ent.Server()：按注入的身份構建帶租戶作用域的 viewer
//
// auth 必須在 ent 之前：順序顛倒則 ent 總以 md==nil 兜底 SystemViewer，租戶隔離失效。
// 白名單含 Login/GenerateCaptcha/VerifyCaptcha——驗證碼與登錄的雞生蛋問題。
//
// 不再接入 cms 的 api/login 審計日誌中間件（audit 域已裁剪）。
func NewRestMiddleware(
	ctx *bootstrap.Context,
	accessTokenChecker auth.AccessTokenChecker,
	authorizer authzEngine.Engine,
) []middleware.Middleware {
	var ms []middleware.Middleware
	ms = append(ms, logging.Server(ctx.GetLogger()))

	// add white list for authentication.
	// GenerateCaptcha 和 VerifyCaptcha 必须在白名单中，因为登录需要验证码，
	// 而获取验证码时用户尚无 token（鸡生蛋问题）。
	rpc.AddWhiteList(
		adminV1.OperationAuthenticationServiceLogin,
		adminV1.OperationAuthenticationServiceGenerateCaptcha,
		adminV1.OperationAuthenticationServiceVerifyCaptcha,
	)

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
//
// OA admin-service 的 HTTP 邊端註冊的轉發服務：
//   - AuthenticationService（鑑權：登錄/登出/驗證碼/令牌刷新，轉發 core gRPC）
//   - InternalMessage × 3（站內信通知，轉發 core gRPC + SSE 推送）
//   - WorkflowService（工作流定義/申請/審批/任務查詢，轉發 core gRPC）
//   - AttendanceService（圍欄 / Wi-Fi 指紋庫 CRUD，轉發 core gRPC）
//
// 其餘 cms 業務域的 HTTP 註冊均已隨 service 刪除而去除。
func NewRestServer(
	ctx *bootstrap.Context,

	middlewares []middleware.Middleware,

	authenticationService *service.AuthenticationService,

	internalMessageService *service.InternalMessageService,
	internalMessageCategoryService *service.InternalMessageCategoryService,
	internalMessageRecipientService *service.InternalMessageRecipientService,

	workflowService *service.WorkflowService,

	attendanceService *service.AttendanceService,
) *http.Server {
	cfg := ctx.GetConfig()

	if cfg == nil || cfg.Server == nil || cfg.Server.Rest == nil {
		return nil
	}

	srv, err := rpc.CreateRestServer(cfg, middlewares...)
	if err != nil {
		panic(err)
	}

	adminV1.RegisterAuthenticationServiceHTTPServer(srv, authenticationService)

	adminV1.RegisterInternalMessageServiceHTTPServer(srv, internalMessageService)
	adminV1.RegisterInternalMessageCategoryServiceHTTPServer(srv, internalMessageCategoryService)
	adminV1.RegisterInternalMessageRecipientServiceHTTPServer(srv, internalMessageRecipientService)

	adminV1.RegisterWorkflowServiceHTTPServer(srv, workflowService)

	adminV1.RegisterAttendanceServiceHTTPServer(srv, attendanceService)

	if cfg.GetServer().GetRest().GetEnableSwagger() {
		swaggerUI.RegisterSwaggerUIServerWithOption(
			srv,
			swaggerUI.WithTitle("GoWind OA Admin API"),
			swaggerUI.WithMemoryData(assets.OpenApiData, "yaml"),
		)
	}

	return srv
}
