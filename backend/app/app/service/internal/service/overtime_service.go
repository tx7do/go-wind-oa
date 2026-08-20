package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	appV1 "go-wind-oa/api/gen/go/app/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"

	"go-wind-oa/pkg/middleware/auth"
)

// OvertimeService 是 app 边端的加班申请转发层（移动端）。
// 列表强制按当前操作者过滤（覆盖 user_id 参数，防越权）。
type OvertimeService struct {
	appV1.OvertimeServiceHTTPServer

	log *log.Helper

	overtimeServiceClient oaV1.OvertimeServiceClient
}

func NewOvertimeService(
	ctx *bootstrap.Context,
	overtimeServiceClient oaV1.OvertimeServiceClient,
) *OvertimeService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "overtime/service/app-service"))
	return &OvertimeService{
		log:                   l,
		overtimeServiceClient: overtimeServiceClient,
	}
}

func (s *OvertimeService) SubmitOvertimeApplication(ctx context.Context, req *oaV1.SubmitOvertimeApplicationRequest) (*oaV1.SubmitOvertimeApplicationResponse, error) {
	return s.overtimeServiceClient.SubmitOvertimeApplication(ctx, req)
}

func (s *OvertimeService) ListOvertimeApplications(ctx context.Context, req *oaV1.ListOvertimeApplicationsRequest) (*oaV1.ListOvertimeApplicationsResponse, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	req.UserId = operator.GetUserId()
	return s.overtimeServiceClient.ListOvertimeApplications(ctx, req)
}

func (s *OvertimeService) GetOvertimeApplication(ctx context.Context, req *oaV1.GetOvertimeApplicationRequest) (*oaV1.OvertimeApplication, error) {
	return s.overtimeServiceClient.GetOvertimeApplication(ctx, req)
}
