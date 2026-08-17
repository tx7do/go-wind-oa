package server

import (
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport/http"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/kratos-bootstrap/rpc"

	"go-wind-cms/pkg/middleware/auth"
	entmiddleware "go-wind-cms/pkg/middleware/ent"

	oav1 "go-wind-oa/api/gen/go/oa/v1"
	"go-wind-oa/app/oa/service/internal/service"
)

// NewRestMiddleware 构建 OA HTTP 服务的中间件链。
//
// 砖块复用 go-wind-cms：顺序与 cms/app/admin/service/internal/server/rest_server.go
// 完全一致。关键顺序约束：
//  1. logging.Server 先于一切，便于排障；
//  2. auth.Server 在 entmiddleware.Server 之前：auth 对非白名单请求注入
//     OperatorMetadata，随后 entmiddleware 据此构建带租户作用域的 UserViewer，
//     供 TenantPrivacy 策略做行级租户隔离。顺序颠倒将导致 ent 中间件以 md==nil
//     兜底为 SystemViewer，租户隔离失效。
//
// 白名单：与 cms admin-service 同构，含鉴权服务的三个预认证端点
// （Login / GenerateCaptcha / VerifyCaptcha——获取验证码时尚无 token，鸡生蛋问题）。
// Logout / RefreshToken 不在白名单——需 JWT，由 auth.FromContext 取 OperatorMetadata。
// 工作流端点（/admin/v1/oa/...）均不在白名单——必须携带有效 JWT，且经 ent viewer
// 做行级租户隔离。
//
// 注意：auth 与 ent 中间件都来自 go-wind-cms/pkg/middleware，OA 不重写，避免在
// 租户隔离这一安全关键路径上分叉。
func NewRestMiddleware(ctx *bootstrap.Context) []middleware.Middleware {
	var ms []middleware.Middleware
	ms = append(ms, recovery.Recovery(), logging.Server(ctx.GetLogger()))

	// add white list for authentication.
	// Login / GenerateCaptcha / VerifyCaptcha 必须在白名单中——获取验证码与首次
	// 登录时用户尚无 token（与 cms admin-service 同款鸡生蛋处理）。
	rpc.AddWhiteList(
		oav1.OperationAuthenticationServiceLogin,
		oav1.OperationAuthenticationServiceGenerateCaptcha,
		oav1.OperationAuthenticationServiceVerifyCaptcha,
	)

	// 鉴权：对非白名单请求注入 OperatorMetadata。
	ms = append(ms, selector.Server(
		auth.Server(
			auth.WithInjectMetadata(true),
			auth.WithInjectEnt(true),
		),
	).Match(rpc.NewRestWhiteListMatcher()).Build())

	// ent viewer：依据上一步注入的 OperatorMetadata 构建 UserViewer。
	ms = append(ms, entmiddleware.Server())

	return ms
}

// NewRestServer 创建 OA REST 服务并注册生成的 HTTP handler。
//
// 注册两个业务服务（均由 kratos protoc-gen-go-http 生成的 Register*HTTPServer
// 桩绑定到对应路径）：
//   - WorkflowService → /admin/v1/oa/... 工作流审批路径；
//   - AuthenticationService → /admin/v1/login | /logout | /refresh-token |
//     /captcha | /captcha/verify 鉴权转发路径。
//
// 鉴权端点中，Login/GenerateCaptcha/VerifyCaptcha 经白名单放行（见
// NewRestMiddleware），Logout/RefreshToken 经 JWT 鉴权后由
// auth.FromContext 取 OperatorMetadata 回填——与 cms admin-service 同款流程。
func NewRestServer(
	ctx *bootstrap.Context,
	middlewares []middleware.Middleware,
	workflowService *service.WorkflowService,
	authenticationService *service.AuthenticationService,
) *http.Server {
	cfg := ctx.GetConfig()
	if cfg == nil || cfg.Server == nil || cfg.Server.Rest == nil {
		return nil
	}

	srv, err := rpc.CreateRestServer(cfg, middlewares...)
	if err != nil {
		panic(err)
	}

	oav1.RegisterWorkflowServiceHTTPServer(srv, workflowService)
	oav1.RegisterAuthenticationServiceHTTPServer(srv, authenticationService)

	return srv
}
