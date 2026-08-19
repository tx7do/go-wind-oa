/**
 * OA 工作流定义管理 composable。
 *
 * 对齐 cms internal-message.ts 模式：useQuery/fetchQuery 包列表查询，
 * useMutation 包创建。类型来自 @/api/generated/admin/service/v1（由
 * buf.vue-element.oa.typescript.gen.yaml 生成）。
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
  oaservicev1_AttendanceFence,
  oaservicev1_AttendanceWifi,
  oaservicev1_ListAttendanceFenceResponse,
  oaservicev1_ListAttendanceWifiResponse,
  oaservicev1_CreateAttendanceFenceRequest,
  oaservicev1_UpdateAttendanceFenceRequest,
  oaservicev1_DeleteAttendanceFenceRequest,
  oaservicev1_CreateAttendanceWifiRequest,
  oaservicev1_DeleteAttendanceWifiRequest,
} from "@/api/generated/admin/service/v1";
import { type PaginationQuery } from "@/core/transport/rest";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";

// ==============================
// 流程定义列表
// ==============================
export function useListWorkflowDefinitions(
  query: PaginationQuery,
  options?: UseQueryOptions<oaservicev1_ListWorkflowDefinitionResponse, Error>
) {
  return useQuery({
    queryKey: ["listWorkflowDefinitions", query],
    queryFn: () =>
      apiClient.workflowService.ListWorkflowDefinition({
        paging: query.toRawParams(),
      }),
    ...options,
  });
}

export async function fetchListWorkflowDefinitions(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listWorkflowDefinitions", params],
    queryFn: () =>
      apiClient.workflowService.ListWorkflowDefinition({
        paging: params.toRawParams(),
      }),
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
// 考勤围栏库
// ==============================

export function useListAttendanceFences(
  query: PaginationQuery,
  options?: UseQueryOptions<oaservicev1_ListAttendanceFenceResponse, Error>
) {
  return useQuery({
    queryKey: ["listAttendanceFences", query],
    queryFn: () =>
      apiClient.attendanceService.ListAttendanceFence({
        paging: query.toRawParams(),
      }),
    ...options,
  });
}

export async function fetchListAttendanceFences(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listAttendanceFences", params],
    queryFn: () =>
      apiClient.attendanceService.ListAttendanceFence({
        paging: params.toRawParams(),
      }),
    staleTime: 0,
    retry: 0,
  });
}

export function useCreateAttendanceFence(
  options?: UseMutationOptions<
    oaservicev1_AttendanceFence,
    Error,
    oaservicev1_CreateAttendanceFenceRequest
  >
) {
  return useMutation({
    mutationFn: (req) => apiClient.attendanceService.CreateAttendanceFence(req),
    ...options,
  });
}

export function useUpdateAttendanceFence(
  options?: UseMutationOptions<
    oaservicev1_AttendanceFence,
    Error,
    oaservicev1_UpdateAttendanceFenceRequest
  >
) {
  return useMutation({
    mutationFn: (req) => apiClient.attendanceService.UpdateAttendanceFence(req),
    ...options,
  });
}

export function useDeleteAttendanceFence(
  options?: UseMutationOptions<
    Record<never, never>,
    Error,
    oaservicev1_DeleteAttendanceFenceRequest
  >
) {
  return useMutation({
    mutationFn: (req) =>
      apiClient.attendanceService.DeleteAttendanceFence(req),
    ...options,
  });
}

// ==============================
// 考勤 Wi-Fi 指纹库
// ==============================

export function useListAttendanceWifis(
  query: PaginationQuery,
  options?: UseQueryOptions<oaservicev1_ListAttendanceWifiResponse, Error>
) {
  return useQuery({
    queryKey: ["listAttendanceWifis", query],
    queryFn: () =>
      apiClient.attendanceService.ListAttendanceWifi({
        paging: query.toRawParams(),
      }),
    ...options,
  });
}

export async function fetchListAttendanceWifis(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listAttendanceWifis", params],
    queryFn: () =>
      apiClient.attendanceService.ListAttendanceWifi({
        paging: params.toRawParams(),
      }),
    staleTime: 0,
    retry: 0,
  });
}

export function useCreateAttendanceWifi(
  options?: UseMutationOptions<
    oaservicev1_AttendanceWifi,
    Error,
    oaservicev1_CreateAttendanceWifiRequest
  >
) {
  return useMutation({
    mutationFn: (req) => apiClient.attendanceService.CreateAttendanceWifi(req),
    ...options,
  });
}

export function useDeleteAttendanceWifi(
  options?: UseMutationOptions<
    Record<never, never>,
    Error,
    oaservicev1_DeleteAttendanceWifiRequest
  >
) {
  return useMutation({
    mutationFn: (req) =>
      apiClient.attendanceService.DeleteAttendanceWifi(req),
    ...options,
  });
}

