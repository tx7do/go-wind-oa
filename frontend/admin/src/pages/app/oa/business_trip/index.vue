<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig">
      <template #tripStatus="scope: any">
        <ElTag :type="statusTag(scope.row.tripStatus)">{{ statusLabel(scope.row.tripStatus) }}</ElTag>
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
  oaservicev1_ListBusinessTripApplicationsRequest,
} from "@/api/generated/admin/service/v1";
import { fetchListBusinessTripApplications } from "@/api/composables";

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
      const req: oaservicev1_ListBusinessTripApplicationsRequest = {
        userId: query.userId ?? 0,
        status: query.status,
        page: query.page,
        pageSize: query.pageSize,
      };
      const result = await fetchListBusinessTripApplications(req);
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
      { prop: "title", label: "标题", minWidth: 160, showOverflowTooltip: true },
      { prop: "destination", label: "目的地", width: 160, showOverflowTooltip: true },
      {
        label: "起止",
        minWidth: 200,
        formatter: (row: any) => `${fmtDate(row.startDate)} ~ ${fmtDate(row.endDate)}`,
      },
      { prop: "tripStatus", label: "状态", width: 100, slotName: "tripStatus" },
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
