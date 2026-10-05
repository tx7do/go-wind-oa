<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig">
      <template #expenseStatus="scope: any">
        <ElTag :type="workflowInstanceStatusToTag(scope.row.expenseStatus)">
          {{ workflowInstanceStatusToName(scope.row.expenseStatus) }}
        </ElTag>
      </template>
      <template #operation="scope: any">
        <ElButton size="small" type="primary" link @click="openDetail(scope.row.items)">
          {{ $t("common.button.view") }}
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
  workflowInstanceStatusToName,
  workflowInstanceStatusToTag,
} from "@/api/composables";
import type { oaservicev1_ListExpenseApplicationsRequest } from "@/api/generated/admin/service/v1";
import { formatDateTime } from "@/utils/date";
import { $t } from "@/core/i18n";

const pageRef = ref();
const drawerRef = ref();

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
      { prop: "createdBy", label: $t("pages.oa.expense.colCreatedBy"), width: 100 },
      {
        prop: "title",
        label: $t("pages.oa.expense.colTitle"),
        minWidth: 200,
        showOverflowTooltip: true,
      },
      { prop: "totalAmount", label: $t("pages.oa.expense.colTotalAmount"), width: 120 },
      {
        prop: "expenseStatus",
        label: $t("pages.oa.expense.colStatus"),
        width: 100,
        slotName: "expenseStatus",
      },
      { prop: "instanceId", label: $t("pages.oa.expense.colInstanceId"), width: 100 },
      {
        prop: "createdAt",
        label: $t("pages.oa.expense.colCreatedAt"),
        width: 170,
        formatter: (row: any) => formatDateTime(row.createdAt || row.created_at),
      },
      {
        prop: "operation",
        label: $t("pages.oa.expense.colOperation"),
        width: 100,
        slotName: "operation",
      },
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
