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

const pageRef = ref();
const drawerRef = ref();

function handleAdd() {
  drawerRef.value?.open();
}

function handleSuccess() {
  pageRef.value?.refresh();
}

const pageConfig = computed<ProPageConfig>(() => ({
  table: {
    listAction: async () => {
      const result = await fetchListLeaveBalances({ userId: 0, year: 0 });
      return { items: (result as any)?.items ?? [], total: 0 };
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    pagination: false,
    tableAttrs: { border: true, stripe: true },
    columns: [
      { prop: "userId", label: "用户ID", width: 100 },
      { prop: "leaveTypeId", label: "类型ID", width: 100 },
      { prop: "year", label: "年度", width: 100 },
      { prop: "totalDays", label: "总额度(天)", width: 120 },
      { prop: "usedDays", label: "已用(天)", width: 120 },
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
