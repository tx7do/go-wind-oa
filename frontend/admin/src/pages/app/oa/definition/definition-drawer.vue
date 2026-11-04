<template>
  <ProModal
    v-model:visible="visible"
    :title="title"
    :config="{ component: 'drawer', drawer: { size: '100%', closeOnClickModal: false } }"
  >
    <ElForm ref="formRef" :model="formData" :rules="formRules" label-width="120px">
      <ElFormItem :label="$t('pages.oa.definition.fieldCode')" prop="code">
        <ElInput
          v-model="formData.code"
          :placeholder="$t('pages.oa.definition.phCode')"
          style="width: 320px"
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.oa.definition.fieldVersion')" prop="version">
        <ElInputNumber
          v-model="formData.version"
          :min="1"
          :max="9999"
          controls-position="right"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.oa.definition.fieldRemark')">
        <ElInput
          v-model="formData.remark"
          :placeholder="$t('pages.oa.definition.phRemark')"
          :rows="2"
          type="textarea"
        />
      </ElFormItem>

      <!-- ============ 审批流（node_config） ============ -->
      <ElFormItem :label="$t('pages.oa.definition.graphSection')">
        <div class="editor-switch">
          <ElRadioGroup v-model="nodeMode" size="small" @change="onNodeModeChange">
            <ElRadioButton value="visual">{{ $t("pages.oa.definition.visualMode") }}</ElRadioButton>
            <ElRadioButton value="json">JSON</ElRadioButton>
          </ElRadioGroup>
        </div>

        <div v-if="nodeMode === 'visual'" class="designer-host">
          <FlowDesigner
            ref="designerRef"
            height="52vh"
            :theme="designerTheme"
            :document="designerDocument"
            @ready="onDesignerReady"
            @error="onDesignerError"
          />

        </div>

        <ElInput
          v-else
          v-model="formData.node_config"
          type="textarea"
          placeholder='{"version":2,"nodes":[{"id":"start","type":"START"},...],"edges":[...]}'
          :rows="12"
        />
      </ElFormItem>

      <!-- ============ 申请表单（form_schema） ============ -->
      <ElFormItem :label="$t('pages.oa.definition.formSection')">
        <div class="editor-switch">
          <ElRadioGroup v-model="formMode" size="small" @change="onFormModeChange">
            <ElRadioButton value="visual">{{ $t("pages.oa.definition.visualMode") }}</ElRadioButton>
            <ElRadioButton value="json">JSON</ElRadioButton>
          </ElRadioGroup>
        </div>

        <FormSchemaEditor
          v-if="formMode === 'visual'"
          :drafts="formFieldDrafts"
          @add="onAddField"
          @remove="onRemoveField"
        />

        <ElInput
          v-else
          v-model="formData.form_schema"
          type="textarea"
          placeholder='[{"key":"reason","label":"事由","type":"textarea","required":true}]（可留空）'
          :rows="6"
        />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="handleClose">{{ $t("pages.oa.definition.cancel") }}</ElButton>
        <ElButton type="primary" :loading="loading" @click="handleSubmit">
          {{ $t("pages.oa.definition.submit") }}
        </ElButton>
      </div>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import { ElMessage, type FormInstance, type FormRules } from "element-plus";
import { ElButton, ElInput, ElRadioButton, ElRadioGroup } from "element-plus";
import { nextTick, ref, computed, watch } from "vue";

import { usePreferences } from "@/core/preferences";


import { useCreateWorkflowDefinition } from "@/api/composables";
import { $t } from "@/core/i18n";
import ProModal from "@/components/Pro/ProModal/index.vue";

import FormSchemaEditor from "./form-schema-editor.vue";
import {
  designerToNodeConfig,
  graphToDesigner,
  parseNodeConfig,
  type V2Graph,
} from "./flow-designer-convert";
import { fieldsToSchema, schemaToFields, type FieldDraft } from "./workflow-graph";
import "bpmn-js/dist/assets/diagram-js.css";
import "bpmn-js/dist/assets/bpmn-js.css";
import "bpmn-js/dist/assets/bpmn-font/css/bpmn.css";
import "@bpmn-io/properties-panel/dist/assets/properties-panel.css";
import "@go-wind/flow-designer-vue/style.css";
import { FlowDesigner } from "@go-wind/flow-designer-vue";
import type { DesignerDocument, DesignerHandle } from "@go-wind/flow-designer-core";

