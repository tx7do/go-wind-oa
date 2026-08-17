package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"go-wind-cms/pkg/middleware/auth"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	authenticationV1 "go-wind-cms/api/gen/go/authentication/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/v1"
)

// AuthenticationService 实现 OA 鉴权服务的 HTTP server 接口。
//
// 砖块式复用：OA 不自建账号体系，登录/登出/验证码/令牌刷新均经此服务转发至
// go-wind-cms 的 admin-service AuthenticationService（cms
// authentication.proto + i_authentication.proto 的 admin 端 HTTP 包装）。
//
// 类型翻译：oa.v1 与 authentication.service.v1 的对应消息（LoginRequest /
// LoginResponse / VerifyCaptchaRequest / VerifyCaptchaResponse /
// GenerateCaptchaResponse）字段编号、类型、oneof 结构逐字段对齐，wire 格式
// 完全一致。故采用 proto.Marshal(oa) → proto.Unmarshal(cms) 做跨包翻译——
// 这是 wire-compatible proto 跨 Go 类型的标准做法，完整保留 oneof
// （Identifier: username/email/mobile）与所有 optional 字段，无需逐字段
// 手赋。
//
// 鉴权操作符回填：Logout / RefreshToken 的 UserId / ClientType / Jti 由
// cms auth 中间件从 JWT 注入 OperatorMetadata（UserTokenPayload），OA
// 转发层经 auth.FromContext 取出后回填到 cms 请求——与 cms admin-service
// 同款流程（见 cms authentication_service.go Logout/RefreshToken）。
// Login 的 ClientType 固定为 admin（与 cms 同）。
//
// 这三步流程（翻译 → 回填操作符 → 转发）与 cms admin-service 的对应方法
// 在语义上逐行等价，差异仅在多了一层 oa↔cms 的 proto 翻译。
type AuthenticationService struct {
	oaV1.UnimplementedAuthenticationServiceServer

	authenticationServiceClient authenticationV1.AuthenticationServiceClient
	log                         *log.Helper
}

// NewAuthenticationService 构造 OA 鉴权转发服务。
func NewAuthenticationService(
	ctx *bootstrap.Context,
	authenticationServiceClient authenticationV1.AuthenticationServiceClient,
) *AuthenticationService {
	return &AuthenticationService{
		authenticationServiceClient: authenticationServiceClient,
		log:                         ctx.NewLoggerHelper("auth/service/oa-service"),
	}
}

// Login 登录。转发至 cms AuthenticationService.Login。
//
// 流程对齐 cms admin-service Login：
//   - 翻译 oa 请求为 cms 请求；
//   - 强制 ClientType = admin；
//   - refresh_token 授权：从 OperatorMetadata 取 Jti / UserId 回填；
//     password 授权：cms 服务端自行校验验证码（经 HTTP Header 透传，见
//     cms verifyLoginCaptcha）—— OA 转发层不介入；
//   - 转发，翻译响应回 oa 类型。
func (s *AuthenticationService) Login(ctx context.Context, req *oaV1.LoginRequest) (*oaV1.LoginResponse, error) {
	if req == nil {
		return nil, oaV1.ErrorBadRequest("invalid request")
	}

	if s.authenticationServiceClient == nil {
		return nil, oaV1.ErrorInternalError("authentication service not wired")
	}

	cmsReq, err := translateLoginRequest(req)
	if err != nil {
		s.log.Errorf("translate login request failed: %s", err.Error())
		return nil, oaV1.ErrorInternalError("translate login request failed")
	}

	// 强制 admin 客户端类型（与 cms admin-service 同）。
	cmsReq.ClientType = trans.Ptr(authenticationV1.ClientType_admin)

	// refresh_token 授权需回填操作符（与 cms 同款流程）。
	if cmsReq.GetGrantType() == authenticationV1.GrantType_refresh_token {
		operator, err := auth.FromContext(ctx)
		if err != nil {
			return nil, err
		}
		cmsReq.Jti = operator.Jti
		cmsReq.UserId = trans.Ptr(operator.GetUserId())
	}

	cmsResp, err := s.authenticationServiceClient.Login(ctx, cmsReq)
	if err != nil {
		return nil, err
	}

	return translateLoginResponse(cmsResp)
}

// Logout 登出。转发至 cms AuthenticationService.Logout。
//
// 流程对齐 cms admin-service Logout：
//   - 从 OperatorMetadata 取 UserId / ClientType 构建 cms LogoutRequest；
//   - 转发。
func (s *AuthenticationService) Logout(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	if s.authenticationServiceClient == nil {
		return nil, oaV1.ErrorInternalError("authentication service not wired")
	}

	return s.authenticationServiceClient.Logout(ctx, &authenticationV1.LogoutRequest{
		ClientType: authenticationV1.ClientType_admin,
		UserId:     operator.GetUserId(),
	})
}

