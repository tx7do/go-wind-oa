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
        updateMask: "definitionStatus",
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
// 出差申请单
// ==============================

import type {
  oaservicev1_ListBusinessTripApplicationsRequest,
  oaservicev1_ListBusinessTripApplicationsResponse,
} from "@/api/generated/admin/service/v1";

export function useListBusinessTripApplications(
  req: oaservicev1_ListBusinessTripApplicationsRequest,
  options?: Omit<UseQueryOptions<oaservicev1_ListBusinessTripApplicationsResponse, Error>, "queryKey">
) {
  return useQuery({
    queryKey: ["listBusinessTripApplications", req],
    queryFn: () => apiClient.businessTripService.ListBusinessTripApplications(req),
    ...options,
  });
}

// ==============================
// 加班 / 用印 / 外出申请单
// ==============================

import type {
  oaservicev1_ListOvertimeApplicationsRequest,
  oaservicev1_ListOvertimeApplicationsResponse,
  oaservicev1_ListSealApplicationsRequest,
  oaservicev1_ListSealApplicationsResponse,
  oaservicev1_ListOutingApplicationsRequest,
  oaservicev1_ListOutingApplicationsResponse,
} from "@/api/generated/admin/service/v1";

export function useListOvertimeApplications(
  req: oaservicev1_ListOvertimeApplicationsRequest,
  options?: Omit<UseQueryOptions<oaservicev1_ListOvertimeApplicationsResponse, Error>, "queryKey">
) {
  return useQuery({
    queryKey: ["listOvertimeApplications", req],
    queryFn: () => apiClient.overtimeService.ListOvertimeApplications(req),
    ...options,
  });
}

export function useListSealApplications(
  req: oaservicev1_ListSealApplicationsRequest,
  options?: Omit<UseQueryOptions<oaservicev1_ListSealApplicationsResponse, Error>, "queryKey">
) {
  return useQuery({
    queryKey: ["listSealApplications", req],
    queryFn: () => apiClient.sealApplicationService.ListSealApplications(req),
    ...options,
  });
}

export function useListOutingApplications(
  req: oaservicev1_ListOutingApplicationsRequest,
  options?: Omit<UseQueryOptions<oaservicev1_ListOutingApplicationsResponse, Error>, "queryKey">
) {
  return useQuery({
    queryKey: ["listOutingApplications", req],
    queryFn: () => apiClient.outingService.ListOutingApplications(req),
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

export function useGetAttendanceSetting(
  options?: Omit<UseQueryOptions<oaservicev1_AttendanceSetting, Error>, "queryKey">
) {
  return useQuery({
    queryKey: ["getAttendanceSetting"],
    queryFn: () => apiClient.attendanceService.GetAttendanceSetting({}),
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

export async function fetchMyTasks(
  listType: oaservicev1_ListType,
  page?: number,
  pageSize?: number
) {
  return queryClient.fetchQuery({
    queryKey: ["myTasks", listType, page, pageSize],
    queryFn: () =>
      apiClient.workflowService.GetMyTasks({
        listType,
        page: page ?? undefined,
        pageSize: pageSize ?? undefined,
      }),
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

// ==============================
// 节假日/调休日设置
// ==============================

import type {
  oaservicev1_Holiday,
  oaservicev1_ListHolidaysResponse,
} from "@/api/generated/admin/service/v1";

export function useListHolidays(
  req: { year: number },
  options?: Omit<UseQueryOptions<oaservicev1_ListHolidaysResponse, Error>, "queryKey">
) {
  return useQuery({
    queryKey: ["listHolidays", req.year],
    queryFn: () => apiClient.attendanceService.ListHolidays(req),
    ...options,
  });
}

export function useUpsertHoliday(
  options?: UseMutationOptions<Record<never, never>, Error, oaservicev1_Holiday>
) {
  return useMutation({
    mutationFn: (req) => apiClient.attendanceService.UpsertHoliday(req),
    ...options,
  });
}

export function useDeleteHoliday(
  options?: UseMutationOptions<Record<never, never>, Error, { id: number }>
) {
  return useMutation({
    mutationFn: (req) => apiClient.attendanceService.DeleteHoliday(req),
    ...options,
  });
}

/** 节假日类型 → 展示文案/标签色。 */
export function holidayTypeLabel(t?: string): string {
  return t === "WORKDAY" ? "调休上班" : "法定假日";
}

export function holidayTypeTag(t?: string): "danger" | "success" {
  return t === "WORKDAY" ? "danger" : "success";
}

// ==============================
// 流程编辑器数据源：用户 / 职位列表
// ==============================

import type {
  identityservicev1_ListUserResponse,
  identityservicev1_User,
  identityservicev1_ListPositionResponse,
  identityservicev1_Position,
} from "@/api/generated/admin/service/v1";

/** 用户列表（审批人选择器）。取前 200 条，展示 姓名(#ID)。 */
export async function fetchUsers() {
  return queryClient.fetchQuery({
    queryKey: ["workflowEditorUsers"],
    queryFn: () =>
      apiClient.userService.List({
        page: 1,
        pageSize: 200,
        noPaging: false,
        sorting: undefined,
      }),
    staleTime: 0,
    retry: 0,
  }) as Promise<identityservicev1_ListUserResponse>;
}

export function userDisplayName(u?: identityservicev1_User): string {
  if (!u) return "";
  return u.realname || u.nickname || u.username || `#${u.id}`;
}

/** 职位列表（职位审批人选择器）。取前 200 条。 */
export async function fetchPositions() {
  return queryClient.fetchQuery({
    queryKey: ["workflowEditorPositions"],
    queryFn: () =>
      apiClient.positionService.List({
        page: 1,
        pageSize: 200,
        noPaging: false,
        sorting: undefined,
      }),
    staleTime: 0,
    retry: 0,
  }) as Promise<identityservicev1_ListPositionResponse>;
}

export function positionDisplayName(p?: identityservicev1_Position): string {
  if (!p) return "";
  return p.name || `#${p.id}`;
}
