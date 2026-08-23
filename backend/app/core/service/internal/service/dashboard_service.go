package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-oa/app/core/service/internal/data"

	dashboardV1 "go-wind-oa/api/gen/go/dashboard/service/v1"
)

// DashboardService 为 admin 分析页提供只读 OA 聚合统计（gRPC，由 admin BFF 转发）。
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

// GetOverview 返回四张 OA 概览卡的计数。
func (s *DashboardService) GetOverview(ctx context.Context, _ *emptypb.Empty) (*dashboardV1.OaDashboardOverviewResponse, error) {
	instanceCount, err := s.dashboardRepo.CountWorkflowInstances(ctx)
	if err != nil {
		return nil, err
	}
	pendingTaskCount, err := s.dashboardRepo.CountPendingTasks(ctx)
	if err != nil {
		return nil, err
	}
	todayNewInstanceCount, err := s.dashboardRepo.CountTodayNewInstances(ctx)
	if err != nil {
		return nil, err
	}
	todayWorkflowActionCount, err := s.dashboardRepo.CountTodayWorkflowActions(ctx)
	if err != nil {
		return nil, err
	}

	instanceCountVal := uint32(instanceCount)
	pendingTaskCountVal := uint32(pendingTaskCount)
	todayNewInstanceCountVal := uint32(todayNewInstanceCount)
	todayWorkflowActionCountVal := uint32(todayWorkflowActionCount)
	return &dashboardV1.OaDashboardOverviewResponse{
		WorkflowInstanceCount:     &instanceCountVal,
		PendingTaskCount:          &pendingTaskCountVal,
		TodayNewInstanceCount:     &todayNewInstanceCountVal,
		TodayWorkflowActionCount: &todayWorkflowActionCountVal,
	}, nil
}

// GetOaTrend 返回近 days 天每日新增工单数趋势，按日期升序、缺日补零。
func (s *DashboardService) GetOaTrend(ctx context.Context, req *dashboardV1.GetOaTrendRequest) (*dashboardV1.OaTrendResponse, error) {
	days := int(req.GetDays())
	rows, err := s.dashboardRepo.InstanceCreationTrend(ctx, days)
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
	return &dashboardV1.OaTrendResponse{Points: points}, nil
}

// GetOaInstanceStatusDistribution 返回工单按 instance_status 的分布。
func (s *DashboardService) GetOaInstanceStatusDistribution(ctx context.Context, _ *emptypb.Empty) (*dashboardV1.OaDistributionResponse, error) {
	rows, err := s.dashboardRepo.InstanceStatusDistribution(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*dashboardV1.DistributionItem, 0, len(rows))
	for _, r := range rows {
		label, count := r.InstanceStatus, uint32(r.Count)
		items = append(items, &dashboardV1.DistributionItem{
			Label: &label,
			Count: &count,
		})
	}
	return &dashboardV1.OaDistributionResponse{Items: items}, nil
}

// GetOaAttendanceDayResultDistribution 返回考勤记录按 day_result 的分布。
func (s *DashboardService) GetOaAttendanceDayResultDistribution(ctx context.Context, _ *emptypb.Empty) (*dashboardV1.OaDistributionResponse, error) {
	rows, err := s.dashboardRepo.AttendanceDayResultDistribution(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*dashboardV1.DistributionItem, 0, len(rows))
	for _, r := range rows {
		label, count := r.DayResult, uint32(r.Count)
		items = append(items, &dashboardV1.DistributionItem{
			Label: &label,
			Count: &count,
		})
	}
	return &dashboardV1.OaDistributionResponse{Items: items}, nil
}
