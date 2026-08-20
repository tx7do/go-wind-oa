package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	adminV1 "go-wind-oa/api/gen/go/admin/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// OvertimeService 是 admin 边端的加班申请管理转发层（仅申请查询）。
type OvertimeService struct {
	adminV1.OvertimeServiceHTTPServer

	log *log.Helper

	overtimeServiceClient oaV1.OvertimeServiceClient
}

func NewOvertimeService(
	ctx *bootstrap.Context,
	overtimeServiceClient oaV1.OvertimeServiceClient,
) *OvertimeService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "overtime/service/admin-service"))
	return &OvertimeService{
		log:                   l,
		overtimeServiceClient: overtimeServiceClient,
	}
}

func (s *OvertimeService) ListOvertimeApplications(ctx context.Context, req *oaV1.ListOvertimeApplicationsRequest) (*oaV1.ListOvertimeApplicationsResponse, error) {
	return s.overtimeServiceClient.ListOvertimeApplications(ctx, req)
}

func (s *OvertimeService) GetOvertimeApplication(ctx context.Context, req *oaV1.GetOvertimeApplicationRequest) (*oaV1.OvertimeApplication, error) {
	return s.overtimeServiceClient.GetOvertimeApplication(ctx, req)
}
