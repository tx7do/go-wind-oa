<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ElCard shadow="never" class="flex-1">
      <ElTable :data="applications" border stripe v-loading="loadingApps">
        <ElTableColumn prop="id" label="ID" width="80" />
        <ElTableColumn prop="applicantName" label="申请人" width="120" />
        <ElTableColumn prop="title" label="标题" min-width="160" show-overflow-tooltip />
        <ElTableColumn prop="destination" label="目的地" width="160" show-overflow-tooltip />
        <ElTableColumn label="起止" min-width="200">
          <template #default="{ row }">
            {{ fmtDate(row.startDate) }} ~ {{ fmtDate(row.endDate) }}
          </template>
        </ElTableColumn>
        <ElTableColumn label="状态" width="100">
          <template #default="{ row }">
            <ElTag :type="statusTag(row.tripStatus)">{{ statusLabel(row.tripStatus) }}</ElTag>
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

import { useListBusinessTripApplications } from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";

const applications = ref<any[]>([]);
const loadingApps = ref(false);
const appPage = ref(1);
const appPageSize = ref(10);
const appTotal = ref(0);
const appsQuery = useListBusinessTripApplications(
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

function fmtDate(ts?: { seconds?: number; nanos?: number } | string): string {
  if (!ts) return "";
  let d: Date | null = null;
  if (typeof ts === "object" && ts.seconds) {
    d = new Date(Number(ts.seconds) * 1000);
  } else if (typeof ts === "string") {
    d = new Date(ts);
  }
  if (!d || isNaN(d.getTime())) return "";
  return d.toISOString().slice(0, 10);
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
