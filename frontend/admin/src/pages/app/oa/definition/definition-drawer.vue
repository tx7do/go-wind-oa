<template>
  <ProModal
    v-model:visible="visible"
    :title="title"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <ElForm ref="formRef" :model="formData" :rules="formRules" label-width="120px">
      <ElFormItem :label="$t('pages.oa.definition.fieldCode')" prop="code">
        <ElInput
          v-model="formData.code"
          :placeholder="$t('pages.oa.definition.phCode')"
          clearable
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

        <div v-if="nodeMode === 'visual'" class="node-editor">
          <div class="hint" style="margin-bottom: 8px">
            {{ $t("pages.oa.definition.graphHint") }}
          </div>

          <GraphCanvas :nodes="graphNodes" :edges="graphEdges" @node-move="onNodeMove" />

          <!-- 节点列表 -->
          <div class="graph-section">
            <div class="graph-section-title">{{ $t("pages.oa.definition.nodesTitle") }}</div>
            <div v-for="(node, ni) in graphNodes" :key="'gn' + ni" class="graph-node-card">
              <div class="node-head">
                <span class="node-title">
                  {{ $t("pages.oa.definition.nodeTitle", { index: ni + 1 }) }}
                </span>
                <div class="node-ops">
                  <ElButton
                    size="small"
                    text
                    type="danger"
                    @click="
                      graphNodes.splice(ni, 1);
                      syncEdges();
                    "
                  >
                    {{ $t("common.button.delete") }}
                  </ElButton>
                </div>
              </div>
              <ElFormItem :label="$t('pages.oa.definition.nodeIdLabel')" label-width="80px">
                <ElInput v-model="node.id" style="width: 200px" @change="syncEdges()" />
              </ElFormItem>
              <ElFormItem :label="$t('pages.oa.definition.typeLabel')" label-width="80px">
                <ElSelect v-model="node.type" style="width: 240px" @change="onNodeTypeChange(node)">
                  <ElOption
                    v-for="t in GRAPH_NODE_TYPE_OPTIONS"
                    :key="t.value"
                    :label="`${t.label} (${t.value})`"
                    :value="t.value"
                  />
                </ElSelect>
              </ElFormItem>

              <!-- TASK 节点：审批策略 + 审批人列表 -->
              <template v-if="node.type === 'TASK'">
                <ElFormItem :label="$t('pages.oa.definition.strategyLabel')" label-width="80px">
                  <ElRadioGroup v-model="node.strategy">
                    <ElRadio value="ALL">
                      {{ $t("pages.oa.definition.strategy.all") }}
                    </ElRadio>
                    <ElRadio value="ANY">
                      {{ $t("pages.oa.definition.strategy.any") }}
                    </ElRadio>
                  </ElRadioGroup>
                </ElFormItem>
                <div class="approver-list">
                  <div v-for="(ap, ai) in node.approvers" :key="ai" class="approver-row">
                    <ElSelect
                      v-model="ap.type"
                      style="width: 150px"
                      @change="onApproverTypeChange(ap)"
                    >
                      <ElOption :label="$t('pages.oa.definition.approverType.USER')" value="USER" />
                      <ElOption
                        :label="$t('pages.oa.definition.approverType.LEADER')"
                        value="LEADER"
                      />
                      <ElOption
                        :label="$t('pages.oa.definition.approverType.POSITION')"
                        value="POSITION"
                      />
                    </ElSelect>
                    <ElSelect
                      v-if="ap.type === 'USER' && users.length"
                      :model-value="ap.id"
                      filterable
                      :placeholder="$t('pages.oa.definition.phSelectUser')"
                      style="width: 220px"
                      @update:model-value="(v: any) => (ap.id = Number(v))"
                    >
                      <ElOption
                        v-for="u in users"
                        :key="u.id"
                        :label="`${userDisplayName(u)} (#${u.id})`"
                        :value="u.id!"
                      />
                    </ElSelect>
                    <ElSelect
                      v-else-if="ap.type === 'POSITION' && positions.length"
                      :model-value="ap.id"
                      filterable
                      :placeholder="$t('pages.oa.definition.phSelectPosition')"
                      style="width: 220px"
                      @update:model-value="(v: any) => (ap.id = Number(v))"
                    >
                      <ElOption
                        v-for="p in positions"
                        :key="p.id"
                        :label="`${positionDisplayName(p)} (#${p.id})`"
                        :value="p.id!"
                      />
                    </ElSelect>
                    <ElInputNumber
                      v-else-if="ap.type !== 'LEADER'"
                      v-model="ap.id"
                      :min="1"
                      :controls="false"
                      :placeholder="
                        ap.type === 'USER'
                          ? $t('pages.oa.definition.phUserId')
                          : $t('pages.oa.definition.phPositionId')
                      "
                      style="width: 160px"
                    />
                    <span v-else class="hint-inline">
                      {{ $t("pages.oa.definition.leaderHint") }}
                    </span>
                    <ElButton
                      size="small"
                      text
                      type="danger"
                      :disabled="(node.approvers?.length ?? 0) <= 1"
                      @click="node.approvers?.splice(ai, 1)"
                    >
                      {{ $t("pages.oa.definition.remove") }}
                    </ElButton>
                  </div>
                  <ElButton
                    size="small"
                    plain
                    @click="node.approvers?.push({ type: 'USER', id: undefined })"
                  >
                    {{ $t("pages.oa.definition.addApprover") }}
                  </ElButton>
                </div>
              </template>

              <!-- SUBPROCESS 节点：子流程定义引用 -->
              <template v-if="node.type === 'SUBPROCESS'">
                <ElFormItem :label="$t('pages.oa.definition.subprocessLabel')" label-width="80px">
                  <ElInput
                    v-model="node.subprocessDefinition"
                    :placeholder="$t('pages.oa.definition.phSubprocess')"
                    style="width: 300px"
                  />
                </ElFormItem>
              </template>
            </div>
            <div class="node-toolbar">
              <ElButton size="small" plain @click="addGraphNode('TASK')">
                {{ $t("pages.oa.definition.addTask") }}
              </ElButton>
              <ElButton size="small" plain @click="addGraphNode('EXCLUSIVE_GATEWAY')">
                {{ $t("pages.oa.definition.addGateway") }}
              </ElButton>
              <ElButton size="small" plain @click="addGraphNode('PARALLEL_GATEWAY_FORK')">
                {{ $t("pages.oa.definition.addFork") }}
              </ElButton>
              <ElButton size="small" plain @click="addGraphNode('PARALLEL_GATEWAY_JOIN')">
                {{ $t("pages.oa.definition.addJoin") }}
              </ElButton>
              <ElButton size="small" plain @click="addGraphNode('SUBPROCESS')">
                {{ $t("pages.oa.definition.addSubprocess") }}
              </ElButton>
              <ElButton size="small" plain @click="addGraphNode('START')">
                {{ $t("pages.oa.definition.addStart") }}
              </ElButton>
              <ElButton size="small" plain @click="addGraphNode('END')">
                {{ $t("pages.oa.definition.addEnd") }}
              </ElButton>
            </div>
          </div>

          <!-- 边列表 -->
          <div class="graph-section">
            <div class="graph-section-title">{{ $t("pages.oa.definition.edgesTitle") }}</div>
            <div v-for="(edge, ei) in graphEdges" :key="'ge' + ei" class="graph-edge-row">
              <ElSelect
                v-model="edge.from"
                :placeholder="$t('pages.oa.definition.edgeFrom')"
                style="width: 180px"
                @change="onEdgeFromChange(edge)"
              >
                <ElOption
                  v-for="n in graphNodes"
                  :key="n.id"
                  :label="n.id + ' (' + nodeTypeLabel(n.type) + ')'"
                  :value="n.id"
                />
              </ElSelect>
              <span class="edge-arrow">→</span>
              <ElSelect
                v-model="edge.to"
                :placeholder="$t('pages.oa.definition.edgeTo')"
                style="width: 180px"
              >
                <ElOption
                  v-for="n in graphNodes"
                  :key="n.id"
                  :label="n.id + ' (' + nodeTypeLabel(n.type) + ')'"
                  :value="n.id"
                />
              </ElSelect>
              <ElInput
                v-if="isExclusiveEdge(edge)"
                v-model="edge.condition"
                :placeholder="$t('pages.oa.definition.phCondition')"
                style="width: 220px"
              />
              <ElButton size="small" text type="danger" @click="graphEdges.splice(ei, 1)">
                {{ $t("common.button.delete") }}
              </ElButton>
            </div>
            <ElButton size="small" plain @click="graphEdges.push({ from: '', to: '' })">
              {{ $t("pages.oa.definition.addEdge") }}
            </ElButton>
          </div>

          <div v-if="graphNodes.length" class="json-preview">
            <div class="preview-title">{{ $t("pages.oa.definition.previewTitle") }}</div>
            <code>{{ graphConfigPreview }}</code>
          </div>
        </div>

        <ElInput
          v-else
          v-model="formData.node_config"
          type="textarea"
          placeholder='[{"approvers":[{"type":"USER","id":123},{"type":"LEADER"}],"strategy":"ALL"}]'
          :rows="8"
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
import {
  ElButton,
  ElInput,
  ElInputNumber,
  ElOption,
  ElRadioButton,
  ElRadioGroup,
  ElRadio,
  ElSelect,
} from "element-plus";
import { ref, reactive, computed, watch } from "vue";

