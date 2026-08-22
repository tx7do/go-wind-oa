<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig">
      <template #sealStatus="scope: any">
        <ElTag :type="statusTag(scope.row.sealStatus)">{{ statusLabel(scope.row.sealStatus) }}</ElTag>
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
  oaservicev1_ListSealApplicationsRequest,
} from "@/api/generated/admin/service/v1";
import { fetchListSealApplications } from "@/api/composables";

const pageRef = ref();

function sealTypeLabel(s?: string): string {
  switch (s) {
    case "OFFICIAL_SEAL": return "公章";
    case "CONTRACT_SEAL": return "合同章";
    case "FINANCE_SEAL": return "财务章";
    case "LEGAL_SEAL": return "法人章";
    default: return "-";
  }
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
      const req: oaservicev1_ListSealApplicationsRequest = {
        userId: query.userId ?? 0,
        status: query.status,
        page: query.page,
        pageSize: query.pageSize,
      };
      const result = await fetchListSealApplications(req);
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
      { prop: "purpose", label: "用印事由", minWidth: 160, showOverflowTooltip: true },
      {
        prop: "sealType",
        label: "印章类型",
        width: 120,
        formatter: (row: any) => sealTypeLabel(row.sealType),
      },
      { prop: "fileCount", label: "文件份数", width: 100 },
      { prop: "recipient", label: "收件方", width: 160, showOverflowTooltip: true },
      { prop: "sealStatus", label: "状态", width: 100, slotName: "sealStatus" },
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
