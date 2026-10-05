<template>
  <div class="form-editor">
    <ElTable v-if="drafts.length" :data="drafts" border size="small">
      <ElTableColumn :label="$t('pages.oa.definition.fieldKey')" width="130">
        <template #default="{ row }">
          <ElInput
            v-model="row.key"
            :placeholder="$t('pages.oa.definition.phFieldKey')"
            size="small"
          />
        </template>
      </ElTableColumn>
      <ElTableColumn :label="$t('pages.oa.definition.fieldLabel')" width="130">
        <template #default="{ row }">
          <ElInput
            v-model="row.label"
            :placeholder="$t('pages.oa.definition.phFieldLabel')"
            size="small"
          />
        </template>
      </ElTableColumn>
      <ElTableColumn :label="$t('pages.oa.definition.typeLabel')" width="120">
        <template #default="{ row }">
          <ElSelect v-model="row.type" size="small">
            <ElOption :label="$t('pages.oa.definition.fieldType.text')" value="text" />
            <ElOption :label="$t('pages.oa.definition.fieldType.textarea')" value="textarea" />
            <ElOption :label="$t('pages.oa.definition.fieldType.number')" value="number" />
            <ElOption :label="$t('pages.oa.definition.fieldType.date')" value="date" />
            <ElOption :label="$t('pages.oa.definition.fieldType.select')" value="select" />
          </ElSelect>
        </template>
      </ElTableColumn>
      <ElTableColumn :label="$t('pages.oa.definition.fieldRequired')" width="60" align="center">
        <template #default="{ row }">
          <ElCheckbox v-model="row.required" size="small" />
        </template>
      </ElTableColumn>
      <ElTableColumn :label="$t('pages.oa.definition.fieldOptions')" min-width="160">
        <template #default="{ row }">
          <ElInput
            v-if="row.type === 'select'"
            v-model="row.options"
            :placeholder="$t('pages.oa.definition.phFieldOptions')"
            size="small"
          />
          <span v-else class="hint-inline">-</span>
        </template>
      </ElTableColumn>
      <ElTableColumn :label="$t('common.table.operation')" width="60" align="center">
        <template #default="{ $index }">
          <ElButton size="small" text type="danger" @click="emit('remove', $index)">
            {{ $t("pages.oa.definition.fieldRemove") }}
          </ElButton>
        </template>
      </ElTableColumn>
    </ElTable>
    <ElButton size="small" plain class="add-field-btn" @click="emit('add')">
      {{ $t("pages.oa.definition.addField") }}
    </ElButton>
    <div class="hint">{{ $t("pages.oa.definition.formHint") }}</div>
    <div v-if="drafts.some((f) => f.key.trim())" class="json-preview">
      <div class="preview-title">{{ $t("pages.oa.definition.previewTitle") }}</div>
      <code>{{ preview }}</code>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import {
  ElButton,
  ElCheckbox,
  ElInput,
  ElOption,
  ElSelect,
  ElTable,
  ElTableColumn,
} from "element-plus";

import { fieldsToSchema, type FieldDraft } from "./workflow-graph";

const props = defineProps<{
  drafts: FieldDraft[];
}>();

const emit = defineEmits<{
  (e: "add"): void;
  (e: "remove", index: number): void;
}>();

const preview = computed(() => fieldsToSchema(props.drafts));
</script>

<style lang="scss" scoped>
.form-editor {
  width: 100%;
}

.add-field-btn {
  margin-top: 8px;
}

.hint {
  margin-top: 6px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.hint-inline {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.json-preview {
  margin-top: 8px;
  padding: 8px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  font-size: 12px;
  word-break: break-all;

  .preview-title {
    font-weight: 600;
    margin-bottom: 4px;
  }

  code {
    white-space: pre-wrap;
  }
}
</style>
