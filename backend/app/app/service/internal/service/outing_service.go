package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	appV1 "go-wind-oa/api/gen/go/app/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"

	"go-wind-oa/pkg/middleware/auth"
)

// OutingService 是 app 边端的外出申请转发层（移动端）。
// 列表强制按当前操作者过滤（覆盖 user_id 参数，防越权）。
type OutingService struct {
	appV1.OutingServiceHTTPServer

	log *log.Helper

	outingServiceClient oaV1.OutingServiceClient
}

func NewOutingService(
	ctx *bootstrap.Context,
	outingServiceClient oaV1.OutingServiceClient,
) *OutingService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "outing/service/app-service"))
	return &OutingService{
		log:                 l,
		outingServiceClient: outingServiceClient,
	}
}

func (s *OutingService) SubmitOutingApplication(ctx context.Context, req *oaV1.SubmitOutingApplicationRequest) (*oaV1.SubmitOutingApplicationResponse, error) {
	return s.outingServiceClient.SubmitOutingApplication(ctx, req)
}

func (s *OutingService) ListOutingApplications(ctx context.Context, req *oaV1.ListOutingApplicationsRequest) (*oaV1.ListOutingApplicationsResponse, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	req.UserId = operator.GetUserId()
	return s.outingServiceClient.ListOutingApplications(ctx, req)
}

func (s *OutingService) GetOutingApplication(ctx context.Context, req *oaV1.GetOutingApplicationRequest) (*oaV1.OutingApplication, error) {
	return s.outingServiceClient.GetOutingApplication(ctx, req)
}