import {
  useCreateWorkflowDefinition,
  fetchUsers,
  fetchPositions,
  userDisplayName,
  positionDisplayName,
} from "@/api/composables";
import type {
  identityservicev1_User,
  identityservicev1_Position,
} from "@/api/generated/admin/service/v1";
import { $t } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";
import ProModal from "@/components/Pro/ProModal/index.vue";

import GraphCanvas from "./graph-canvas.vue";
import FormSchemaEditor from "./form-schema-editor.vue";
import {
  applyApproverTypeChange,
  applyEdgeFromChange,
  applyNodeTypeChange,
  configToNodes,
  ensureLayout,
  fieldsToSchema,
  filterEdges,
  isExclusiveEdge as isExclusiveEdgeOf,
  makeGraphNode,
  nodesToConfig,
  schemaToFields,
  nodeTypeLabel,
  type ApproverDraft,
  type FieldDraft,
  type GraphEdgeDraft,
  type GraphNodeDraft,
} from "./workflow-graph";

const emit = defineEmits(["success"]);

const { mutateAsync: createDefinition } = useCreateWorkflowDefinition();

const visible = ref(false);
const loading = ref(false);
const formRef = ref<FormInstance>();

const nodeMode = ref<"visual" | "json">("visual");
const formMode = ref<"visual" | "json">("visual");
const graphNodes = ref<GraphNodeDraft[]>([]);
const graphEdges = ref<GraphEdgeDraft[]>([]);
const formFieldDrafts = ref<FieldDraft[]>([]);

