package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	appV1 "go-wind-oa/api/gen/go/app/service/v1"
	authenticationV1 "go-wind-oa/api/gen/go/authentication/service/v1"

	"go-wind-oa/pkg/middleware/auth"
)

type AuthenticationService struct {
	appV1.AuthenticationServiceHTTPServer

	authenticationServiceClient authenticationV1.AuthenticationServiceClient

	log *log.Helper
}

func NewAuthenticationService(
	ctx *bootstrap.Context,
	authenticationServiceClient authenticationV1.AuthenticationServiceClient,
) *AuthenticationService {
	return &AuthenticationService{
		log:                         ctx.NewLoggerHelper("authn/service/app-service"),
		authenticationServiceClient: authenticationServiceClient,
	}
}

// Login 登陆
func (s *AuthenticationService) Login(ctx context.Context, req *authenticationV1.LoginRequest) (*authenticationV1.LoginResponse, error) {
	if req == nil {
		return nil, authenticationV1.ErrorBadRequest("invalid request")
	}

	// refresh_token 授权不再从 access claims 取身份（Login 在白名单内，无 claims；
	// 且刷新语义上 access 往往已过期）——身份由 core 从刷新令牌自身解析。
	req.ClientType = trans.Ptr(authenticationV1.ClientType_app)

	return s.authenticationServiceClient.Login(ctx, req)
}

// Logout 登出
func (s *AuthenticationService) Logout(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	return s.authenticationServiceClient.Logout(ctx, &authenticationV1.LogoutRequest{
		ClientType: authenticationV1.ClientType_app,
		UserId:     operator.GetUserId(),
	})
}

// RefreshToken 刷新认证令牌
//
// 刷新发生在访问令牌过期之后，本端点在鉴权白名单内（无 access claims），
// 不得依赖 auth.FromContext 取身份——否则白名单下必然 401。此处仅强制
// ClientType 与 grant_type 后透传，身份由 core 从刷新令牌自身解析。
func (s *AuthenticationService) RefreshToken(ctx context.Context, req *authenticationV1.LoginRequest) (*authenticationV1.LoginResponse, error) {
	if req == nil {
		return nil, authenticationV1.ErrorBadRequest("invalid request")
	}

	req.ClientType = trans.Ptr(authenticationV1.ClientType_app)
	req.GrantType = authenticationV1.GrantType_refresh_token

	return s.authenticationServiceClient.RefreshToken(ctx, req)
}
