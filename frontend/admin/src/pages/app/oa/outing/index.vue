<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig">
      <template #outingStatus="scope: any">
        <ElTag :type="workflowInstanceStatusToTag(scope.row.outingStatus)">
          {{ workflowInstanceStatusToName(scope.row.outingStatus) }}
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
import type { oaservicev1_ListOutingApplicationsRequest } from "@/api/generated/admin/service/v1";
import {
  fetchListOutingApplications,
  workflowInstanceStatusOptions,
  workflowInstanceStatusToName,
  workflowInstanceStatusToTag,
} from "@/api/composables";
import { formatDate } from "@/utils/date";
import { $t } from "@/core/i18n";

const pageRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({
  search: {
    grid: true,
    fields: [
      {
        type: "select",
        label: $t("pages.oa.outing.searchStatus"),
        field: "status",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true },
        options: workflowInstanceStatusOptions.value,
      },
    ],
  },
  table: {
    listAction: async (query: any) => {
      const req: oaservicev1_ListOutingApplicationsRequest = {
        userId: query.userId ?? 0,
        status: query.status,
        page: query.page,
        pageSize: query.pageSize,
      };
      const result = await fetchListOutingApplications(req);
      return { items: (result as any)?.items ?? [], total: (result as any)?.total ?? 0 };
    },
    toolbar: [],
    toolbarRight: [],
    defaultToolbar: ["refresh", "filter"],
    pagination: true,
    tableAttrs: { border: true, stripe: true },
    columns: [
      { prop: "id", label: "ID", width: 80 },
      { prop: "applicantName", label: $t("pages.oa.outing.colApplicant"), width: 120 },
      {
        prop: "destination",
        label: $t("pages.oa.outing.colDestination"),
        width: 160,
        showOverflowTooltip: true,
      },
      {
        label: $t("pages.oa.outing.colDateRange"),
        minWidth: 200,
        formatter: (row: any) => `${formatDate(row.startTime)} ~ ${formatDate(row.endTime)}`,
      },
      {
        prop: "outingStatus",
        label: $t("pages.oa.outing.colStatus"),
        width: 100,
        slotName: "outingStatus",
      },
      { prop: "instanceId", label: $t("pages.oa.outing.colInstanceId"), width: 100 },
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
