<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ElCard shadow="never" class="flex-1">
      <ElTable :data="applications" border stripe v-loading="loading">
        <ElTableColumn type="expand">
          <template #default="{ row }">
            <ElTable :data="row.items ?? []" border size="small">
              <ElTableColumn prop="category" label="类别" width="120" />
              <ElTableColumn prop="amount" label="金额" width="120" />
              <ElTableColumn label="费用日期" width="120">
                <template #default="{ row: item }">{{ fmtDate(item.expenseDate) }}</template>
              </ElTableColumn>
              <ElTableColumn prop="description" label="说明" min-width="160" />
              <ElTableColumn prop="invoiceFileId" label="发票文件ID" width="110" />
            </ElTable>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="id" label="ID" width="80" />
        <ElTableColumn prop="createdBy" label="申请人ID" width="100" />
        <ElTableColumn prop="title" label="事由" min-width="200" show-overflow-tooltip />
        <ElTableColumn prop="totalAmount" label="总额" width="120" />
        <ElTableColumn label="状态" width="100">
          <template #default="{ row }">
            <ElTag :type="statusTag(row.expenseStatus)">{{ statusLabel(row.expenseStatus) }}</ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="instanceId" label="流程实例" width="100" />
        <ElTableColumn prop="created_at" label="提交时间" width="170">
          <template #default="{ row }">{{ fmtTime(row.createdAt || row.created_at) }}</template>
        </ElTableColumn>
      </ElTable>
    </ElCard>
  </div>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref } from "vue";
import {
  ElCard,
  ElTable,
  ElTableColumn,
  ElTag,
} from "element-plus";

import { useListExpenseApplications } from "@/api/composables";

const applications = ref<any[]>([]);
const loading = ref(false);
const appsQuery = useListExpenseApplications(reactive({ userId: 0, status: undefined }), {
  enabled: false,
});

async function load() {
  loading.value = true;
  try {
    const resp = await appsQuery.refetch();
    applications.value = resp.data?.items ?? [];
  } finally {
    loading.value = false;
  }
}

function fmtDate(v?: string) {
  return v ? String(v).slice(0, 10) : "-";
}

function fmtTime(v?: string) {
  return v ? String(v).replace("T", " ").slice(0, 19) : "-";
}

function statusLabel(s?: string): string {
  switch (s) {
    case "APPROVED": return "已通过";
    case "REJECTED": return "已驳回";
    case "WITHDRAWN": return "已撤回";
    default: return "审批中";
  }
}

function statusTag(s?: string): "success" | "danger" | "info" | "warning" {
  switch (s) {
    case "APPROVED": return "success";
    case "REJECTED": return "danger";
    case "WITHDRAWN": return "info";
    default: return "warning";
  }
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
