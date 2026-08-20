<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ElCard shadow="never" class="flex-1">
      <div class="flex items-center justify-between mb-4">
        <div class="flex items-center gap-3">
          <span class="label">年度</span>
          <ElDatePicker
            v-model="year"
            type="year"
            value-format="YYYY"
            :clearable="false"
            style="width: 120px"
            @change="load"
          />
          <ElButton :loading="loading" @click="load">查询</ElButton>
        </div>
        <ElButton type="primary" @click="dialogVisible = true">新增节假日/调休</ElButton>
      </div>

      <ElTable :data="items" border stripe v-loading="loading">
        <ElTableColumn prop="date" label="日期" width="140">
          <template #default="{ row }">{{ fmtDate(row.date) }}</template>
        </ElTableColumn>
        <ElTableColumn prop="weekday" label="星期" width="90">
          <template #default="{ row }">{{ weekdayLabel(row.date) }}</template>
        </ElTableColumn>
        <ElTableColumn label="类型" width="120">
          <template #default="{ row }">
            <ElTag :type="holidayTypeTag(row.holidayType)">{{ holidayTypeLabel(row.holidayType) }}</ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="name" label="名称" min-width="160" />
        <ElTableColumn label="操作" width="100">
          <template #default="{ row }">
            <ElButton size="small" type="danger" link @click="remove(row)">删除</ElButton>
          </template>
        </ElTableColumn>
      </ElTable>
      <div class="hint">法定假日=该日休息（即使工作日）；调休上班=该日照常结算（即使周末）。未设置的日期按周末判定。</div>
    </ElCard>

    <ElDialog v-model="dialogVisible" title="设置节假日/调休" width="420px">
      <ElForm label-width="90px">
        <ElFormItem label="日期">
          <ElDatePicker v-model="form.date" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
        </ElFormItem>
        <ElFormItem label="类型">
          <ElRadioGroup v-model="form.holidayType">
            <ElRadio value="HOLIDAY">法定假日（休息）</ElRadio>
            <ElRadio value="WORKDAY">调休上班</ElRadio>
          </ElRadioGroup>
        </ElFormItem>
        <ElFormItem label="名称">
          <ElInput v-model="form.name" placeholder="如：国庆节 / 调休上班" />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="dialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="saving" @click="save">保存</ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref } from "vue";
import {
  ElButton,
  ElCard,
  ElDatePicker,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElMessageBox,
  ElRadio,
  ElRadioGroup,
  ElTable,
  ElTableColumn,
  ElTag,
} from "element-plus";

import type { oaservicev1_Holiday_HolidayType } from "@/api/generated/admin/service/v1";
import {
  holidayTypeLabel,
  holidayTypeTag,
  useDeleteHoliday,
  useListHolidays,
  useUpsertHoliday,
} from "@/api/composables";

const year = ref(String(new Date().getFullYear()));
const items = ref<any[]>([]);
const loading = ref(false);
const holidaysQuery = useListHolidays(
  { year: Number(year.value) },
  { enabled: false }
);

async function load() {
  loading.value = true;
  try {
    const resp = await holidaysQuery.refetch();
    items.value = (resp.data as any)?.items ?? [];
  } finally {
    loading.value = false;
  }
}

const dialogVisible = ref(false);
const saving = ref(false);
const form = reactive({ date: "", holidayType: "HOLIDAY", name: "" });

const upsertMutation = useUpsertHoliday({
  onSuccess: () => {
    ElMessage.success("已保存");
    dialogVisible.value = false;
    load();
  },
  onError: (err: Error) => ElMessage.error(err.message || "保存失败"),
});

function save() {
  if (!form.date) {
    ElMessage.warning("请选择日期");
    return;
  }
  saving.value = true;
  upsertMutation.mutate(
    { date: `${form.date}T00:00:00Z`, holidayType: form.holidayType as oaservicev1_Holiday_HolidayType, name: form.name },
    { onSettled: () => (saving.value = false) }
  );
}

const deleteMutation = useDeleteHoliday({
  onSuccess: () => {
    ElMessage.success("已删除");
    load();
  },
  onError: (err: Error) => ElMessage.error(err.message || "删除失败"),
});

function remove(row: any) {
  ElMessageBox.confirm(`确认删除 ${fmtDate(row.date)} 的设置？`, "删除", { type: "warning" })
    .then(() => deleteMutation.mutate({ id: row.id as number }))
    .catch(() => {});
}

// 时间戳为 UTC 瞬间，渲染须转本地时区（直接切片会差一天）。
function localDate(v?: string): Date | null {
  return v ? new Date(v) : null;
}

function fmtDate(v?: string) {
  const d = localDate(v);
  return d ? d.toLocaleDateString("sv-SE") : "-";
}

const weekdays = ["周日", "周一", "周二", "周三", "周四", "周五", "周六"];
function weekdayLabel(v?: string) {
  const d = localDate(v);
  return d ? weekdays[d.getDay()] : "-";
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
.label {
  font-size: 14px;
  color: var(--el-text-color-regular);
}
.hint {
  margin-top: 10px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
