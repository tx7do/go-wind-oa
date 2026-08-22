package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-oa/app/core/service/internal/data"

	dashboardV1 "go-wind-oa/api/gen/go/dashboard/service/v1"
)

// DashboardService 为 admin 分析页提供只读聚合统计（gRPC，由 admin BFF 转发）。
// 多租户隔离由 ent Policy + viewer 元数据自动完成。
type DashboardService struct {
	dashboardV1.UnimplementedDashboardServiceServer

	log *log.Helper

	dashboardRepo *data.DashboardRepo
}

func NewDashboardService(
	ctx *bootstrap.Context,
	dashboardRepo *data.DashboardRepo,
) *DashboardService {
	return &DashboardService{
		log:           ctx.NewLoggerHelper("dashboard/service/core-service"),
		dashboardRepo: dashboardRepo,
	}
}

// GetOverview 返回四张概览卡的计数。
func (s *DashboardService) GetOverview(ctx context.Context, _ *emptypb.Empty) (*dashboardV1.DashboardOverviewResponse, error) {
	userCount, err := s.dashboardRepo.CountActiveUsers(ctx)
	if err != nil {
		return nil, err
	}
	roleCount, err := s.dashboardRepo.CountRoles(ctx)
	if err != nil {
		return nil, err
	}
	todayLoginCount, err := s.dashboardRepo.CountTodayLogins(ctx)
	if err != nil {
		return nil, err
	}
	todayOperationCount, err := s.dashboardRepo.CountTodayOperations(ctx)
	if err != nil {
		return nil, err
	}

	userCountVal, roleCountVal := uint32(userCount), uint32(roleCount)
	todayLoginCountVal, todayOperationCountVal := uint32(todayLoginCount), uint32(todayOperationCount)
	return &dashboardV1.DashboardOverviewResponse{
		UserCount:           &userCountVal,
		RoleCount:           &roleCountVal,
		TodayLoginCount:     &todayLoginCountVal,
		TodayOperationCount: &todayOperationCountVal,
	}, nil
}

// GetLoginTrend 返回近 days 天每日登录次数趋势，按日期升序、缺日补零。
func (s *DashboardService) GetLoginTrend(ctx context.Context, req *dashboardV1.GetLoginTrendRequest) (*dashboardV1.LoginTrendResponse, error) {
	days := int(req.GetDays())
	rows, err := s.dashboardRepo.LoginTrend(ctx, days)
	if err != nil {
		return nil, err
	}

	points := make([]*dashboardV1.TrendPoint, 0, len(rows))
	for _, r := range rows {
		date, count := r.Date, uint32(r.Count)
		points = append(points, &dashboardV1.TrendPoint{
			Date:  &date,
			Count: &count,
		})
	}
	return &dashboardV1.LoginTrendResponse{Points: points}, nil
}

// GetOperationActionDistribution 返回操作审计按 action 的分布。
func (s *DashboardService) GetOperationActionDistribution(ctx context.Context, _ *emptypb.Empty) (*dashboardV1.ActionDistributionResponse, error) {
	rows, err := s.dashboardRepo.OperationActionDistribution(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*dashboardV1.DistributionItem, 0, len(rows))
	for _, r := range rows {
		label, count := r.Action, uint32(r.Count)
		items = append(items, &dashboardV1.DistributionItem{
			Label: &label,
			Count: &count,
		})
	}
	return &dashboardV1.ActionDistributionResponse{Items: items}, nil
}

// GetLoginStatusDistribution 返回登录审计按 status 的分布。
func (s *DashboardService) GetLoginStatusDistribution(ctx context.Context, _ *emptypb.Empty) (*dashboardV1.StatusDistributionResponse, error) {
	rows, err := s.dashboardRepo.LoginStatusDistribution(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*dashboardV1.DistributionItem, 0, len(rows))
	for _, r := range rows {
		label, count := r.Status, uint32(r.Count)
		items = append(items, &dashboardV1.DistributionItem{
			Label: &label,
			Count: &count,
		})
	}
	return &dashboardV1.StatusDistributionResponse{Items: items}, nil
}
