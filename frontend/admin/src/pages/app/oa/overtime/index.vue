<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig">
      <template #overtimeStatus="scope: any">
        <ElTag :type="statusTag(scope.row.overtimeStatus)">{{ statusLabel(scope.row.overtimeStatus) }}</ElTag>
      </template>
    </ProPage>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import type {
  oaservicev1_ListOvertimeApplicationsRequest,
} from "@/api/generated/admin/service/v1";
import { fetchListOvertimeApplications } from "@/api/composables";

const pageRef = ref();

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

function fmtDate(ts?: { seconds?: number; nanos?: number } | string): string {
  if (!ts) return "";
  let d: Date | null = null;
  if (typeof ts === "object" && ts.seconds) {
    d = new Date(Number(ts.seconds) * 1000);
  } else if (typeof ts === "string") {
    d = new Date(ts);
  }
  if (!d || isNaN(d.getTime())) return "";
  return d.toISOString().slice(0, 10);
}

const pageConfig = computed<ProPageConfig>(() => ({
  table: {
    listAction: async (query: any) => {
      const req: oaservicev1_ListOvertimeApplicationsRequest = {
        userId: query.userId ?? 0,
        status: query.status,
        page: query.page,
        pageSize: query.pageSize,
      };
      const result = await fetchListOvertimeApplications(req);
      return { items: (result as any)?.items ?? [], total: (result as any)?.total ?? 0 };
    },
    toolbar: [],
    toolbarRight: [],
    defaultToolbar: ["refresh", "filter"],
    pagination: true,
    tableAttrs: { border: true, stripe: true },
    columns: [
      { prop: "id", label: "ID", width: 80 },
      { prop: "applicantName", label: "申请人", width: 120 },
      { prop: "reason", label: "事由", minWidth: 160, showOverflowTooltip: true },
      {
        label: "起止",
        minWidth: 200,
        formatter: (row: any) => `${fmtDate(row.startTime)} ~ ${fmtDate(row.endTime)}`,
      },
      { prop: "overtimeStatus", label: "状态", width: 100, slotName: "overtimeStatus" },
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
