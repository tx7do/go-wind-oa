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
      <ElButton type="primary" plain @click="openSettings">
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

    <ElDialog
      v-model="settingsVisible"
      :title="$t('pages.oa.attendance.records.attendanceSettings')"
      width="420px"
    >
      <ElForm label-width="100px">
        <ElFormItem :label="$t('pages.oa.attendance.records.fieldWorkStartTime')">
          <ElInput v-model="settings.workStartTime" placeholder="09:00" />
        </ElFormItem>
        <ElFormItem :label="$t('pages.oa.attendance.records.fieldWorkEndTime')">
          <ElInput v-model="settings.workEndTime" placeholder="18:00" />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="settingsVisible = false">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="savingSettings" @click="saveSettings">
          {{ $t("common.button.save") }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, reactive } from "vue";
import {
  ElButton,
  ElDatePicker,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElTag,
} from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import {
  attendanceDayResultToName,
  attendanceDayResultToTag,
  fetchListAttendanceRecords,
  useRunDailySettlement,
  useUpdateAttendanceSetting,
} from "@/api/composables";
import { apiClient } from "@/api/client";
import { formatDate, formatDateTime } from "@/utils/date";
import { $t } from "@/core/i18n";

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

const settingsVisible = ref(false);
const savingSettings = ref(false);
const settings = reactive({ workStartTime: "09:00", workEndTime: "18:00" });

async function openSettings() {
  const resp = await apiClient.attendanceService.GetAttendanceSetting({});
  settings.workStartTime = resp?.workStartTime ?? "09:00";
  settings.workEndTime = resp?.workEndTime ?? "18:00";
  settingsVisible.value = true;
}

const settingsMutation = useUpdateAttendanceSetting({
  onSuccess: () => {
    ElMessage.success($t("common.notification.saveSuccess"));
    settingsVisible.value = false;
  },
  onError: (err: Error) => ElMessage.error(err.message || $t("common.notification.saveFailed")),
});

function saveSettings() {
  savingSettings.value = true;
  settingsMutation.mutate(
    { workStartTime: settings.workStartTime, workEndTime: settings.workEndTime },
    { onSettled: () => (savingSettings.value = false) }
  );
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
