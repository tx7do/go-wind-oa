package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	adminV1 "go-wind-oa/api/gen/go/admin/service/v1"
	dashboardV1 "go-wind-oa/api/gen/go/dashboard/service/v1"
)

// DashboardService 为分析页提供只读聚合统计（HTTP BFF，转发 core DashboardService）。
type DashboardService struct {
	adminV1.DashboardServiceHTTPServer

	log *log.Helper

	dashboardServiceClient dashboardV1.DashboardServiceClient
}

func NewDashboardService(ctx *bootstrap.Context, dashboardServiceClient dashboardV1.DashboardServiceClient) *DashboardService {
	return &DashboardService{
		log:                    ctx.NewLoggerHelper("dashboard/service/admin-service"),
		dashboardServiceClient: dashboardServiceClient,
	}
}

func (s *DashboardService) GetOverview(ctx context.Context, req *emptypb.Empty) (*dashboardV1.DashboardOverviewResponse, error) {
	return s.dashboardServiceClient.GetOverview(ctx, req)
}

func (s *DashboardService) GetLoginTrend(ctx context.Context, req *dashboardV1.GetLoginTrendRequest) (*dashboardV1.LoginTrendResponse, error) {
	return s.dashboardServiceClient.GetLoginTrend(ctx, req)
}

func (s *DashboardService) GetOperationActionDistribution(ctx context.Context, req *emptypb.Empty) (*dashboardV1.ActionDistributionResponse, error) {
	return s.dashboardServiceClient.GetOperationActionDistribution(ctx, req)
}

func (s *DashboardService) GetLoginStatusDistribution(ctx context.Context, req *emptypb.Empty) (*dashboardV1.StatusDistributionResponse, error) {
	return s.dashboardServiceClient.GetLoginStatusDistribution(ctx, req)
}