// RefreshToken 刷新令牌。转发至 cms AuthenticationService.RefreshToken。
//
// 流程对齐 cms admin-service RefreshToken：
//   - 翻译 oa 请求为 cms 请求；
//   - 从 OperatorMetadata 回填 ClientType / UserId / Jti；
//   - 转发，翻译响应回 oa 类型。
func (s *AuthenticationService) RefreshToken(ctx context.Context, req *oaV1.LoginRequest) (*oaV1.LoginResponse, error) {
	if req == nil {
		return nil, oaV1.ErrorBadRequest("invalid request")
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	if s.authenticationServiceClient == nil {
		return nil, oaV1.ErrorInternalError("authentication service not wired")
	}

	cmsReq, err := translateLoginRequest(req)
	if err != nil {
		s.log.Errorf("translate login request failed: %s", err.Error())
		return nil, oaV1.ErrorInternalError("translate login request failed")
	}

	cmsReq.ClientType = trans.Ptr(authenticationV1.ClientType_admin)
	cmsReq.UserId = trans.Ptr(operator.GetUserId())
	cmsReq.Jti = operator.Jti

	cmsResp, err := s.authenticationServiceClient.RefreshToken(ctx, cmsReq)
	if err != nil {
		return nil, err
	}

	return translateLoginResponse(cmsResp)
}

// GenerateCaptcha 生成验证码。转发至 cms AuthenticationService.GenerateCaptcha。
//
// 与 cms admin-service 不同：cms 自建 captchaClient（Redis 落盘）本地生成；
// OA 不自建验证码基础设施，直接转发至 cms gRPC 客户端（cms client 接口
// 已含 GenerateCaptcha），由 cms 服务端生成并落盘。
func (s *AuthenticationService) GenerateCaptcha(ctx context.Context, _ *emptypb.Empty) (*oaV1.GenerateCaptchaResponse, error) {
	if s.authenticationServiceClient == nil {
		return nil, oaV1.ErrorInternalError("authentication service not wired")
	}

	cmsResp, err := s.authenticationServiceClient.GenerateCaptcha(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}

	return translateGenerateCaptchaResponse(cmsResp)
}

// VerifyCaptcha 验证验证码。转发至 cms AuthenticationService.VerifyCaptcha。
func (s *AuthenticationService) VerifyCaptcha(ctx context.Context, req *oaV1.VerifyCaptchaRequest) (*oaV1.VerifyCaptchaResponse, error) {
	if req == nil {
		return nil, oaV1.ErrorBadRequest("invalid request")
	}

	if s.authenticationServiceClient == nil {
		return nil, oaV1.ErrorInternalError("authentication service not wired")
	}

	// 翻译 oa 请求为 cms 请求（wire-compatible，proto.Marshal/Unmarshal）。
	cmsReqBytes, err := proto.Marshal(req)
	if err != nil {
		s.log.Errorf("marshal verify captcha request failed: %s", err.Error())
		return nil, oaV1.ErrorInternalError("translate verify captcha request failed")
	}
	cmsReq := &authenticationV1.VerifyCaptchaRequest{}
	if err := proto.Unmarshal(cmsReqBytes, cmsReq); err != nil {
		s.log.Errorf("unmarshal verify captcha request failed: %s", err.Error())
		return nil, oaV1.ErrorInternalError("translate verify captcha request failed")
	}

	cmsResp, err := s.authenticationServiceClient.VerifyCaptcha(ctx, cmsReq)
	if err != nil {
		return nil, err
	}

	// 翻译 cms 响应为 oa 响应。
	cmsRespBytes, err := proto.Marshal(cmsResp)
	if err != nil {
		s.log.Errorf("marshal verify captcha response failed: %s", err.Error())
		return nil, oaV1.ErrorInternalError("translate verify captcha response failed")
	}
	oaResp := &oaV1.VerifyCaptchaResponse{}
	if err := proto.Unmarshal(cmsRespBytes, oaResp); err != nil {
		s.log.Errorf("unmarshal verify captcha response failed: %s", err.Error())
		return nil, oaV1.ErrorInternalError("translate verify captcha response failed")
	}

	return oaResp, nil
}

// ---------------------------------------------------------------------------
// oa↔cms proto 翻译辅助。
//
// oa.v1 与 authentication.service.v1 的对应消息 wire 格式逐字段一致（字段
// 编号、类型、oneof 结构、json_name 均对齐），故用 proto.Marshal/Unmarshal
// 做跨 Go 类型翻译。这是 wire-compatible proto 跨包翻译的标准做法，完整
// 保留 oneof（LoginRequest.Identifier）与所有 optional 字段。
// ---------------------------------------------------------------------------

func translateLoginRequest(oaReq *oaV1.LoginRequest) (*authenticationV1.LoginRequest, error) {
	bytes, err := proto.Marshal(oaReq)
	if err != nil {
		return nil, err
	}
	cmsReq := &authenticationV1.LoginRequest{}
	if err := proto.Unmarshal(bytes, cmsReq); err != nil {
		return nil, err
	}
	return cmsReq, nil
}

func translateLoginResponse(cmsResp *authenticationV1.LoginResponse) (*oaV1.LoginResponse, error) {
	if cmsResp == nil {
		return nil, nil
	}
	bytes, err := proto.Marshal(cmsResp)
	if err != nil {
		return nil, err
	}
	oaResp := &oaV1.LoginResponse{}
	if err := proto.Unmarshal(bytes, oaResp); err != nil {
		return nil, err
	}
	return oaResp, nil
}

func translateGenerateCaptchaResponse(cmsResp *authenticationV1.GenerateCaptchaResponse) (*oaV1.GenerateCaptchaResponse, error) {
	if cmsResp == nil {
		return nil, nil
	}
	bytes, err := proto.Marshal(cmsResp)
	if err != nil {
		return nil, err
	}
	oaResp := &oaV1.GenerateCaptchaResponse{}
	if err := proto.Unmarshal(bytes, oaResp); err != nil {
		return nil, err
	}
	return oaResp, nil
}
