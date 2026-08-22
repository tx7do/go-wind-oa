<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig">
      <template #leaveStatus="scope: any">
        <ElTag :type="statusTag(scope.row.leaveStatus)">{{ statusLabel(scope.row.leaveStatus) }}</ElTag>
      </template>
    </ProPage>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import { fetchListLeaveApplications } from "@/api/composables";
import type { oaservicev1_ListLeaveApplicationsRequest } from "@/api/generated/admin/service/v1";

const pageRef = ref();

function fmtDate(v?: string) {
  return v ? String(v).slice(0, 10) : "-";
}

function statusLabel(s?: string): string {
  switch (s) {
    case "APPROVED": return "已通过";
    case "REJECTED": return "已驳回";
    case "WITHDRAWN": return "已撤回";
    default: return "审批中";
  }
}

function statusTag(s?: string): "success" | "danger" | "info" | "warning" {
  switch (s) {
    case "APPROVED": return "success";
    case "REJECTED": return "danger";
    case "WITHDRAWN": return "info";
    default: return "warning";
  }
}

const pageConfig = computed<ProPageConfig>(() => ({
  table: {
    listAction: async (query: any) => {
      const req: oaservicev1_ListLeaveApplicationsRequest = {
        userId: 0,
        status: undefined,
        page: query.page,
        pageSize: query.pageSize,
      };
      const result = await fetchListLeaveApplications(req);
      return { items: (result as any)?.items ?? [], total: (result as any)?.total ?? 0 };
    },
    toolbar: [],
    toolbarRight: [],
    defaultToolbar: ["refresh", "filter"],
    pagination: true,
    tableAttrs: { border: true, stripe: true },
    columns: [
      { prop: "id", label: "ID", width: 80 },
      { prop: "createdBy", label: "申请人ID", width: 100 },
      { prop: "leaveTypeName", label: "类型", width: 120 },
      {
        label: "起止",
        minWidth: 200,
        formatter: (row: any) => `${fmtDate(row.startDate)} ~ ${fmtDate(row.endDate)}`,
      },
      { prop: "days", label: "天数", width: 80 },
      { prop: "reason", label: "事由", minWidth: 160, showOverflowTooltip: true },
      { prop: "leaveStatus", label: "状态", width: 100, slotName: "leaveStatus" },
      { prop: "instanceId", label: "流程实例", width: 100 },
    ],
  },
}));
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
