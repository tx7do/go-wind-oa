<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig">
      <template #overtimeStatus="scope: any">
        <ElTag :type="workflowInstanceStatusToTag(scope.row.overtimeStatus)">
          {{ workflowInstanceStatusToName(scope.row.overtimeStatus) }}
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
import type { oaservicev1_ListOvertimeApplicationsRequest } from "@/api/generated/admin/service/v1";
import {
  fetchListOvertimeApplications,
  workflowInstanceStatusToName,
  workflowInstanceStatusToTag,
} from "@/api/composables";
import { formatDate } from "@/utils/date";
import { $t } from "@/core/i18n";

const pageRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({
  table: {
    listAction: async (query: any) => {
      const req: oaservicev1_ListOvertimeApplicationsRequest = {
        userId: query.userId ?? 0,
        status: query.status,
        page: query.page,
        pageSize: query.pageSize,
      };
      const result = await fetchListOvertimeApplications(req);
      return { items: (result as any)?.items ?? [], total: (result as any)?.total ?? 0 };
    },
    toolbar: [],
    toolbarRight: [],
    defaultToolbar: ["refresh", "filter"],
    pagination: true,
    tableAttrs: { border: true, stripe: true },
    columns: [
      { prop: "id", label: "ID", width: 80 },
      { prop: "applicantName", label: $t("pages.oa.overtime.colApplicant"), width: 120 },
      {
        prop: "reason",
        label: $t("pages.oa.overtime.colReason"),
        minWidth: 160,
        showOverflowTooltip: true,
      },
      {
        label: $t("pages.oa.overtime.colDateRange"),
        minWidth: 200,
        formatter: (row: any) => `${formatDate(row.startTime)} ~ ${formatDate(row.endTime)}`,
      },
      {
        prop: "overtimeStatus",
        label: $t("pages.oa.overtime.colStatus"),
        width: 100,
        slotName: "overtimeStatus",
      },
      { prop: "instanceId", label: $t("pages.oa.overtime.colInstanceId"), width: 100 },
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
