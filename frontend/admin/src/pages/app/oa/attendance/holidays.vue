<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage
      ref="pageRef"
      :config="pageConfig"
      @add="handleAdd"
    >
      <template #holidayType="scope: any">
        <ElTag :type="holidayTypeTag(scope.row.holidayType)">{{ holidayTypeLabel(scope.row.holidayType) }}</ElTag>
      </template>
      <template #operation="scope: any">
        <ElButton size="small" type="danger" link @click="handleDelete(scope.row)">删除</ElButton>
      </template>
    </ProPage>

    <HolidayDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElButton, ElMessageBox, ElMessage, ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import HolidayDrawer from "./holiday-drawer.vue";
import {
  holidayTypeLabel,
  holidayTypeTag,
  fetchListHolidays,
  useDeleteHoliday,
} from "@/api/composables";

const pageRef = ref();
const drawerRef = ref();
const year = ref(String(new Date().getFullYear()));

const deleteMutation = useDeleteHoliday({
  onSuccess: () => {
    ElMessage.success("已删除");
    pageRef.value?.refresh();
  },
  onError: (err: Error) => ElMessage.error(err.message || "删除失败"),
});

function handleDelete(row: any) {
  ElMessageBox.confirm(`确认删除 ${fmtDate(row.date)} 的设置？`, "删除", { type: "warning" })
    .then(() => deleteMutation.mutate({ id: row.id as number }))
    .catch(() => {});
}

function handleAdd() {
  drawerRef.value?.open();
}

function handleSuccess() {
  pageRef.value?.refresh();
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

const pageConfig = computed<ProPageConfig>(() => ({
  table: {
    listAction: async () => {
      const result = await fetchListHolidays({ year: Number(year.value) });
      return (result as any)?.items ?? [];
    },
    deleteAction: async (ids: string) => {
      await deleteMutation.mutateAsync({ id: Number(ids) });
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    pagination: false,
    tableAttrs: { border: true, stripe: true },
    columns: [
      {
        prop: "date",
        label: "日期",
        width: 140,
        formatter: (row: any) => fmtDate(row.date),
      },
      {
        prop: "weekday",
        label: "星期",
        width: 90,
        formatter: (row: any) => weekdayLabel(row.date),
      },
      { prop: "holidayType", label: "类型", width: 120, slotName: "holidayType" },
      { prop: "name", label: "名称", minWidth: 160 },
      { prop: "operation", label: "操作", width: 100, slotName: "operation" },
    ],
  },
}));
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
