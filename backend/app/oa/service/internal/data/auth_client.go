package data

import (
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/kratos-bootstrap/rpc"

	authenticationV1 "go-wind-cms/api/gen/go/authentication/service/v1"
	cmsServiceID "go-wind-cms/pkg/serviceid"
)

// NewAuthenticationServiceClient 创建对 go-wind-cms 站内鉴权组件的 gRPC 客户端。
//
// 砖块式复用：OA 不自建账号体系，登录/登出/验证码/令牌刷新均经此客户端转发
// 至 cms 的 admin-service AuthenticationService（cms authentication.proto +
// i_authentication.proto 的 admin 端 HTTP 包装）。
//
// 构造方式与 go-wind-cms/app/admin/service/internal/data/data.go 中的
// NewAuthenticationServiceClient 同构：经服务发现定位 admin-service（区别于
// NewNotificationServiceClient 定位的 core-service），由 rpc.CreateGrpcClient
// 取底层 grpc.ClientConn，再由生成客户端包装。
//
// 注意：鉴权操作符（user_id / client_type / jti）由 cms auth 中间件从 JWT
// 注入 OperatorMetadata（UserTokenPayload），OA 转发层在 Logout/RefreshToken
// 中经 auth.FromContext 取出后回填请求，与 cms admin-service 同款流程。
func NewAuthenticationServiceClient(ctx *bootstrap.Context, r registry.Discovery) authenticationV1.AuthenticationServiceClient {
	cli, err := rpc.CreateGrpcClient(ctx.Context(), r, cmsServiceID.NewDiscoveryName(cmsServiceID.AdminService), ctx.GetConfig())
	if err != nil {
		return nil
	}
	return authenticationV1.NewAuthenticationServiceClient(cli)
}
