<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ElCard shadow="never">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <span class="label">工作日</span>
          <ElDatePicker
            v-model="workDate"
            type="date"
            value-format="YYYY-MM-DD"
            :clearable="false"
            style="width: 180px"
            @change="loadRecords"
          />
          <ElButton :loading="loading" @click="loadRecords">查询</ElButton>
          <ElButton type="primary" plain :loading="settling" @click="runSettlement">
            执行当日结算
          </ElButton>
        </div>
        <ElButton type="primary" plain @click="openSettings">考勤设置</ElButton>
      </div>
    </ElCard>

    <ElCard shadow="never" class="mt-4 flex-1">
      <ElTable :data="records" border stripe v-loading="loading">
        <ElTableColumn prop="userId" label="用户ID" width="100" />
        <ElTableColumn prop="workDate" label="工作日" width="120">
          <template #default="{ row }">{{ fmtDate(row.workDate) }}</template>
        </ElTableColumn>
        <ElTableColumn prop="checkInAt" label="签到时间" width="180">
          <template #default="{ row }">{{ fmtTime(row.checkInAt) }}</template>
        </ElTableColumn>
        <ElTableColumn label="签到定位" min-width="160">
          <template #default="{ row }">
            <span v-if="row.checkInLatitude">
              {{ row.checkInLatitude }}, {{ row.checkInLongitude }}
              <span v-if="row.checkInWifiBssid"> / {{ row.checkInWifiBssid }}</span>
            </span>
            <span v-else>-</span>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="checkOutAt" label="签退时间" width="180">
          <template #default="{ row }">{{ fmtTime(row.checkOutAt) }}</template>
        </ElTableColumn>
        <ElTableColumn label="结果" width="100">
          <template #default="{ row }">
            <ElTag :type="resultTagType(row.dayResult)">{{ resultLabel(row.dayResult) }}</ElTag>
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
          @current-change="loadRecords"
          @size-change="loadRecords"
        />
      </div>
    </ElCard>

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
import { computed, onMounted, reactive, ref } from "vue";
import {
  ElButton,
  ElCard,
  ElDatePicker,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElPagination,
  ElTable,
  ElTableColumn,
  ElTag,
} from "element-plus";

import {
  useListAttendanceRecords,
  useRunDailySettlement,
  useUpdateAttendanceSetting,
} from "@/api/composables";
import { apiClient } from "@/api/client";

const today = () => new Date().toISOString().slice(0, 10);

const workDate = ref(today());
const records = ref<any[]>([]);
const loading = ref(false);
const page = ref(1);
const pageSize = ref(10);
const total = ref(0);

const recordsQuery = useListAttendanceRecords(
  reactive({
    userId: 0,
    workDate: computed(() => `${workDate.value}T00:00:00Z`),
    page: computed(() => page.value),
    pageSize: computed(() => pageSize.value),
  }),
  { enabled: false }
);

async function loadRecords() {
  loading.value = true;
  try {
    const resp = await recordsQuery.refetch();
    records.value = resp.data?.items ?? [];
    total.value = Number(resp.data?.total ?? 0);
  } finally {
    loading.value = false;
  }
}

const settling = ref(false);
const settlementMutation = useRunDailySettlement({
  onSuccess: (resp: any) => {
    ElMessage.success(`结算完成，处理 ${resp?.settledCount ?? 0} 条记录`);
    loadRecords();
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

onMounted(loadRecords);
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
.label {
  font-size: 14px;
  color: var(--el-text-color-regular);
}
.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}
</style>
