<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import { fetchDirectoryUsers } from "@/api/composables";

const pageRef = ref();

function joinArr(arr: any): string {
  if (!arr || !Array.isArray(arr)) return "";
  return (arr as string[]).join("、");
}

const pageConfig = computed<ProPageConfig>(() => ({
  table: {
    listAction: async () => {
      const resp = await fetchDirectoryUsers();
      return (resp as any)?.items ?? [];
    },
    toolbar: [],
    toolbarRight: [],
    defaultToolbar: ["refresh", "filter"],
    pagination: false,
    tableAttrs: { border: true, stripe: true },
    columns: [
      { prop: "id", label: "ID", width: 70 },
      { prop: "nickname", label: "昵称", width: 120 },
      { prop: "realname", label: "姓名", width: 120 },
      {
        label: "部门",
        minWidth: 180,
        formatter: (row: any) => joinArr(row.orgUnitNames),
      },
      {
        label: "职位",
        minWidth: 180,
        formatter: (row: any) => joinArr(row.positionNames),
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