const emit = defineEmits(["success"]);

const { mutateAsync: createDefinition } = useCreateWorkflowDefinition();
// 设计器主题跟随应用暗色模式（FlowDesigner 默认 auto 跟随系统，与应用主题脱节）。
const { theme: designerTheme } = usePreferences();

const visible = ref(false);
const loading = ref(false);
const formRef = ref<FormInstance>();

const nodeMode = ref<"visual" | "json">("visual");
const formMode = ref<"visual" | "json">("visual");
const formFieldDrafts = ref<FieldDraft[]>([]);

// go-wind-flow 官方设计器（bpmn-js 内核）：ready 后持有句柄，
// 模式切换/重开抽屉经 handle.loadDocument 推送画布内容。
const designerRef = ref();
const designerHandle = ref<DesignerHandle | null>(null);
const designerDocument = ref<DesignerDocument>(defaultDesignerDocument());

const formData = reactive({
  code: "",
  version: 1,
  remark: "",
  node_config: "",
  form_schema: "",
});

const formRules: FormRules = {
  code: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  version: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  form_schema: [
    {
      validator: (_rule: unknown, value: string, callback: (err?: Error) => void) => {
        if (!value) return callback(); // 允许为空（无表单定义的流程）
        let fields: any[];
        try {
          const parsed = JSON.parse(value);
          fields = Array.isArray(parsed) ? parsed : [parsed];
        } catch {
          callback(new Error($t("pages.oa.definition.vFormSchemaJson")));
          return;
        }
        for (const f of fields) {
          if (!f.key || !f.label) {
            callback(new Error($t("pages.oa.definition.vFieldKeyLabel")));
            return;
          }
          if (!["text", "textarea", "number", "date", "select"].includes(f.type || "text")) {
            callback(new Error($t("pages.oa.definition.vFieldTypeInvalid", { key: f.key })));
            return;
          }
          if (f.type === "select" && (!Array.isArray(f.options) || f.options.length === 0)) {
            callback(new Error($t("pages.oa.definition.vFieldOptionsEmpty", { key: f.key })));
            return;
          }
        }
        callback();
      },
      trigger: "blur",
    },
  ],
  node_config: [
    { required: true, message: $t("common.validation.required"), trigger: "blur" },
    {
      validator: (_rule: unknown, value: string, callback: (err?: Error) => void) => {
        try {
          parseNodeConfig(value);
        } catch (err) {
          callback(new Error((err as Error).message || $t("pages.oa.definition.vNodeConfigJson")));
          return;
        }
        callback();
      },
      trigger: "blur",
    },
  ],
};

const title = computed(() =>
  $t("common.modal.create", { moduleName: $t("pages.oa.definition.title") })
);

/** 新建默认文档：开始 → 主管审批（会签）→ 结束。 */
function defaultDesignerDocument(): DesignerDocument {
  const seed: V2Graph = {
    version: 2,
    nodes: [
      { id: "start", type: "START" },
      { id: "node_0", type: "TASK", approvers: [{ type: "LEADER" }], strategy: "ALL" },
      { id: "end", type: "END" },
    ],
    edges: [
      { from: "start", to: "node_0" },
      { from: "node_0", to: "end" },
    ],
  };
  return graphToDesigner(seed, "new_process", "");
}

function onDesignerReady(handle: DesignerHandle) {
  designerHandle.value = handle;
  refreshDesignerCanvas();
}

function onDesignerError(_err: unknown) {
  ElMessage.error($t("common.message.loadFailed"));
}

/** 把当前 designerDocument 推送到画布（抽屉重开/模式切换共用）。 */
function refreshDesignerCanvas() {
  const handle = designerHandle.value;
  if (!handle) return;
  handle
    .loadDocument(designerDocument.value)
    .then(() => handle.zoom("fit-viewport"))
    .catch((err: unknown) => {
      console.error("[flow-designer] loadDocument failed:", err);
      ElMessage.error(`画布加载失败：${(err as Error).message ?? err}`);
    });
}

