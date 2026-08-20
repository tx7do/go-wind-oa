package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	appV1 "go-wind-oa/api/gen/go/app/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"

	"go-wind-oa/pkg/middleware/auth"
)

// SealApplicationService 是 app 边端的用印申请转发层（移动端）。
// 列表强制按当前操作者过滤（覆盖 user_id 参数，防越权）。
type SealApplicationService struct {
	appV1.SealApplicationServiceHTTPServer

	log *log.Helper

	sealApplicationServiceClient oaV1.SealApplicationServiceClient
}

func NewSealApplicationService(
	ctx *bootstrap.Context,
	sealApplicationServiceClient oaV1.SealApplicationServiceClient,
) *SealApplicationService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "seal-application/service/app-service"))
	return &SealApplicationService{
		log:                          l,
		sealApplicationServiceClient: sealApplicationServiceClient,
	}
}

func (s *SealApplicationService) SubmitSealApplication(ctx context.Context, req *oaV1.SubmitSealApplicationRequest) (*oaV1.SubmitSealApplicationResponse, error) {
	return s.sealApplicationServiceClient.SubmitSealApplication(ctx, req)
}

func (s *SealApplicationService) ListSealApplications(ctx context.Context, req *oaV1.ListSealApplicationsRequest) (*oaV1.ListSealApplicationsResponse, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	req.UserId = operator.GetUserId()
	return s.sealApplicationServiceClient.ListSealApplications(ctx, req)
}

func (s *SealApplicationService) GetSealApplication(ctx context.Context, req *oaV1.GetSealApplicationRequest) (*oaV1.SealApplication, error) {
	return s.sealApplicationServiceClient.GetSealApplication(ctx, req)
}
