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
import { formatDateTime } from "@/utils/date";
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
  table: {
    listAction: async (query: any) => {
      const result = await fetchListLeaveTypes(
        new PaginationQuery({
          paging: { page: query.page || 1, pageSize: query.pageSize || 10 },
        })
      );
      return { items: (result as any)?.items ?? [], total: (result as any)?.total ?? 0 };
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    pagination: true,
    tableAttrs: { border: true, stripe: true },
    columns: [
      { prop: "id", label: "ID", width: 80 },
      { prop: "code", label: $t("pages.oa.leave.types.colCode"), width: 160 },
      { prop: "name", label: $t("pages.oa.leave.types.colName"), width: 200 },
      { prop: "remark", label: $t("pages.oa.leave.types.colRemark"), minWidth: 200 },
      {
        prop: "createdAt",
        label: $t("pages.oa.leave.types.colCreatedAt"),
        width: 170,
        formatter: (row: any) => formatDateTime(row.createdAt || row.created_at),
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