// 审批人选择器数据源（用户/职位，列表加载失败时选择器回退手填 ID）。
const users = ref<identityservicev1_User[]>([]);
const positions = ref<identityservicev1_Position[]>([]);

// 表单数据（JSON 模式下的文本框 + 提交基础字段）
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
        let parsed: any;
        try {
          parsed = JSON.parse(value);
        } catch {
          callback(new Error($t("pages.oa.definition.vNodeConfigJson")));
          return;
        }

        // 图格式 {version:2, nodes:[], edges:[]}
        if (
          parsed &&
          typeof parsed === "object" &&
          !Array.isArray(parsed) &&
          parsed.version === 2
        ) {
          if (!Array.isArray(parsed.nodes) || parsed.nodes.length === 0) {
            callback(new Error($t("pages.oa.definition.vNodesEmpty")));
            return;
          }
          for (const node of parsed.nodes) {
            if (!node.id || !node.type) {
              callback(new Error($t("pages.oa.definition.vNodeIdType")));
              return;
            }
            if (node.type === "TASK") {
              const approvers = Array.isArray(node.approvers) ? node.approvers : [];
              if (approvers.length === 0) {
                callback(
                  new Error($t("pages.oa.definition.vTaskApproverRequired", { id: node.id }))
                );
                return;
              }
              for (const approver of approvers) {
                if (!["USER", "LEADER", "POSITION"].includes(approver.type)) {
                  callback(
                    new Error(
                      $t("pages.oa.definition.vApproverTypeInvalid", { type: approver.type })
                    )
                  );
                  return;
                }
                if (approver.type !== "LEADER" && !approver.id) {
                  callback(new Error($t("pages.oa.definition.vApproverIdRequired")));
                  return;
                }
              }
              const strategy = node.strategy ?? "ALL";
              if (strategy !== "ALL" && strategy !== "ANY") {
                callback(new Error($t("pages.oa.definition.vStrategyInvalid", { strategy })));
                return;
              }
            }
            if (
              node.type === "SUBPROCESS" &&
              (!node.subprocessDefinition || !String(node.subprocessDefinition).includes(":"))
            ) {
              callback(new Error($t("pages.oa.definition.vSubprocessRefInvalid", { id: node.id })));
              return;
            }
          }
          callback();
          return;
        }

        // 旧数组格式
        if (Array.isArray(parsed)) {
          if (parsed.length === 0) {
            callback(new Error($t("pages.oa.definition.vNodeConfigEmpty")));
            return;
          }
          for (const node of parsed) {
            const strategy = node.strategy ?? "ALL";
            if (strategy !== "ALL" && strategy !== "ANY") {
              callback(new Error($t("pages.oa.definition.vStrategyInvalid", { strategy })));
              return;
            }
            const approvers =
              Array.isArray(node.approvers) && node.approvers.length > 0
                ? node.approvers
                : node.approver_type === "USER" && node.approver
                  ? [{ type: "USER", id: node.approver }]
                  : [];
            if (approvers.length === 0) {
              callback(new Error($t("pages.oa.definition.vLegacyApproverRequired")));
              return;
            }
            for (const approver of approvers) {
              if (!["USER", "LEADER", "POSITION"].includes(approver.type)) {
                callback(
                  new Error(
                    $t("pages.oa.definition.vLegacyApproverTypeInvalid", { type: approver.type })
                  )
                );
                return;
              }
              if (approver.type !== "LEADER" && !approver.id) {
                callback(new Error($t("pages.oa.definition.vApproverIdRequired")));
                return;
              }
            }
          }
          callback();
          return;
        }

        callback(new Error($t("pages.oa.definition.vNodeConfigFormat")));
      },
      trigger: "blur",
    },
  ],
};

