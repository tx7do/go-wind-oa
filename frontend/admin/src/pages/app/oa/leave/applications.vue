<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig">
      <template #leaveStatus="scope: any">
        <ElTag :type="workflowInstanceStatusToTag(scope.row.leaveStatus)">
          {{ workflowInstanceStatusToName(scope.row.leaveStatus) }}
        </ElTag>
      </template>
    </ProPage>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import {
  fetchListLeaveApplications,
  workflowInstanceStatusOptions,
  workflowInstanceStatusToName,
  workflowInstanceStatusToTag,
} from "@/api/composables";
import type { oaservicev1_ListLeaveApplicationsRequest } from "@/api/generated/admin/service/v1";
import { formatDate } from "@/utils/date";
import { $t } from "@/core/i18n";

const pageRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({
  search: {
    grid: true,
    fields: [
      {
        type: "select",
        label: $t("pages.oa.leave.applications.searchStatus"),
        field: "status",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true },
        options: workflowInstanceStatusOptions.value,
      },
    ],
  },
  table: {
    listAction: async (query: any) => {
      const req: oaservicev1_ListLeaveApplicationsRequest = {
        userId: 0,
        status: query.status,
        page: query.page,
        pageSize: query.pageSize,
      };
      const result = await fetchListLeaveApplications(req);
      return { items: (result as any)?.items ?? [], total: (result as any)?.total ?? 0 };
    },
    toolbar: [],
    toolbarRight: [],
    defaultToolbar: ["refresh", "filter"],
    pagination: true,
    tableAttrs: { border: true, stripe: true },
    columns: [
      { prop: "id", label: "ID", width: 80 },
      { prop: "createdBy", label: $t("pages.oa.leave.applications.colCreatedBy"), width: 100 },
      { prop: "leaveTypeName", label: $t("pages.oa.leave.applications.colType"), width: 120 },
      {
        label: $t("pages.oa.leave.applications.colDateRange"),
        minWidth: 200,
        formatter: (row: any) => `${formatDate(row.startDate)} ~ ${formatDate(row.endDate)}`,
      },
      { prop: "days", label: $t("pages.oa.leave.applications.colDays"), width: 80 },
      {
        prop: "reason",
        label: $t("pages.oa.leave.applications.colReason"),
        minWidth: 160,
        showOverflowTooltip: true,
      },
      {
        prop: "leaveStatus",
        label: $t("pages.oa.leave.applications.colStatus"),
        width: 100,
        slotName: "leaveStatus",
      },
      { prop: "instanceId", label: $t("pages.oa.leave.applications.colInstanceId"), width: 100 },
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
