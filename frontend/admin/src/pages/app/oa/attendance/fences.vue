<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage
      ref="pageRef"
      :config="pageConfig"
      @add="handleAdd"
    >
      <template #operation="scope: any">
        <ElButton size="small" type="danger" link @click="handleDelete(scope.row)">删除</ElButton>
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
import {
  fetchListGeofences,
  useDeleteGeofence,
} from "@/api/composables";

const pageRef = ref();
const drawerRef = ref();

const deleteMutation = useDeleteGeofence({
  onSuccess: () => {
    ElMessage.success("已删除");
    pageRef.value?.refresh();
  },
  onError: (err: Error) => ElMessage.error(err.message || "删除失败"),
});

function handleDelete(row: any) {
  ElMessageBox.confirm("删除后，该围栏不再参与打卡判定。确认删除？", "删除围栏", { type: "warning" })
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
      { prop: "name", label: "名称", minWidth: 160 },
      { prop: "latitude", label: "纬度", width: 140 },
      { prop: "longitude", label: "经度", width: 140 },
      { prop: "radiusMeters", label: "半径(米)", width: 120 },
      { prop: "createdAt", label: "创建时间", width: 180 },
      { prop: "operation", label: "操作", width: 100, slotName: "operation" },
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
