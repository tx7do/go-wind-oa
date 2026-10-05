<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <div class="status-bar">
      <ElRadioGroup v-model="currentListType" @change="onListTypeChange">
        <ElRadioButton value="PENDING">{{ $t("pages.oa.approval.tabPending") }}</ElRadioButton>
        <ElRadioButton value="DONE">{{ $t("pages.oa.approval.tabDone") }}</ElRadioButton>
      </ElRadioGroup>
    </div>
    <ProPage ref="pageRef" :config="pageConfig">
      <template #status="scope: any">
        <ElTag v-if="currentListType === 'PENDING'" type="warning" size="small">
          {{ $t("pages.oa.approval.pendingTag") }}
        </ElTag>
        <template v-else-if="scope.row.auditAction">
          {{ auditActionLabel(scope.row.auditAction) }}
        </template>
        <template v-else>{{ scope.row.statusLabel }}</template>
      </template>
      <template #operation="scope: any">
        <ElButton
          v-if="currentListType === 'PENDING' && scope.row.taskId"
          size="small"
          type="primary"
          link
          @click="openDetail(scope.row.taskId)"
        >
          {{ $t("pages.oa.approval.approve") }}
        </ElButton>
      </template>
    </ProPage>

    <ApprovalDetailDrawer ref="drawerRef" @success="onDrawerSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElButton, ElRadioButton, ElRadioGroup, ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import ApprovalDetailDrawer from "./detail-drawer.vue";
import { auditActionLabel, fetchMyTasks } from "@/api/composables";
import type { oaservicev1_ListType } from "@/api/generated/admin/service/v1";
import { formatDateTime } from "@/utils/date";
import { $t } from "@/core/i18n";

const pageRef = ref();
const drawerRef = ref();
const currentListType = ref<oaservicev1_ListType>("PENDING");

const pageConfig = computed<ProPageConfig>(() => ({
  table: {
    listAction: async (query: any) => {
      const result = await fetchMyTasks(currentListType.value, query.page, query.pageSize);
      return { items: (result as any)?.items ?? [], total: (result as any)?.total ?? 0 };
    },
    toolbar: [],
    toolbarRight: [],
    defaultToolbar: ["refresh", "filter"],
    pagination: true,
    tableAttrs: { border: true, stripe: true },
    columns: [
      { prop: "instanceId", label: $t("pages.oa.approval.colInstanceId"), width: 100 },
      {
        prop: "statusLabel",
        label: $t("pages.oa.approval.colStatus"),
        width: 140,
        slotName: "status",
      },
      {
        prop: "createdAt",
        label: $t("pages.oa.approval.colTime"),
        width: 200,
        formatter: (row: any) => formatDateTime(row.createdAt),
      },
      {
        prop: "operation",
        label: $t("pages.oa.approval.colOperation"),
        width: 140,
        slotName: "operation",
      },
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
