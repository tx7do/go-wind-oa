<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd">
      <!-- definition_status：枚举 Tag -->
      <template #definition_status="scope: any">
        <ElTag
          size="small"
          effect="dark"
          round
          :color="definitionStatusColor(scope.row.definition_status)"
        >
          {{ definitionStatusLabel(scope.row.definition_status) }}
        </ElTag>
      </template>
      <!-- 操作列：查看 / 启用/禁用切换 -->
      <template #operation="scope: any">
        <ElButton
          size="small"
          type="primary"
          link
          @click="handleView(scope.row)"
        >
          {{ $t("common.view") }}
        </ElButton>
        <ElButton
          v-if="scope.row.definition_status !== 'ENABLED'"
          size="small"
          type="success"
          link
          @click="handleToggleStatus(scope.row, 'ENABLED')"
        >
          {{ $t("pages.oa.definition.enable") }}
        </ElButton>
        <ElButton
          v-else
          size="small"
          type="warning"
          link
          @click="handleToggleStatus(scope.row, 'DISABLED')"
        >
          {{ $t("pages.oa.definition.disable") }}
        </ElButton>
      </template>
    </ProPage>

    <!-- 抽屉 -->
    <DefinitionDrawer ref="drawerRef" @success="handleSuccess" />
    <DetailDrawer ref="detailDrawerRef" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElTag, ElButton, ElMessageBox, ElMessage } from "element-plus";
import type {
  oaservicev1_WorkflowDefinition,
  oaservicev1_WorkflowDefinition_DefinitionStatus,
} from "@/api/generated/admin/service/v1";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";

import {
  definitionStatusList,
  definitionStatusLabel,
  definitionStatusColor,
  fetchListWorkflowDefinitions,
  useUpdateWorkflowDefinitionStatus,
} from "@/api/composables";
import { queryClient } from "@/plugins/vue-query";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";

import DefinitionDrawer from "./definition-drawer.vue";
import DetailDrawer from "./detail-drawer.vue";

const pageRef = ref();
const drawerRef = ref();
const detailDrawerRef = ref();

const toggleStatusMutation = useUpdateWorkflowDefinitionStatus({
  onSuccess: () => {
    ElMessage.success($t("common.success"));
    queryClient.invalidateQueries({ queryKey: ["listWorkflowDefinitions"] });
    pageRef.value?.refresh();
  },
  onError: (err: Error) => {
    ElMessage.error(err.message || $t("common.error"));
  },
});

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.oa.definition.searchName"),
        field: "name",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("pages.oa.definition.searchStatus"),
        field: "definition_status",
        attrs: {
          placeholder: $t("common.placeholder.select"),
          clearable: true,
          filterable: true,
        },
        options: definitionStatusList.value,
      },
    ],
  },
  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListWorkflowDefinitions(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    pagination: false,
    tableAttrs: { border: true, stripe: false },
    columns: [
      {
        prop: "name",
        label: $t("pages.oa.definition.colName"),
        minWidth: 200,
      },
      {
        prop: "code",
        label: $t("pages.oa.definition.colCode"),
        minWidth: 150,
      },
      {
        prop: "version",
        label: $t("pages.oa.definition.colVersion"),
        width: 90,
      },
      {
        prop: "definition_status",
        label: $t("pages.oa.definition.colStatus"),
        width: 120,
        slotName: "definition_status",
      },
      {
        prop: "operation",
        label: $t("pages.oa.definition.colOperation"),
        width: 160,
        slotName: "operation",
      },
      {
        prop: "created_at",
        label: $t("pages.oa.definition.colCreatedAt"),
        width: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
    ],
  },
}));

function handleAdd() {
  drawerRef.value?.open();
}

function handleView(row: oaservicev1_WorkflowDefinition) {
  detailDrawerRef.value?.open(row.id as number);
}

function handleToggleStatus(
  row: oaservicev1_WorkflowDefinition,
  target: oaservicev1_WorkflowDefinition_DefinitionStatus
) {
  const isEnable = target === "ENABLED";
  const title = isEnable
    ? $t("pages.oa.definition.enableConfirmTitle")
    : $t("pages.oa.definition.disableConfirmTitle");
  const content = isEnable
    ? $t("pages.oa.definition.enableConfirmContent")
    : $t("pages.oa.definition.disableConfirmContent");
  ElMessageBox.confirm(content, title, {
    confirmButtonText: $t("common.confirm"),
    cancelButtonText: $t("common.cancel"),
    type: isEnable ? "warning" : "error",
  })
    .then(() => {
      toggleStatusMutation.mutate({ id: row.id as number, status: target });
    })
    .catch(() => {});
}

function handleSuccess() {
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
</style>
