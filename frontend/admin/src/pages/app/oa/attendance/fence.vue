<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd">
      <template #operation="scope: any">
        <ElButton
          size="small"
          type="primary"
          link
          @click="handleEdit(scope.row)"
        >
          {{ $t("common.edit") }}
        </ElButton>
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

    <FenceDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElButton, ElMessage, ElMessageBox } from "element-plus";
import type { oaservicev1_AttendanceFence } from "@/api/generated/admin/service/v1";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";

import {
  fetchListAttendanceFences,
  useDeleteAttendanceFence,
} from "@/api/composables";
import { queryClient } from "@/plugins/vue-query";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";

import FenceDrawer from "./fence-drawer.vue";

const pageRef = ref();
const drawerRef = ref();

const deleteMutation = useDeleteAttendanceFence({
  onSuccess: () => {
    ElMessage.success($t("common.notification.deleteSuccess"));
    queryClient.invalidateQueries({ queryKey: ["listAttendanceFences"] });
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
      const result = await fetchListAttendanceFences(
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
        prop: "name",
        label: $t("pages.oa.attendance.fenceColName"),
        minWidth: 200,
      },
      {
        prop: "longitude",
        label: $t("pages.oa.attendance.fenceColLongitude"),
        width: 140,
      },
      {
        prop: "latitude",
        label: $t("pages.oa.attendance.fenceColLatitude"),
        width: 140,
      },
      {
        prop: "radius",
        label: $t("pages.oa.attendance.fenceColRadius"),
        width: 120,
      },
      {
        prop: "created_at",
        label: $t("pages.oa.attendance.fenceColCreatedAt"),
        width: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "operation",
        label: $t("pages.oa.attendance.colOperation"),
        width: 140,
        slotName: "operation",
      },
    ],
  },
}));

function handleAdd() {
  drawerRef.value?.open();
}

function handleEdit(row: oaservicev1_AttendanceFence) {
  drawerRef.value?.open(row);
}

function handleDelete(row: oaservicev1_AttendanceFence) {
  ElMessageBox.confirm(
    $t("pages.oa.attendance.deleteFenceConfirmContent"),
    $t("pages.oa.attendance.deleteFenceConfirmTitle"),
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
