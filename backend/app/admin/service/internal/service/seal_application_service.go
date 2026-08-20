package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	adminV1 "go-wind-oa/api/gen/go/admin/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// SealApplicationService 是 admin 边端的用印申请管理转发层（仅申请查询）。
type SealApplicationService struct {
	adminV1.SealApplicationServiceHTTPServer

	log *log.Helper

	sealApplicationServiceClient oaV1.SealApplicationServiceClient
}

func NewSealApplicationService(
	ctx *bootstrap.Context,
	sealApplicationServiceClient oaV1.SealApplicationServiceClient,
) *SealApplicationService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "seal-application/service/admin-service"))
	return &SealApplicationService{
		log:                          l,
		sealApplicationServiceClient: sealApplicationServiceClient,
	}
}

func (s *SealApplicationService) ListSealApplications(ctx context.Context, req *oaV1.ListSealApplicationsRequest) (*oaV1.ListSealApplicationsResponse, error) {
	return s.sealApplicationServiceClient.ListSealApplications(ctx, req)
}

func (s *SealApplicationService) GetSealApplication(ctx context.Context, req *oaV1.GetSealApplicationRequest) (*oaV1.SealApplication, error) {
	return s.sealApplicationServiceClient.GetSealApplication(ctx, req)
}
