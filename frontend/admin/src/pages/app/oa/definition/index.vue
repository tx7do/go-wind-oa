<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd">
      <!-- definition_status：枚举 Tag -->
      <template #definition_status="scope: any">
        <ElTag
          size="small"
          effect="dark"
          round
          :color="definitionStatusColor(scope.row.definitionStatus)"
        >
          {{ definitionStatusLabel(scope.row.definitionStatus) }}
        </ElTag>
      </template>
      <!-- 操作列：查看 / 启用/禁用切换 / 导出 -->
      <template #operation="scope: any">
        <ElButton size="small" type="primary" link @click="handleView(scope.row)">
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
        <ElButton size="small" type="info" link @click="handleExport(scope.row)">导出</ElButton>
      </template>
    </ProPage>

    <!-- 抽屉 -->
    <DefinitionDrawer ref="drawerRef" @success="handleSuccess" />
    <DetailDrawer ref="detailDrawerRef" />

    <input
      ref="importInputRef"
      type="file"
      accept=".json"
      style="display: none"
      @change="handleImportFile"
    />
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
  fetchWorkflowDefinition,
  useCreateWorkflowDefinition,
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
const importInputRef = ref<HTMLInputElement>();

const importMutation = useCreateWorkflowDefinition({
  onSuccess: () => {
    ElMessage.success("导入成功");
    pageRef.value?.refresh();
  },
  onError: (err: Error) => ElMessage.error(err.message || "导入失败"),
});

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
        prop: "remark",
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
        prop: "definitionStatus",
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
        prop: "createdAt",
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
    confirmButtonText: $t("common.button.confirm"),
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

function handleExport(row: oaservicev1_WorkflowDefinition) {
  ElMessageBox.confirm("确定导出此流程定义？导出文件包含流程图配置和表单定义。", "导出确认", {
    confirmButtonText: $t("common.button.confirm"),
    cancelButtonText: $t("common.cancel"),
    type: "info",
  })
    .then(async () => {
      try {
        const def = await fetchWorkflowDefinition(row.id as number);
        const exportData = {
          code: def.code,
          version: def.version,
          remark: def.remark,
          nodeConfig: def.nodeConfig,
          formSchema: def.formSchema,
        };
        const blob = new Blob([JSON.stringify(exportData, null, 2)], {
          type: "application/json",
        });
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `workflow-${def.code}-v${def.version}.json`;
        a.click();
        URL.revokeObjectURL(url);
        ElMessage.success("已导出");
      } catch {
        ElMessage.error("导出失败");
      }
    })
    .catch(() => {});
}

function handleImportClick() {
  importInputRef.value?.click();
}

function handleImportFile(e: Event) {
  const input = e.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  const reader = new FileReader();
  reader.onload = () => {
    try {
      const parsed = JSON.parse(reader.result as string);
      if (!parsed.code || !parsed.version || !parsed.nodeConfig) {
        ElMessage.error("导入文件缺少必要字段（code/version/nodeConfig）");
        return;
      }
      ElMessageBox.confirm("确定导入此流程定义？将创建新的流程定义。", "导入确认", {
        confirmButtonText: $t("common.button.confirm"),
        cancelButtonText: $t("common.cancel"),
        type: "warning",
      })
        .then(() => {
          importMutation.mutate({
            data: {
              code: parsed.code,
              version: parsed.version,
              remark: parsed.remark || undefined,
              nodeConfig: parsed.nodeConfig,
              formSchema: parsed.formSchema || undefined,
            },
          });
        })
        .catch(() => {});
    } catch {
      ElMessage.error("导入文件解析失败");
    }
    input.value = "";
  };
  reader.readAsText(file);
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
