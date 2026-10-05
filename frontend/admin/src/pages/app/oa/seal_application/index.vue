<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig">
      <template #sealStatus="scope: any">
        <ElTag :type="workflowInstanceStatusToTag(scope.row.sealStatus)">
          {{ workflowInstanceStatusToName(scope.row.sealStatus) }}
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
import type { oaservicev1_ListSealApplicationsRequest } from "@/api/generated/admin/service/v1";
import {
  fetchListSealApplications,
  sealTypeToName,
  workflowInstanceStatusToName,
  workflowInstanceStatusToTag,
} from "@/api/composables";
import { $t } from "@/core/i18n";

const pageRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({
  table: {
    listAction: async (query: any) => {
      const req: oaservicev1_ListSealApplicationsRequest = {
        userId: query.userId ?? 0,
        status: query.status,
        page: query.page,
        pageSize: query.pageSize,
      };
      const result = await fetchListSealApplications(req);
      return { items: (result as any)?.items ?? [], total: (result as any)?.total ?? 0 };
    },
    toolbar: [],
    toolbarRight: [],
    defaultToolbar: ["refresh", "filter"],
    pagination: true,
    tableAttrs: { border: true, stripe: true },
    columns: [
      { prop: "id", label: "ID", width: 80 },
      { prop: "applicantName", label: $t("pages.oa.seal.colApplicant"), width: 120 },
      {
        prop: "purpose",
        label: $t("pages.oa.seal.colPurpose"),
        minWidth: 160,
        showOverflowTooltip: true,
      },
      {
        prop: "sealType",
        label: $t("pages.oa.seal.colSealType"),
        width: 120,
        formatter: (row: any) => sealTypeToName(row.sealType),
      },
      { prop: "fileCount", label: $t("pages.oa.seal.colFileCount"), width: 100 },
      {
        prop: "recipient",
        label: $t("pages.oa.seal.colRecipient"),
        width: 160,
        showOverflowTooltip: true,
      },
      {
        prop: "sealStatus",
        label: $t("pages.oa.seal.colStatus"),
        width: 100,
        slotName: "sealStatus",
      },
      { prop: "instanceId", label: $t("pages.oa.seal.colInstanceId"), width: 100 },
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
