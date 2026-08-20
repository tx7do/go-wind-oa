<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ElCard shadow="never" class="flex-1">
      <ElTabs v-model="tab" @tab-change="onTabChange">
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
      <div class="pager">
        <ElPagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @current-change="load"
          @size-change="load"
        />
      </div>
    </ElCard>

    <ApprovalDetailDrawer ref="drawerRef" @success="load" />
  </div>
</template>

<script lang="ts" setup>
import { onMounted, ref } from "vue";
import {
  ElButton,
  ElCard,
  ElPagination,
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
const page = ref(1);
const pageSize = ref(10);
const total = ref(0);
const drawerRef = ref();

async function load() {
  loading.value = true;
  try {
    const resp = await fetchMyTasks(tab.value, page.value, pageSize.value);
    items.value = resp.items ?? [];
    total.value = Number(resp.total ?? 0);
  } finally {
    loading.value = false;
  }
}

function onTabChange() {
  page.value = 1;
  load();
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
.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}
</style>
