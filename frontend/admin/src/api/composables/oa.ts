/**
 * OA 工作流定义管理 composable。
 *
 * 对齐 cms internal-message.ts 模式：useQuery/fetchQuery 包列表查询，
 * useMutation 包创建。类型来自 @/api/generated/admin/service/v1（由
 * buf.admin.typescript.gen.yaml 生成）。
 *
 * 注意：列表查询的 queryKey 为 ["listWorkflowDefinitions", query]，创建成功后
 * 由 ProPage 的 refetch 触发列表刷新（与 cms admin 各 feature 一致，不在
 * mutation 内显式 invalidate）。
 */
import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  oaservicev1_ListWorkflowDefinitionResponse,
  oaservicev1_CreateWorkflowDefinitionRequest,
  oaservicev1_UpdateWorkflowDefinitionRequest,
  oaservicev1_WorkflowDefinition,
  oaservicev1_WorkflowDefinition_DefinitionStatus,
  oaservicev1_ListLeaveTypesResponse,
  oaservicev1_CreateLeaveTypeRequest,
  oaservicev1_LeaveType,
  oaservicev1_GrantLeaveBalanceRequest,
  oaservicev1_ListLeaveBalancesRequest,
  oaservicev1_ListLeaveBalancesResponse,
  oaservicev1_ListLeaveApplicationsRequest,
  oaservicev1_ListLeaveApplicationsResponse,
  oaservicev1_ListExpenseApplicationsRequest,
  oaservicev1_ListExpenseApplicationsResponse,
  oaservicev1_ListAttendanceRecordsRequest,
  oaservicev1_ListAttendanceRecordsResponse,
  oaservicev1_AttendanceSetting,
  oaservicev1_RunDailySettlementRequest,
  oaservicev1_RunDailySettlementResponse,
} from "@/api/generated/admin/service/v1";
import { type PaginationQuery } from "@/core/transport/rest";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";

// ==============================
// 流程定义列表
// ==============================
export function useListWorkflowDefinitions(
  query: PaginationQuery,
  options?: Omit<UseQueryOptions<oaservicev1_ListWorkflowDefinitionResponse, Error>, "queryKey">
) {
  return useQuery({
    queryKey: ["listWorkflowDefinitions", query],
    queryFn: () =>
      apiClient.workflowService.ListWorkflowDefinition(query.toRawParams()),
    ...options,
  });
}

