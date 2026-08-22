<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <div class="status-bar">
      <ElRadioGroup v-model="currentListType" @change="onListTypeChange">
        <ElRadioButton value="PENDING">待我审批</ElRadioButton>
        <ElRadioButton value="DONE">已办</ElRadioButton>
      </ElRadioGroup>
    </div>
    <ProPage ref="pageRef" :config="pageConfig">
      <template #operation="scope: any">
        <ElButton
          v-if="currentListType === 'PENDING' && scope.row.taskId"
          size="small"
          type="primary"
          link
          @click="openDetail(scope.row.taskId)"
        >
          审批
        </ElButton>
      </template>
    </ProPage>

    <ApprovalDetailDrawer ref="drawerRef" @success="onDrawerSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElButton, ElRadioButton, ElRadioGroup } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import ApprovalDetailDrawer from "./detail-drawer.vue";
import { fetchMyTasks } from "@/api/composables";
import type { oaservicev1_ListType } from "@/api/generated/admin/service/v1";

const pageRef = ref();
const drawerRef = ref();
const currentListType = ref<oaservicev1_ListType>("PENDING");

function fmtTime(v?: string) {
  return v ? String(v).replace("T", " ").slice(0, 19) : "-";
}

const pageConfig = computed<ProPageConfig>(() => ({
  table: {
    listAction: async (query: any) => {
      const result = await fetchMyTasks(
        currentListType.value,
        query.page,
        query.pageSize
      );
      return { items: (result as any)?.items ?? [], total: (result as any)?.total ?? 0 };
    },
    toolbar: [],
    toolbarRight: [],
    defaultToolbar: ["refresh", "filter"],
    pagination: true,
    tableAttrs: { border: true, stripe: true },
    columns: [
      { prop: "instanceId", label: "实例ID", width: 100 },
      { prop: "statusLabel", label: "状态", width: 140 },
      {
        prop: "createdAt",
        label: "时间",
        width: 200,
        formatter: (row: any) => fmtTime(row.createdAt),
      },
      { prop: "operation", label: "操作", width: 140, slotName: "operation" },
    ],
  },
}));

function onListTypeChange() {
  pageRef.value?.refresh();
}

function openDetail(taskId: number) {
  drawerRef.value?.open(taskId);
}

function onDrawerSuccess() {
  pageRef.value?.refresh();
}
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
.status-bar {
  padding: 12px 16px;
}
</style>
