<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <div class="status-bar">
      <span class="label">工作日</span>
      <ElDatePicker
        v-model="workDate"
        type="date"
        value-format="YYYY-MM-DD"
        :clearable="false"
        style="width: 180px"
        @change="onWorkDateChange"
      />
      <ElButton :loading="settling" @click="runSettlement">执行当日结算</ElButton>
      <ElButton type="primary" plain @click="openSettings">考勤设置</ElButton>
    </div>
    <ProPage ref="pageRef" :config="pageConfig">
      <template #dayResult="scope: any">
        <ElTag :type="resultTagType(scope.row.dayResult)">{{ resultLabel(scope.row.dayResult) }}</ElTag>
      </template>
    </ProPage>

    <ElDialog v-model="settingsVisible" title="考勤设置" width="420px">
      <ElForm label-width="100px">
        <ElFormItem label="上班时间">
          <ElInput v-model="settings.workStartTime" placeholder="09:00" />
        </ElFormItem>
        <ElFormItem label="下班时间">
          <ElInput v-model="settings.workEndTime" placeholder="18:00" />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="settingsVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="savingSettings" @click="saveSettings">保存</ElButton>
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
  fetchListAttendanceRecords,
  useRunDailySettlement,
  useUpdateAttendanceSetting,
} from "@/api/composables";
import { apiClient } from "@/api/client";

const pageRef = ref();

const today = () => new Date().toISOString().slice(0, 10);
const workDate = ref(today());
const settling = ref(false);

const settlementMutation = useRunDailySettlement({
  onSuccess: (resp: any) => {
    ElMessage.success(`结算完成，处理 ${resp?.settledCount ?? 0} 条记录`);
    pageRef.value?.refresh();
  },
  onError: (err: Error) => ElMessage.error(err.message || "结算失败"),
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
    ElMessage.success("已保存");
    settingsVisible.value = false;
  },
  onError: (err: Error) => ElMessage.error(err.message || "保存失败"),
});

function saveSettings() {
  savingSettings.value = true;
  settingsMutation.mutate(
    { workStartTime: settings.workStartTime, workEndTime: settings.workEndTime },
    { onSettled: () => (savingSettings.value = false) }
  );
}

function fmtDate(v?: string) {
  return v ? String(v).slice(0, 10) : "-";
}

function fmtTime(v?: string) {
  return v ? String(v).replace("T", " ").slice(0, 19) : "-";
}

function locateStr(row: any): string {
  if (row.checkInLatitude) {
    let s = `${row.checkInLatitude}, ${row.checkInLongitude}`;
    if (row.checkInWifiBssid) s += ` / ${row.checkInWifiBssid}`;
    return s;
  }
  return "-";
}

function resultLabel(r?: string): string {
  switch (r) {
    case "NORMAL": return "正常";
    case "LATE": return "迟到";
    case "EARLY_LEAVE": return "早退";
    case "ABSENT": return "旷工";
    case "ON_LEAVE": return "请假";
    default: return "待结算";
  }
}

function resultTagType(r?: string): "success" | "warning" | "danger" | "info" {
  switch (r) {
    case "NORMAL":
    case "ON_LEAVE":
      return "success";
    case "LATE":
    case "EARLY_LEAVE":
      return "warning";
    case "ABSENT":
      return "danger";
    default:
      return "info";
  }
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
      { prop: "userId", label: "用户ID", width: 100 },
      {
        prop: "workDate",
        label: "工作日",
        width: 120,
        formatter: (row: any) => fmtDate(row.workDate),
      },
      {
        prop: "checkInAt",
        label: "签到时间",
        width: 180,
        formatter: (row: any) => fmtTime(row.checkInAt),
      },
      {
        label: "签到定位",
        minWidth: 160,
        formatter: (row: any) => locateStr(row),
      },
      {
        prop: "checkOutAt",
        label: "签退时间",
        width: 180,
        formatter: (row: any) => fmtTime(row.checkOutAt),
      },
      { prop: "dayResult", label: "结果", width: 100, slotName: "dayResult" },
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
