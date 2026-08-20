<template>
  <ProModal
    v-model:visible="visible"
    :title="$t('pages.oa.definition.detailTitle')"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <ElForm label-width="140px" v-if="detail">
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
      <ElFormItem label="备注">
        <ElInput :model-value="detail.remark" type="textarea" :rows="3" readonly />
      </ElFormItem>
      <ElFormItem :label="$t('pages.oa.definition.fieldNodeConfig')">
        <ElInput
          :model-value="detail.nodeConfig"
          type="textarea"
          :rows="8"
          readonly
        />
      </ElFormItem>
      <ElFormItem :label="$t('pages.oa.definition.fieldFormSchema')">
        <ElInput
          :model-value="detail.formSchema"
          type="textarea"
          :rows="8"
          readonly
        />
      </ElFormItem>
    </ElForm>
  </ProModal>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElForm, ElFormItem, ElInput, ElTag } from "element-plus";
import ProModal from "@/components/Pro/ProModal/index.vue";
import {
  definitionStatusLabel,
  definitionStatusColor,
  fetchWorkflowDefinition,
} from "@/api/composables";
import type { oaservicev1_WorkflowDefinition } from "@/api/generated/admin/service/v1";
import { $t } from "@/core/i18n";

const DRAWER_WIDTH = "60%";
const visible = ref(false);
const detail = ref<oaservicev1_WorkflowDefinition | null>(null);

async function open(id: number) {
  try {
    detail.value = await fetchWorkflowDefinition(id);
    visible.value = true;
  } catch {
    detail.value = null;
  }
}

defineExpose({ open });
</script>
