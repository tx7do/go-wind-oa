package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	adminV1 "go-wind-oa/api/gen/go/admin/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// BusinessTripService 是 admin 边端的出差申请管理转发层（仅申请查询）。
type BusinessTripService struct {
	adminV1.BusinessTripServiceHTTPServer

	log *log.Helper

	businessTripServiceClient oaV1.BusinessTripServiceClient
}

func NewBusinessTripService(
	ctx *bootstrap.Context,
	businessTripServiceClient oaV1.BusinessTripServiceClient,
) *BusinessTripService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "business-trip/service/admin-service"))
	return &BusinessTripService{
		log:                       l,
		businessTripServiceClient: businessTripServiceClient,
	}
}

func (s *BusinessTripService) ListBusinessTripApplications(ctx context.Context, req *oaV1.ListBusinessTripApplicationsRequest) (*oaV1.ListBusinessTripApplicationsResponse, error) {
	return s.businessTripServiceClient.ListBusinessTripApplications(ctx, req)
}

func (s *BusinessTripService) GetBusinessTripApplication(ctx context.Context, req *oaV1.GetBusinessTripApplicationRequest) (*oaV1.BusinessTripApplication, error) {
	return s.businessTripServiceClient.GetBusinessTripApplication(ctx, req)
}
