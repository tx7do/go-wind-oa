<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd">
      <template #holidayType="scope: any">
        <ElTag :type="holidayTypeTag(scope.row.holidayType)">
          {{ holidayTypeLabel(scope.row.holidayType) }}
        </ElTag>
      </template>
      <template #operation="scope: any">
        <ElButton size="small" type="danger" link @click="handleDelete(scope.row)">
          {{ $t("common.button.delete") }}
        </ElButton>
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
import { i18n, $t } from "@/core/i18n";
import { formatDate } from "@/utils/date";

const pageRef = ref();
const drawerRef = ref();
const year = ref(String(new Date().getFullYear()));

const deleteMutation = useDeleteHoliday({
  onSuccess: () => {
    ElMessage.success($t("common.notification.deleteSuccess"));
    pageRef.value?.refresh();
  },
  onError: (err: Error) => ElMessage.error(err.message || $t("common.notification.deleteFailed")),
});

function handleDelete(row: any) {
  ElMessageBox.confirm(
    $t("pages.oa.attendance.holidays.deleteConfirmContent", { date: fmtDate(row.date) }),
    $t("common.dialog.confirm"),
    { type: "warning" }
  )
    .then(() => deleteMutation.mutate({ id: row.id as number }))
    .catch(() => {});
}

function handleAdd() {
  drawerRef.value?.open();
}

function handleSuccess() {
  pageRef.value?.refresh();
}

// formatDate 统一按上海时区取日期（直接切片会随浏览器时区漂移差一天），
// 星期名通过 Intl 跟随应用语言。
function fmtDate(v?: string) {
  return formatDate(v) || "-";
}

function weekdayLabel(v?: string): string {
  if (!v) return "-";
  return new Date(`${formatDate(v)}T00:00:00Z`).toLocaleDateString(i18n.global.locale.value, {
    weekday: "long",
    timeZone: "UTC",
  });
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
        label: $t("pages.oa.attendance.holidays.colDate"),
        width: 140,
        formatter: (row: any) => fmtDate(row.date),
      },
      {
        prop: "weekday",
        label: $t("pages.oa.attendance.holidays.colWeekday"),
        width: 90,
        formatter: (row: any) => weekdayLabel(row.date),
      },
      {
        prop: "holidayType",
        label: $t("pages.oa.attendance.holidays.colType"),
        width: 120,
        slotName: "holidayType",
      },
      { prop: "name", label: $t("pages.oa.attendance.holidays.colName"), minWidth: 160 },
      {
        prop: "operation",
        label: $t("pages.oa.attendance.holidays.colOperation"),
        width: 100,
        slotName: "operation",
      },
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