/** 可视化画布 → node_config 文本（含 OA 子集转换校验）。 */
async function serializeDesigner(): Promise<string> {
  const rawDoc = designerHandle.value
    ? await designerHandle.value.getDocument(formData.code || "new_process")
    : designerDocument.value;
  return designerToNodeConfig(rawDoc);
}

function onAddField() {
  formFieldDrafts.value.push({
    key: "",
    label: "",
    type: "text",
    required: false,
    options: "",
  });
}

function onRemoveField(index: number) {
  formFieldDrafts.value.splice(index, 1);
}

// ============ 模式切换 ============

function onNodeModeChange(mode: string | number | boolean | undefined) {
  if (mode === "json") {
    // 可视化 → JSON：导出当前画布。
    void serializeDesigner()
      .then((cfg) => {
        formData.node_config = cfg;
      })
      .catch((err: Error) => {
        ElMessage.error(err.message);
        nodeMode.value = "visual";
      });
    return;
  }
  // JSON → 可视化：解析文本并推送画布；解析失败退回 JSON。
  try {
    designerDocument.value = graphToDesigner(
      parseNodeConfig(formData.node_config),
      formData.code || "new_process",
    );
  } catch (err) {
    ElMessage.error(`画布加载失败：${(err as Error).message}`);
    nodeMode.value = "json";
    return;
  }
  void nextTick(() => refreshDesignerCanvas());
}

function onFormModeChange(mode: string | number | boolean | undefined) {
  if (mode === "visual") {
    const drafts = schemaToFields(formData.form_schema);
    if (drafts.length) formFieldDrafts.value = drafts;
  } else {
    formData.form_schema = fieldsToSchema(formFieldDrafts.value);
  }
}

// ============ 生命周期与提交 ============

function resetForm() {
  formData.code = "";
  formData.version = 1;
  formData.remark = "";
  formData.node_config = "";
  formData.form_schema = "";
  nodeMode.value = "visual";
  formMode.value = "visual";
  designerDocument.value = defaultDesignerDocument();
  formFieldDrafts.value = [];
  formRef.value?.clearValidate();
  void nextTick(() => refreshDesignerCanvas());
}

function open() {
  visible.value = true;
  resetForm();
}

function handleClose() {
  visible.value = false;
  resetForm();
}

async function handleSubmit() {
  if (!formRef.value) return;

  // 可视化模式：先把画布导出为 node_config，再走统一表单校验。
  if (nodeMode.value === "visual") {
    try {
      formData.node_config = await serializeDesigner();
    } catch (err) {
      ElMessage.error((err as Error).message);
      return;
    }
  }
  if (formMode.value === "visual") formData.form_schema = fieldsToSchema(formFieldDrafts.value);

  const valid = await formRef.value.validate().then(
    () => true,
    () => false
  );
  if (!valid) return;

  try {
    loading.value = true;
    await createDefinition({
      data: {
        code: formData.code,
        version: formData.version,
        remark: formData.remark || undefined,
        nodeConfig: formData.node_config,
        formSchema: formData.form_schema || undefined,
      },
    });
    ElMessage.success($t("common.notification.createSuccess"));
    emit("success");
    handleClose();
  } catch {
    ElMessage.error($t("common.notification.createFailed"));
  } finally {
    loading.value = false;
  }
}

watch(visible, (val) => {
  if (!val) resetForm();
});

defineExpose({ open });
</script>

<style lang="scss" scoped>
.editor-switch {
  margin-bottom: 10px;
}

.hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.designer-host {
  width: 100%; // .el-form-item__content 是 flex 容器，不加会收缩到内容宽

  // go-wind-flow 组件的边框/滚动条 token 在宿主侧未定义时回退成白色，这里给默认值
  --gwf-border: var(--el-border-color);

  :deep(.gwf-designer) {
    background: transparent;
  }

  :deep(.gwf-panel) {
    flex: 0 0 320px;
    background: var(--el-bg-color);
    scrollbar-width: thin;
    scrollbar-color: var(--el-border-color) transparent;
  }

  // @bpmn-io/properties-panel 脱离其默认布局时会被压缩，强制撑满面板列
  :deep(.bio-properties-panel-container),
  :deep(.bio-properties-panel),
  :deep(.bio-properties-panel-scroll-container) {
    width: 100%;
    min-width: 0;
    background: var(--el-bg-color);
  }
}




</style>