const title = computed(() =>
  $t("common.modal.create", { moduleName: $t("pages.oa.definition.title") })
);

// 节点类型下拉选项（label 走词条，value 为原始类型值；computed 以响应语言切换）
const GRAPH_NODE_TYPE_OPTIONS = computed(() =>
  (
    [
      "START",
      "END",
      "TASK",
      "EXCLUSIVE_GATEWAY",
      "PARALLEL_GATEWAY_FORK",
      "PARALLEL_GATEWAY_JOIN",
      "SUBPROCESS",
    ] as const
  ).map((value) => ({ value, label: nodeTypeLabel(value) }))
);

// 选择器数据源尽力加载：失败不阻塞编辑（回退手填 ID）。
async function loadSelectors() {
  try {
    const resp = await fetchUsers();
    users.value = resp.items ?? [];
  } catch {
    users.value = [];
  }
  try {
    const resp = await fetchPositions();
    positions.value = resp.items ?? [];
  } catch {
    positions.value = [];
  }
}
loadSelectors();

// ============ 图节点编辑 ============

function addGraphNode(type: GraphNodeDraft["type"]) {
  graphNodes.value.push(makeGraphNode(type));
  ensureLayout(graphNodes.value);
}

function onNodeTypeChange(node: GraphNodeDraft) {
  applyNodeTypeChange(node);
}

function onApproverTypeChange(ap: ApproverDraft) {
  applyApproverTypeChange(ap);
}

/** 节点 id 变更后，删除引用了旧 id 的边。 */
function syncEdges() {
  graphEdges.value = filterEdges(graphNodes.value, graphEdges.value);
}

function isExclusiveEdge(edge: GraphEdgeDraft): boolean {
  return isExclusiveEdgeOf(graphNodes.value, edge);
}

function onEdgeFromChange(edge: GraphEdgeDraft) {
  applyEdgeFromChange(graphNodes.value, edge);
}

function onNodeMove(ni: number, x: number, y: number) {
  const node = graphNodes.value[ni];
  if (node) {
    node.x = x;
    node.y = y;
  }
}

