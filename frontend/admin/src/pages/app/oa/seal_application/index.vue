<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ElCard shadow="never" class="flex-1">
      <ElTable :data="applications" border stripe v-loading="loadingApps">
        <ElTableColumn prop="id" label="ID" width="80" />
        <ElTableColumn prop="applicantName" label="申请人" width="120" />
        <ElTableColumn prop="purpose" label="用印事由" min-width="160" show-overflow-tooltip />
        <ElTableColumn label="印章类型" width="120">
          <template #default="{ row }">{{ sealTypeLabel(row.sealType) }}</template>
        </ElTableColumn>
        <ElTableColumn prop="fileCount" label="文件份数" width="100" />
        <ElTableColumn prop="recipient" label="收件方" width="160" show-overflow-tooltip />
        <ElTableColumn label="状态" width="100">
          <template #default="{ row }">
            <ElTag :type="statusTag(row.sealStatus)">{{ statusLabel(row.sealStatus) }}</ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="instanceId" label="流程实例" width="100" />
      </ElTable>
      <div class="pager">
        <ElPagination
          v-model:current-page="appPage"
          v-model:page-size="appPageSize"
          :total="appTotal"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @current-change="loadAll"
          @size-change="loadAll"
        />
      </div>
    </ElCard>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref } from "vue";
import {
  ElCard,
  ElPagination,
  ElTable,
  ElTableColumn,
  ElTag,
} from "element-plus";

import { useListSealApplications } from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";

const applications = ref<any[]>([]);
const loadingApps = ref(false);
const appPage = ref(1);
const appPageSize = ref(10);
const appTotal = ref(0);
const appsQuery = useListSealApplications(
  reactive({
    userId: 0,
    status: undefined,
    page: computed(() => appPage.value),
    pageSize: computed(() => appPageSize.value),
  }),
  { enabled: false }
);

async function loadAll() {
  loadingApps.value = true;
  try {
    const a = await appsQuery.refetch();
    applications.value = a.data?.items ?? [];
    appTotal.value = Number(a.data?.total ?? 0);
  } finally {
    loadingApps.value = false;
  }
}

function sealTypeLabel(s?: string): string {
  switch (s) {
    case "OFFICIAL_SEAL": return "公章";
    case "CONTRACT_SEAL": return "合同章";
    case "FINANCE_SEAL": return "财务章";
    case "LEGAL_SEAL": return "法人章";
    default: return "-";
  }
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

onMounted(loadAll);
</script>

<style scoped>
.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}
</style>
