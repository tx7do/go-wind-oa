<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <div class="status-bar">
      <span class="label">{{ $t("pages.oa.attendance.records.workDateLabel") }}</span>
      <ElDatePicker
        v-model="workDate"
        type="date"
        value-format="YYYY-MM-DD"
        :clearable="false"
        style="width: 180px"
        @change="onWorkDateChange"
      />
      <ElButton :loading="settling" @click="runSettlement">
        {{ $t("pages.oa.attendance.records.runSettlement") }}
      </ElButton>
      <ElButton type="primary" plain @click="goSettings">
        {{ $t("pages.oa.attendance.records.attendanceSettings") }}
      </ElButton>
    </div>
    <ProPage ref="pageRef" :config="pageConfig">
      <template #dayResult="scope: any">
        <ElTag :type="attendanceDayResultToTag(scope.row.dayResult)">
          {{ attendanceDayResultToName(scope.row.dayResult) }}
        </ElTag>
      </template>
    </ProPage>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElButton, ElDatePicker, ElMessage, ElTag } from "element-plus";
import { useRouter } from "vue-router";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import {
  attendanceDayResultToName,
  attendanceDayResultToTag,
  fetchListAttendanceRecords,
  useRunDailySettlement,
} from "@/api/composables";
import { formatDate, formatDateTime } from "@/utils/date";
import { $t } from "@/core/i18n";

const router = useRouter();
const pageRef = ref();

const today = () => formatDate(new Date());
const workDate = ref(today());
const settling = ref(false);

const settlementMutation = useRunDailySettlement({
  onSuccess: (resp: any) => {
    ElMessage.success(
      $t("pages.oa.attendance.records.settleSuccess", { count: resp?.settledCount ?? 0 })
    );
    pageRef.value?.refresh();
  },
  onError: (err: Error) =>
    ElMessage.error(err.message || $t("pages.oa.attendance.records.settleFailed")),
});

function runSettlement() {
  settling.value = true;
  settlementMutation.mutate(
    { workDate: `${workDate.value}T00:00:00Z` },
    { onSettled: () => (settling.value = false) }
  );
}

// 工时设置统一走独立设置页（本页不再维护第二份实现）
function goSettings() {
  router.push("/oa/attendance-setting");
}

function locateStr(row: any): string {
  if (row.checkInLatitude) {
    let s = `${row.checkInLatitude}, ${row.checkInLongitude}`;
    if (row.checkInWifiBssid) s += ` / ${row.checkInWifiBssid}`;
    return s;
  }
  return "-";
}

const pageConfig = computed<ProPageConfig>(() => ({
  table: {
    listAction: async (query: any) => {
      const result = await fetchListAttendanceRecords({
        userId: query.userId ?? 0,
        workDate: `${workDate.value}T00:00:00Z`,
        page: query.page,
        pageSize: query.pageSize,
      } as any);
      return { items: (result as any)?.items ?? [], total: (result as any)?.total ?? 0 };
    },
    toolbar: [],
    toolbarRight: [],
    defaultToolbar: ["refresh", "filter"],
    pagination: true,
    tableAttrs: { border: true, stripe: true },
    columns: [
      { prop: "userId", label: $t("pages.oa.attendance.records.colUserId"), width: 100 },
      {
        prop: "workDate",
        label: $t("pages.oa.attendance.records.colWorkDate"),
        width: 120,
        formatter: (row: any) => formatDate(row.workDate),
      },
      {
        prop: "checkInAt",
        label: $t("pages.oa.attendance.records.colCheckInAt"),
        width: 180,
        formatter: (row: any) => formatDateTime(row.checkInAt),
      },
      {
        label: $t("pages.oa.attendance.records.colCheckInLocation"),
        minWidth: 160,
        formatter: (row: any) => locateStr(row),
      },
      {
        prop: "checkOutAt",
        label: $t("pages.oa.attendance.records.colCheckOutAt"),
        width: 180,
        formatter: (row: any) => formatDateTime(row.checkOutAt),
      },
      {
        prop: "dayResult",
        label: $t("pages.oa.attendance.records.colDayResult"),
        width: 100,
        slotName: "dayResult",
      },
    ],
  },
}));

function onWorkDateChange() {
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
.status-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
}
.label {
  font-size: 14px;
  color: var(--el-text-color-regular);
}
</style>