const graphConfigPreview = computed(() => nodesToConfig(graphNodes.value, graphEdges.value));

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
  if (mode === "visual") {
    const result = configToNodes(formData.node_config);
    if (result.nodes.length) {
      graphNodes.value = result.nodes;
      graphEdges.value = result.edges;
      ensureLayout(graphNodes.value);
    }
  } else {
    const cfg = nodesToConfig(graphNodes.value, graphEdges.value);
    if (cfg) formData.node_config = cfg;
  }
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
  graphNodes.value = [];
  graphEdges.value = [];
  formFieldDrafts.value = [];
  formRef.value?.clearValidate();
}

function open() {
  visible.value = true;
  resetForm();
}

function handleClose() {
  visible.value = false;
  resetForm();
}

/** 可视化模式校验：图结构基础校验 + TASK 节点审批人校验 + 表单字段校验。 */
function validateDrafts(): string | null {
  // 图节点校验
  const validNodes = graphNodes.value.filter((n) => n.id);
  if (!validNodes.length) return $t("pages.oa.definition.vAtLeastOneNode");

  const idSet = new Set<string>();
  for (const node of validNodes) {
    if (idSet.has(node.id)) return $t("pages.oa.definition.vNodeIdDup", { id: node.id });
    idSet.add(node.id);
  }

  for (let i = 0; i < validNodes.length; i++) {
    const node = validNodes[i];
    if (node.type === "TASK") {
      if (!node.approvers || node.approvers.length === 0)
        return $t("pages.oa.definition.vApproverRequired", { id: node.id });
      for (const ap of node.approvers) {
        if (ap.type !== "LEADER" && (!ap.id || ap.id <= 0)) {
          return $t(
            ap.type === "USER"
              ? "pages.oa.definition.vUserRequired"
              : "pages.oa.definition.vPositionRequired",
            { id: node.id }
          );
        }
      }
    }
    if (node.type === "SUBPROCESS") {
      if (!node.subprocessDefinition || !node.subprocessDefinition.includes(":")) {
        return $t("pages.oa.definition.vSubprocessRequired", { id: node.id });
      }
    }
  }

  // 表单字段校验
  const keys = new Set<string>();
  for (const f of formFieldDrafts.value) {
    if (!f.key.trim()) continue;
    if (keys.has(f.key.trim()))
      return $t("pages.oa.definition.vFieldKeyDup", { key: f.key.trim() });
    keys.add(f.key.trim());
    if (f.type === "select") {
      const opts = f.options
        .split(/[,，]/)
        .map((o) => o.trim())
        .filter(Boolean);
      if (!opts.length) return $t("pages.oa.definition.vFieldOptionsRequired", { key: f.key });
    }
  }
  return null;
}

async function handleSubmit() {
  if (!formRef.value) return;

  const valid = await formRef.value.validate().then(
    () => true,
    () => false
  );
  if (!valid) return;

  // 可视化模式：生成 JSON 并做结构校验。
  if (nodeMode.value === "visual" || formMode.value === "visual") {
    const err = validateDrafts();
    if (err) {
      ElMessage.warning(err);
      return;
    }
    if (nodeMode.value === "visual")
      formData.node_config = nodesToConfig(graphNodes.value, graphEdges.value);
    if (formMode.value === "visual") formData.form_schema = fieldsToSchema(formFieldDrafts.value);
  }

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
.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.editor-switch {
  margin-bottom: 10px;
}

.node-editor {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* ============ 图编辑器 ============ */
.graph-section {
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  padding: 12px;
  margin-bottom: 12px;
}

.graph-section-title {
  font-weight: 600;
  font-size: 13px;
  margin-bottom: 8px;
  color: var(--el-text-color-primary);
}

.graph-node-card {
  border: 1px solid var(--el-border-color-light);
  border-radius: 4px;
  padding: 8px;
  margin-bottom: 8px;
  background: var(--el-bg-color);
}

.graph-edge-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.edge-arrow {
  color: var(--el-text-color-placeholder);
  font-size: 14px;
}

.node-toolbar {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.node-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.node-title {
  font-weight: 600;
  font-size: 13px;
}

.node-ops {
  display: flex;
  gap: 4px;
}

.approver-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-items: flex-start;
}

.approver-row {
  display: flex;
  align-items: center;
  gap: 8px;
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
