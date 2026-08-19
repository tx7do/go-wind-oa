<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd">
      <template #operation="scope: any">
        <ElButton
          size="small"
          type="danger"
          link
          @click="handleDelete(scope.row)"
        >
          {{ $t("common.delete") }}
        </ElButton>
      </template>
    </ProPage>

    <WifiDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElButton, ElMessage, ElMessageBox } from "element-plus";
import type { oaservicev1_AttendanceWifi } from "@/api/generated/admin/service/v1";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";

import {
  fetchListAttendanceWifis,
  useDeleteAttendanceWifi,
} from "@/api/composables";
import { queryClient } from "@/plugins/vue-query";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";

import WifiDrawer from "./wifi-drawer.vue";

const pageRef = ref();
const drawerRef = ref();

const deleteMutation = useDeleteAttendanceWifi({
  onSuccess: () => {
    ElMessage.success($t("common.notification.deleteSuccess"));
    queryClient.invalidateQueries({ queryKey: ["listAttendanceWifis"] });
    pageRef.value?.refresh();
  },
  onError: (err: Error) => {
    ElMessage.error(err.message || $t("common.error"));
  },
});

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  table: {
    listAction: async (query: any) => {
      const { page, pageSize } = query;
      const result = await fetchListAttendanceWifis(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: {},
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    pagination: false,
    tableAttrs: { border: true, stripe: false },
    columns: [
      {
        prop: "ssid",
        label: $t("pages.oa.attendance.wifiColSsid"),
        minWidth: 200,
      },
      {
        prop: "bssid",
        label: $t("pages.oa.attendance.wifiColBssid"),
        width: 220,
      },
      {
        prop: "created_at",
        label: $t("pages.oa.attendance.wifiColCreatedAt"),
        width: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "operation",
        label: $t("pages.oa.attendance.colOperation"),
        width: 100,
        slotName: "operation",
      },
    ],
  },
}));

function handleAdd() {
  drawerRef.value?.open();
}

function handleDelete(row: oaservicev1_AttendanceWifi) {
  ElMessageBox.confirm(
    $t("pages.oa.attendance.deleteWifiConfirmContent"),
    $t("pages.oa.attendance.deleteWifiConfirmTitle"),
    {
      confirmButtonText: $t("common.confirm"),
      cancelButtonText: $t("common.cancel"),
      type: "warning",
    }
  )
    .then(() => {
      deleteMutation.mutate({ id: row.id as number });
    })
    .catch(() => {});
}

function handleSuccess() {
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
</style>
