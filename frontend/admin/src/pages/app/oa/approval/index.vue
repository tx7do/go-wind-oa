<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ElCard shadow="never" class="flex-1">
      <ElTabs v-model="tab" @tab-change="load">
        <ElTabPane label="待我审批" name="PENDING" />
        <ElTabPane label="已办" name="DONE" />
        <ElTabPane label="我发起的" name="SUBMITTED" />
      </ElTabs>
      <ElTable :data="items" border stripe v-loading="loading">
        <ElTableColumn prop="instanceId" label="实例ID" width="100" />
        <ElTableColumn prop="statusLabel" label="状态" width="140" />
        <ElTableColumn prop="createdAt" label="时间" width="200">
          <template #default="{ row }">{{ fmtTime(row.createdAt) }}</template>
        </ElTableColumn>
        <ElTableColumn label="操作" width="140">
          <template #default="{ row }">
            <ElButton
              v-if="tab === 'PENDING' && row.taskId"
              size="small"
              type="primary"
              link
              @click="openDetail(row.taskId)"
            >
              审批
            </ElButton>
          </template>
        </ElTableColumn>
      </ElTable>
    </ElCard>

    <ApprovalDetailDrawer ref="drawerRef" @success="load" />
  </div>
</template>

<script lang="ts" setup>
import { onMounted, ref } from "vue";
import {
  ElButton,
  ElCard,
  ElTable,
  ElTableColumn,
  ElTabPane,
  ElTabs,
} from "element-plus";

import ApprovalDetailDrawer from "./detail-drawer.vue";
import { fetchMyTasks } from "@/api/composables";
import type { oaservicev1_ListType } from "@/api/generated/admin/service/v1";

const tab = ref<oaservicev1_ListType>("PENDING");
const items = ref<any[]>([]);
const loading = ref(false);
const drawerRef = ref();

async function load() {
  loading.value = true;
  try {
    const resp = await fetchMyTasks(tab.value);
    items.value = resp.items ?? [];
  } finally {
    loading.value = false;
  }
}

function openDetail(taskId: number) {
  drawerRef.value?.open(taskId);
}

function fmtTime(v?: string) {
  return v ? String(v).replace("T", " ").slice(0, 19) : "-";
}

onMounted(load);
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
