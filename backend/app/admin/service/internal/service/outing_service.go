package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	adminV1 "go-wind-oa/api/gen/go/admin/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// OutingService 是 admin 边端的外出申请管理转发层（仅申请查询）。
type OutingService struct {
	adminV1.OutingServiceHTTPServer

	log *log.Helper

	outingServiceClient oaV1.OutingServiceClient
}

func NewOutingService(
	ctx *bootstrap.Context,
	outingServiceClient oaV1.OutingServiceClient,
) *OutingService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "outing/service/admin-service"))
	return &OutingService{
		log:                 l,
		outingServiceClient: outingServiceClient,
	}
}

func (s *OutingService) ListOutingApplications(ctx context.Context, req *oaV1.ListOutingApplicationsRequest) (*oaV1.ListOutingApplicationsResponse, error) {
	return s.outingServiceClient.ListOutingApplications(ctx, req)
}

func (s *OutingService) GetOutingApplication(ctx context.Context, req *oaV1.GetOutingApplicationRequest) (*oaV1.OutingApplication, error) {
	return s.outingServiceClient.GetOutingApplication(ctx, req)
}
