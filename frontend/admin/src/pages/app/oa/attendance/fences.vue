<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd">
      <template #operation="scope: any">
        <ElButton size="small" type="danger" link @click="handleDelete(scope.row)">
          {{ $t("common.button.delete") }}
        </ElButton>
      </template>
    </ProPage>

    <FenceDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElButton, ElMessageBox, ElMessage } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import FenceDrawer from "./fence-drawer.vue";
import { fetchListGeofences, useDeleteGeofence } from "@/api/composables";
import { $t } from "@/core/i18n";

const pageRef = ref();
const drawerRef = ref();

const deleteMutation = useDeleteGeofence({
  onSuccess: () => {
    ElMessage.success($t("common.notification.deleteSuccess"));
    pageRef.value?.refresh();
  },
  onError: (err: Error) => ElMessage.error(err.message || $t("common.notification.deleteFailed")),
});

function handleDelete(row: any) {
  ElMessageBox.confirm(
    $t("pages.oa.attendance.deleteFenceConfirmContent"),
    $t("pages.oa.attendance.deleteFenceConfirmTitle"),
    { type: "warning" }
  )
    .then(() => deleteMutation.mutate({ id: row.id as number }))
    .catch(() => {});
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
      const result = await fetchListGeofences();
      return (result as any)?.items ?? [];
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    pagination: false,
    tableAttrs: { border: true, stripe: true },
    columns: [
      { prop: "name", label: $t("pages.oa.attendance.fenceColName"), minWidth: 160 },
      { prop: "latitude", label: $t("pages.oa.attendance.fenceColLatitude"), width: 140 },
      { prop: "longitude", label: $t("pages.oa.attendance.fenceColLongitude"), width: 140 },
      { prop: "radiusMeters", label: $t("pages.oa.attendance.fenceColRadius"), width: 120 },
      { prop: "createdAt", label: $t("pages.oa.attendance.fenceColCreatedAt"), width: 180 },
      {
        prop: "operation",
        label: $t("pages.oa.attendance.colOperation"),
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
