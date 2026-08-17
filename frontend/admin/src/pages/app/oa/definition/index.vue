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
    </ProPage>

    <!-- 抽屉 -->
    <DefinitionDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";

import {
  definitionStatusLabel,
  definitionStatusColor,
  fetchListWorkflowDefinitions,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";

import DefinitionDrawer from "./definition-drawer.vue";

const pageRef = ref();
const drawerRef = ref();

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
        options: [],
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