export async function fetchListWorkflowDefinitions(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listWorkflowDefinitions", params],
    queryFn: () =>
      apiClient.workflowService.ListWorkflowDefinition(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

// ==============================
// 流程定义详情
// ==============================
export async function fetchWorkflowDefinition(id: number) {
  return queryClient.fetchQuery({
    queryKey: ["workflowDefinition", id],
    queryFn: () =>
      apiClient.workflowService.GetWorkflowDefinition({
        id,
      }),
    staleTime: 0,
    retry: 0,
  });
}

// ==============================
// 创建流程定义
// ==============================
export function useCreateWorkflowDefinition(
  options?: UseMutationOptions<
    oaservicev1_WorkflowDefinition,
    Error,
    oaservicev1_CreateWorkflowDefinitionRequest
  >
) {
  return useMutation({
    mutationFn: (req) => apiClient.workflowService.CreateWorkflowDefinition(req),
    ...options,
  });
}

// ==============================
// 启用/禁用流程定义
// ==============================
export function useUpdateWorkflowDefinitionStatus(
  options?: UseMutationOptions<
    oaservicev1_WorkflowDefinition,
    Error,
    { id: number; status: oaservicev1_WorkflowDefinition_DefinitionStatus }
  >
) {
  return useMutation({
    mutationFn: ({ id, status }) => {
      const req: oaservicev1_UpdateWorkflowDefinitionRequest = {
        id,
        data: {
          definitionStatus: status,
        },
        updateMask: "definition_status",
      };
      return apiClient.workflowService.UpdateWorkflowDefinition(req);
    },
    ...options,
  });
}

// ==============================
// 枚举工具：definition_status
// ==============================
import { computed } from "vue";
import { $t } from "@/core/i18n";

export const definitionStatusList = computed(() => [
  { value: 'DRAFT' as const, label: $t("enum.definitionStatus.DRAFT") },
  { value: 'ENABLED' as const, label: $t("enum.definitionStatus.ENABLED") },
  { value: 'DISABLED' as const, label: $t("enum.definitionStatus.DISABLED") },
]);

export function definitionStatusLabel(
  s: oaservicev1_WorkflowDefinition_DefinitionStatus | undefined
): string {
  switch (s) {
    case 'DRAFT':
      return $t("enum.definitionStatus.DRAFT");
    case 'ENABLED':
      return $t("enum.definitionStatus.ENABLED");
    case 'DISABLED':
      return $t("enum.definitionStatus.DISABLED");
    default:
      return $t("common.unknown");
  }
}

export function definitionStatusColor(
  s: oaservicev1_WorkflowDefinition_DefinitionStatus | undefined
): string {
  switch (s) {
    case 'ENABLED':
      return "#67c23a"; // 绿
    case 'DISABLED':
      return "#909399"; // 灰
    default:
      return "#e6a23c"; // 黄（DRAFT）
  }
}

// ==============================
// 请假类型 / 额度 / 申请单
// ==============================

export function useListLeaveTypes(
  query: PaginationQuery,
  options?: Omit<UseQueryOptions<oaservicev1_ListLeaveTypesResponse, Error>, "queryKey">
) {
  return useQuery({
    queryKey: ["listLeaveTypes", query],
    queryFn: () =>
      apiClient.leaveService.ListLeaveTypes(query.toRawParams()),
    ...options,
  });
}

export async function fetchListLeaveTypes(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listLeaveTypes", params],
    queryFn: () =>
      apiClient.leaveService.ListLeaveTypes(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useCreateLeaveType(
  options?: UseMutationOptions<
    oaservicev1_LeaveType,
    Error,
    oaservicev1_CreateLeaveTypeRequest
  >
) {
  return useMutation({
    mutationFn: (req) => apiClient.leaveService.CreateLeaveType(req),
    ...options,
  });
}

export function useGrantLeaveBalance(
  options?: UseMutationOptions<
    Record<never, never>,
    Error,
    oaservicev1_GrantLeaveBalanceRequest
  >
) {
  return useMutation({
    mutationFn: (req) => apiClient.leaveService.GrantLeaveBalance(req),
    ...options,
  });
}

export function useListLeaveBalances(
  req: oaservicev1_ListLeaveBalancesRequest,
  options?: Omit<UseQueryOptions<oaservicev1_ListLeaveBalancesResponse, Error>, "queryKey">
) {
  return useQuery({
    queryKey: ["listLeaveBalances", req],
    queryFn: () => apiClient.leaveService.ListLeaveBalances(req),
    ...options,
  });
}

export function useListLeaveApplications(
  req: oaservicev1_ListLeaveApplicationsRequest,
  options?: Omit<UseQueryOptions<oaservicev1_ListLeaveApplicationsResponse, Error>, "queryKey">
) {
  return useQuery({
    queryKey: ["listLeaveApplications", req],
    queryFn: () => apiClient.leaveService.ListLeaveApplications(req),
    ...options,
  });
}

// ==============================
// 报销申请单
// ==============================

export function useListExpenseApplications(
  req: oaservicev1_ListExpenseApplicationsRequest,
  options?: Omit<UseQueryOptions<oaservicev1_ListExpenseApplicationsResponse, Error>, "queryKey">
) {
  return useQuery({
    queryKey: ["listExpenseApplications", req],
    queryFn: () => apiClient.expenseService.ListExpenseApplications(req),
    ...options,
  });
}

// ==============================
// 考勤记录 / 设置 / 结算
// ==============================

export function useListAttendanceRecords(
  req: oaservicev1_ListAttendanceRecordsRequest,
  options?: Omit<UseQueryOptions<oaservicev1_ListAttendanceRecordsResponse, Error>, "queryKey">
) {
  return useQuery({
    queryKey: ["listAttendanceRecords", req],
    queryFn: () => apiClient.attendanceService.ListAttendanceRecords(req),
    ...options,
  });
}

export function useUpdateAttendanceSetting(
  options?: UseMutationOptions<
    Record<never, never>,
    Error,
    oaservicev1_AttendanceSetting
  >
) {
  return useMutation({
    mutationFn: (req) => apiClient.attendanceService.UpdateAttendanceSetting(req),
    ...options,
  });
}

export function useRunDailySettlement(
  options?: UseMutationOptions<
    oaservicev1_RunDailySettlementResponse,
    Error,
    oaservicev1_RunDailySettlementRequest
  >
) {
  return useMutation({
    mutationFn: (req) => apiClient.attendanceService.RunDailySettlement(req),
    ...options,
  });
}

// ==============================
// 审批中心：我的任务 / 详情 / 审批动作
// ==============================

import type {
  oaservicev1_GetMyTasksResponse,
  oaservicev1_GetTaskResponse,
  oaservicev1_AuditTaskRequest,
  oaservicev1_ListType,
} from "@/api/generated/admin/service/v1";

export async function fetchMyTasks(listType: oaservicev1_ListType) {
  return queryClient.fetchQuery({
    queryKey: ["myTasks", listType],
    queryFn: () => apiClient.workflowService.GetMyTasks({ listType }),
    staleTime: 0,
    retry: 0,
  }) as Promise<oaservicev1_GetMyTasksResponse>;
}

export async function fetchTaskDetail(taskId: number) {
  return queryClient.fetchQuery({
    queryKey: ["taskDetail", taskId],
    queryFn: () => apiClient.workflowService.GetTask({ id: taskId }),
    staleTime: 0,
    retry: 0,
  }) as Promise<oaservicev1_GetTaskResponse>;
}

export function useAuditTask(
  options?: UseMutationOptions<
    Record<never, never>,
    Error,
    oaservicev1_AuditTaskRequest
  >
) {
  return useMutation({
    mutationFn: (req) => apiClient.workflowService.AuditTask(req),
    ...options,
  });
}

/** 审批历史动作 → 展示文案。 */
export function auditActionLabel(action?: string): string {
  switch (action) {
    case "SUBMIT": return "已提交";
    case "APPROVE": return "已审批通过";
    case "REJECT": return "已审批驳回";
    case "FORWARD": return "已转办";
    case "WITHDRAW": return "已撤回";
    default: return "-";
  }
}
