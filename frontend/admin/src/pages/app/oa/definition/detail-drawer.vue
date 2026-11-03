<template>
  <ProModal
    v-model:visible="visible"
    :title="$t('pages.oa.definition.detailTitle')"
    :config="{ component: 'drawer', drawer: { size: '70%', closeOnClickModal: false } }"
  >
    <ElForm v-if="detail" label-width="140px">
      <ElFormItem :label="$t('pages.oa.definition.fieldCode')">
        <ElInput :model-value="detail.code" readonly />
      </ElFormItem>
      <ElFormItem :label="$t('pages.oa.definition.fieldVersion')">
        <ElInput :model-value="detail.version" readonly />
      </ElFormItem>
      <ElFormItem :label="$t('pages.oa.definition.colStatus')">
        <ElTag
          size="small"
          effect="dark"
          round
          :color="definitionStatusColor(detail.definitionStatus)"
        >
          {{ definitionStatusLabel(detail.definitionStatus) }}
        </ElTag>
      </ElFormItem>
      <ElFormItem :label="$t('pages.oa.definition.fieldRemark')">
        <ElInput :model-value="detail.remark" type="textarea" :rows="3" readonly />
      </ElFormItem>
      <ElFormItem :label="$t('pages.oa.definition.fieldNodeConfig')">
        <FlowDesigner
          v-if="previewDocument"
          height="420px"
          :document="previewDocument"
        />
        <ElInput v-else :model-value="detail.nodeConfig" type="textarea" :rows="8" readonly />
      </ElFormItem>
      <ElCollapse v-if="detail.nodeConfig || detail.formSchema">
        <ElCollapseItem title="JSON">
          <ElFormItem v-if="detail.nodeConfig" :label="$t('pages.oa.definition.fieldNodeConfig')">
            <ElInput :model-value="detail.nodeConfig" type="textarea" :rows="6" readonly />
          </ElFormItem>
          <ElFormItem v-if="detail.formSchema" :label="$t('pages.oa.definition.fieldFormSchema')">
            <ElInput :model-value="detail.formSchema" type="textarea" :rows="6" readonly />
          </ElFormItem>
        </ElCollapseItem>
      </ElCollapse>
      <ElFormItem v-if="!detail.nodeConfig && detail.formSchema" :label="$t('pages.oa.definition.fieldFormSchema')">
        <ElInput :model-value="detail.formSchema" type="textarea" :rows="8" readonly />
      </ElFormItem>
    </ElForm>
  </ProModal>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElCollapse, ElCollapseItem, ElForm, ElFormItem, ElInput, ElMessage, ElTag } from "element-plus";
import ProModal from "@/components/Pro/ProModal/index.vue";
import {
  definitionStatusLabel,
  definitionStatusColor,
  fetchWorkflowDefinition,
} from "@/api/composables";
import type { oaservicev1_WorkflowDefinition } from "@/api/generated/admin/service/v1";
import { $t } from "@/core/i18n";

import { graphToDesigner, parseNodeConfig } from "./flow-designer-convert";
import "@go-wind/flow-designer-vue/style.css";
import { FlowDesigner } from "@go-wind/flow-designer-vue";
import type { DesignerDocument } from "@go-wind/flow-designer-core";

const visible = ref(false);
const detail = ref<oaservicev1_WorkflowDefinition | null>(null);
const previewDocument = ref<DesignerDocument | null>(null);

async function open(id: number) {
  try {
    detail.value = await fetchWorkflowDefinition(id);
    previewDocument.value = buildPreview(detail.value);
    visible.value = true;
  } catch {
    detail.value = null;
    previewDocument.value = null;
    ElMessage.error($t("common.message.getDetailFailed"));
  }
}

/** node_config → 设计器只读预览文档；解析失败（非法 JSON）回退纯文本展示。 */
function buildPreview(d: oaservicev1_WorkflowDefinition): DesignerDocument | null {
  if (!d.nodeConfig) return null;
  try {
    return graphToDesigner(parseNodeConfig(d.nodeConfig), d.code ?? "", "");
  } catch {
    return null;
  }
}

defineExpose({ open });
</script>
