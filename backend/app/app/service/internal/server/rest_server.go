package server

import (
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

	"go-wind-oa/pkg/middleware/auth"
	entmiddleware "go-wind-oa/pkg/middleware/ent"
)

// NewRestMiddleware 创建中间件
//
// OA app-service 的 REST 中間件鏈（與 admin-service 同構）：
//   - logging：記錄調用
//   - auth.Server + authz.Server（白名單匹配）：非白名單請求注入 OperatorMetadata
//   - ent.Server()：按注入的身份構建帶租戶作用域的 viewer
//
// auth 必須在 ent 之前：順序顛倒則 ent 總以 md==nil 兜底 SystemViewer，租戶隔離失效。
// 白名單只含 Login（鑑權雞生蛋）。工作流端點均需鑑權，不在白名單。
//
// OA app-service 無匿名公開內容，故不需要 cms 的 TenantResolver——所有請求
// 都經 auth 注入身份後由 ent 構建 UserViewer。
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
	// 可构建 UserViewer；白名单请求（登录）md==nil 但在白名单内，兜底 SystemViewer。
	ms = append(ms, entmiddleware.Server())

	return ms
}

// NewRestServer new an REST server.
//
// OA app-service 的 HTTP 邊端僅註冊兩類服務：
//   - AuthenticationService（鑑權：登錄/登出/令牌刷新，轉發 core gRPC）
//   - WorkflowService（工作流申請/審批/任務查詢，轉發 core gRPC）
//   - InternalMessageService（站內信查詢，轉發 core gRPC）
func NewRestServer(
	ctx *bootstrap.Context,

	middlewares []middleware.Middleware,

	authenticationService *service.AuthenticationService,

	workflowService *service.WorkflowService,

	internalMessageService *service.InternalMessageService,
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
	appV1.RegisterInternalMessageServiceHTTPServer(srv, internalMessageService)

	if cfg.GetServer().GetRest().GetEnableSwagger() {
		swaggerUI.RegisterSwaggerUIServerWithOption(
			srv,
			swaggerUI.WithTitle("GoWind OA App API"),
			swaggerUI.WithMemoryData(assets.OpenApiData, "yaml"),
		)
	}

	return srv
}
