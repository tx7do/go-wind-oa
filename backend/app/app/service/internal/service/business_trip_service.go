package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	appV1 "go-wind-oa/api/gen/go/app/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"

	"go-wind-oa/pkg/middleware/auth"
)

// BusinessTripService 是 app 边端的出差申请转发层（移动端）。
// 列表强制按当前操作者过滤（覆盖 user_id 参数，防越权）。
type BusinessTripService struct {
	appV1.BusinessTripServiceHTTPServer

	log *log.Helper

	businessTripServiceClient oaV1.BusinessTripServiceClient
}

func NewBusinessTripService(
	ctx *bootstrap.Context,
	businessTripServiceClient oaV1.BusinessTripServiceClient,
) *BusinessTripService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "business-trip/service/app-service"))
	return &BusinessTripService{
		log:                       l,
		businessTripServiceClient: businessTripServiceClient,
	}
}

func (s *BusinessTripService) SubmitBusinessTripApplication(ctx context.Context, req *oaV1.SubmitBusinessTripApplicationRequest) (*oaV1.SubmitBusinessTripApplicationResponse, error) {
	return s.businessTripServiceClient.SubmitBusinessTripApplication(ctx, req)
}

func (s *BusinessTripService) ListBusinessTripApplications(ctx context.Context, req *oaV1.ListBusinessTripApplicationsRequest) (*oaV1.ListBusinessTripApplicationsResponse, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	// app 端强制按本人查，忽略/覆盖客户端传入的 user_id。
	req.UserId = operator.GetUserId()
	return s.businessTripServiceClient.ListBusinessTripApplications(ctx, req)
}

func (s *BusinessTripService) GetBusinessTripApplication(ctx context.Context, req *oaV1.GetBusinessTripApplicationRequest) (*oaV1.BusinessTripApplication, error) {
	return s.businessTripServiceClient.GetBusinessTripApplication(ctx, req)
}
