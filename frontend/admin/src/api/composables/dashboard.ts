import type {
  dashboardservicev1_OaDashboardOverviewResponse as OaDashboardOverviewResponse,
  dashboardservicev1_OaTrendResponse as OaTrendResponse,
  dashboardservicev1_OaDistributionResponse as OaDistributionResponse,
} from "@/api/generated/admin/service/v1";
import { useQuery, type UseQueryOptions } from "@tanstack/vue-query";
import { apiClient } from "@/api/client";

// =====================================================
// OA 分析概览（只读聚合统计：工作流工单计数 / 审批趋势 / 状态·考勤分布）
// =====================================================

// ------------------------------
// 1. OA 概览统计卡片（工单总数 / 待办任务数 / 今日新增工单 / 今日审批操作数）
// ------------------------------
export function useOaDashboardOverview(
  options?: UseQueryOptions<OaDashboardOverviewResponse, Error>
) {
  return useQuery({
    queryKey: ["oaDashboardOverview"],
    queryFn: () => apiClient.dashboardService.GetOverview({}),
    ...options,
  });
}

// ------------------------------
// 2. 近 N 天每日新增工单趋势
// ------------------------------
export function useOaTrend(
  days: number,
  options?: UseQueryOptions<OaTrendResponse, Error>
) {
  return useQuery({
    queryKey: ["oaTrend", days],
    queryFn: () => apiClient.dashboardService.GetOaTrend({ days }),
    ...options,
  });
}

// ------------------------------
// 3. 工单按 instance_status 分布
// ------------------------------
export function useOaInstanceStatusDistribution(
  options?: UseQueryOptions<OaDistributionResponse, Error>
) {
  return useQuery({
    queryKey: ["oaInstanceStatusDistribution"],
    queryFn: () =>
      apiClient.dashboardService.GetOaInstanceStatusDistribution({}),
    ...options,
  });
}

// ------------------------------
// 4. 考勤记录按 day_result 分布
// ------------------------------
export function useOaAttendanceDayResultDistribution(
  options?: UseQueryOptions<OaDistributionResponse, Error>
) {
  return useQuery({
    queryKey: ["oaAttendanceDayResultDistribution"],
    queryFn: () =>
      apiClient.dashboardService.GetOaAttendanceDayResultDistribution({}),
    ...options,
  });
}
