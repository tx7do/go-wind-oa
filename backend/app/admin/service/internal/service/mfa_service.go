package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	adminV1 "go-wind-oa/api/gen/go/admin/service/v1"
	authenticationV1 "go-wind-oa/api/gen/go/authentication/service/v1"

	"go-wind-oa/pkg/middleware/auth"
)

// MfaService 为 MFA（多因素认证）提供 HTTP 桥接（转发 core MFAService）。
//
// 管理侧 RPC 需登录态，走正常 auth+authz 中间件；登录挑战侧 RPC
// （VerifyMFAChallenge）免鉴权，在 rest_server 白名单中。
// 管理端救援重置（DisableMFA 指定他人 user_id）的平台管理员校验在本层完成
// ——这里才有完整 UserTokenPayload（IsPlatformAdmin），core 信任 BFF 的裁决。
type MfaService struct {
	adminV1.MfaServiceHTTPServer

	log *log.Helper

	mfaServiceClient authenticationV1.MFAServiceClient
}

func NewMfaService(ctx *bootstrap.Context, mfaServiceClient authenticationV1.MFAServiceClient) *MfaService {
	return &MfaService{
		log:              ctx.NewLoggerHelper("mfa/service/admin-service"),
		mfaServiceClient: mfaServiceClient,
	}
}

func (s *MfaService) GetMFAStatus(ctx context.Context, req *authenticationV1.GetMFAStatusRequest) (*authenticationV1.GetMFAStatusResponse, error) {
	return s.mfaServiceClient.GetMFAStatus(ctx, req)
}

func (s *MfaService) ListEnrolledMethods(ctx context.Context, req *authenticationV1.ListEnrolledMethodsRequest) (*authenticationV1.ListEnrolledMethodsResponse, error) {
	return s.mfaServiceClient.ListEnrolledMethods(ctx, req)
}

func (s *MfaService) StartEnrollMethod(ctx context.Context, req *authenticationV1.StartEnrollMethodRequest) (*authenticationV1.StartEnrollMethodResponse, error) {
	return s.mfaServiceClient.StartEnrollMethod(ctx, req)
}

func (s *MfaService) ConfirmEnrollMethod(ctx context.Context, req *authenticationV1.ConfirmEnrollMethodRequest) (*authenticationV1.ConfirmEnrollMethodResponse, error) {
	return s.mfaServiceClient.ConfirmEnrollMethod(ctx, req)
}

// DisableMFA 禁用/移除 MFA 凭证。
// 指定他人 user_id 的管理端救援重置仅平台管理员允许，校验在本层完成。
func (s *MfaService) DisableMFA(ctx context.Context, req *authenticationV1.DisableMFARequest) (*emptypb.Empty, error) {
	if target := req.GetUserId(); target != 0 {
		operator, err := auth.FromContext(ctx)
		if err != nil {
			return nil, err
		}
		if target != operator.GetUserId() && !operator.GetIsPlatformAdmin() {
			return nil, authenticationV1.ErrorForbidden("only platform admin can reset mfa for others")
		}
	}
	return s.mfaServiceClient.DisableMFA(ctx, req)
}

func (s *MfaService) RevokeMFADevice(ctx context.Context, req *authenticationV1.RevokeMFADeviceRequest) (*emptypb.Empty, error) {
	return s.mfaServiceClient.RevokeMFADevice(ctx, req)
}

// VerifyMFAChallenge 验证登录 MFA 挑战。通过则返回 LoginResponse（含真 access_token）。
// 免鉴权：登录流程在密码校验通过、待二次验证阶段调用。
func (s *MfaService) VerifyMFAChallenge(ctx context.Context, req *authenticationV1.VerifyMFAChallengeRequest) (*authenticationV1.LoginResponse, error) {
	return s.mfaServiceClient.VerifyMFAChallenge(ctx, req)
}
