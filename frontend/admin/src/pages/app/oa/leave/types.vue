<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" />

    <LeaveTypeDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import LeaveTypeDrawer from "./leave-type-drawer.vue";
import { fetchListLeaveTypes } from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";

const pageRef = ref();
const drawerRef = ref();

function fmtTime(v?: string) {
  return v ? String(v).replace("T", " ").slice(0, 19) : "-";
}

function handleAdd() {
  drawerRef.value?.open();
}

function handleSuccess() {
  pageRef.value?.refresh();
}

const pageConfig = computed<ProPageConfig>(() => ({
  table: {
    listAction: async () => {
      const result = await fetchListLeaveTypes(new PaginationQuery());
      return { items: (result as any)?.items ?? [], total: 0 };
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    pagination: false,
    tableAttrs: { border: true, stripe: true },
    columns: [
      { prop: "id", label: "ID", width: 80 },
      { prop: "code", label: "代码", width: 160 },
      { prop: "name", label: "名称", width: 200 },
      { prop: "remark", label: "备注", minWidth: 200 },
      {
        prop: "createdAt",
        label: "创建时间",
        width: 170,
        formatter: (row: any) => fmtTime(row.createdAt || row.created_at),
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
