<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig">
      <template #expenseStatus="scope: any">
        <ElTag :type="statusTag(scope.row.expenseStatus)">{{ statusLabel(scope.row.expenseStatus) }}</ElTag>
      </template>
      <template #operation="scope: any">
        <ElButton size="small" type="primary" link @click="openDetail(scope.row.items)">
          查看
        </ElButton>
      </template>
    </ProPage>

    <ExpenseDetailDrawer ref="drawerRef" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElButton, ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import ExpenseDetailDrawer from "./expense-detail-drawer.vue";
import {
  fetchListExpenseApplications,
} from "@/api/composables";
import type {
  oaservicev1_ListExpenseApplicationsRequest,
} from "@/api/generated/admin/service/v1";

const pageRef = ref();
const drawerRef = ref();

function fmtTime(v?: string) {
  return v ? String(v).replace("T", " ").slice(0, 19) : "-";
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

function openDetail(items: any[]) {
  drawerRef.value?.open(items);
}

const pageConfig = computed<ProPageConfig>(() => ({
  table: {
    listAction: async (query: any) => {
      const req: oaservicev1_ListExpenseApplicationsRequest = {
        userId: query.userId ?? 0,
        status: query.status,
        page: query.page,
        pageSize: query.pageSize,
      };
      const result = await fetchListExpenseApplications(req);
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
      { prop: "title", label: "事由", minWidth: 200, showOverflowTooltip: true },
      { prop: "totalAmount", label: "总额", width: 120 },
      { prop: "expenseStatus", label: "状态", width: 100, slotName: "expenseStatus" },
      { prop: "instanceId", label: "流程实例", width: 100 },
      {
        prop: "createdAt",
        label: "提交时间",
        width: 170,
        formatter: (row: any) => fmtTime(row.createdAt || row.created_at),
      },
      { prop: "operation", label: "操作", width: 100, slotName: "operation" },
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
