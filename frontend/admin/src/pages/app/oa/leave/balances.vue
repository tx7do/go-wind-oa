<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" />

    <LeaveBalanceDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import LeaveBalanceDrawer from "./leave-balance-drawer.vue";
import { fetchListLeaveBalances } from "@/api/composables";
import { $t } from "@/core/i18n";

const pageRef = ref();
const drawerRef = ref();

function handleAdd() {
  drawerRef.value?.open();
}

function handleSuccess() {
  pageRef.value?.refresh();
}

const pageConfig = computed<ProPageConfig>(() => ({
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.oa.leave.balances.searchYear"),
        field: "year",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
    ],
  },
  table: {
    listAction: async (query: any) => {
      // 后端 proto 仅支持 userId/year 过滤（无分页参数），year=0 表示当年。
      const result = await fetchListLeaveBalances({
        userId: 0,
        year: Number(query.year) || 0,
      });
      return { items: (result as any)?.items ?? [], total: (result as any)?.total ?? 0 };
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    pagination: false,
    tableAttrs: { border: true, stripe: true },
    columns: [
      { prop: "userId", label: $t("pages.oa.leave.balances.colUserId"), width: 100 },
      { prop: "leaveTypeId", label: $t("pages.oa.leave.balances.colLeaveTypeId"), width: 100 },
      { prop: "year", label: $t("pages.oa.leave.balances.colYear"), width: 100 },
      { prop: "totalDays", label: $t("pages.oa.leave.balances.colTotalDays"), width: 120 },
      { prop: "usedDays", label: $t("pages.oa.leave.balances.colUsedDays"), width: 120 },
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
