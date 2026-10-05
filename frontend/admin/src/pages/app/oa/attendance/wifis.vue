<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd">
      <template #operation="scope: any">
        <ElButton size="small" type="danger" link @click="handleDelete(scope.row)">
          {{ $t("common.button.delete") }}
        </ElButton>
      </template>
    </ProPage>

    <WifiDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElButton, ElMessageBox, ElMessage } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import WifiDrawer from "./wifi-drawer.vue";
import { fetchListWifiFingerprints, useDeleteWifiFingerprint } from "@/api/composables";
import { $t } from "@/core/i18n";

const pageRef = ref();
const drawerRef = ref();

const deleteMutation = useDeleteWifiFingerprint({
  onSuccess: () => {
    ElMessage.success($t("common.notification.deleteSuccess"));
    pageRef.value?.refresh();
  },
  onError: (err: Error) => ElMessage.error(err.message || $t("common.notification.deleteFailed")),
});

function handleDelete(row: any) {
  ElMessageBox.confirm(
    $t("pages.oa.attendance.deleteWifiConfirmContent"),
    $t("pages.oa.attendance.deleteWifiConfirmTitle"),
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
      const result = await fetchListWifiFingerprints();
      return (result as any)?.items ?? [];
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    pagination: false,
    tableAttrs: { border: true, stripe: true },
    columns: [
      { prop: "ssid", label: "SSID", minWidth: 180 },
      { prop: "bssid", label: "BSSID", minWidth: 200 },
      { prop: "createdAt", label: $t("pages.oa.attendance.wifiColCreatedAt"), width: 180 },
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
