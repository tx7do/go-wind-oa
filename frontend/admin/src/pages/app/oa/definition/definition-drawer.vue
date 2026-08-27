<template>
  <ProModal
    v-model:visible="visible"
    :title="title"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <ElForm ref="formRef" :model="formData" :rules="formRules" label-width="120px">
      <ElFormItem :label="$t('pages.oa.definition.fieldCode')" prop="code">
        <ElInput v-model="formData.code" placeholder="如 LEAVE / EXPENSE / TRIP" clearable />
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

      <ElFormItem label="备注">
        <ElInput
          v-model="formData.remark"
          placeholder="流程用途说明（可选）"
          :rows="2"
          type="textarea"
        />
      </ElFormItem>

      <!-- ============ 审批流（node_config） ============ -->
      <ElFormItem label="审批流">
        <div class="editor-switch">
          <ElRadioGroup v-model="nodeMode" size="small" @change="onNodeModeChange">
            <ElRadioButton value="visual">可视化</ElRadioButton>
            <ElRadioButton value="json">JSON</ElRadioButton>
          </ElRadioGroup>
        </div>

        <div v-if="nodeMode === 'visual'" class="node-editor">
          <div class="hint" style="margin-bottom: 8px">
            图编辑器：上方画布可拖拽节点调整布局位置，下方列表配置详细属性。TASK
            节点配置审批人；EXCLUSIVE_GATEWAY 的出边需配置条件表达式（default 为兜底）。
          </div>

          <!-- SVG 画布预览：节点拖拽定位 + 边可视化 -->
          <div class="graph-canvas-container">
            <svg
              ref="svgCanvas"
              class="graph-canvas"
              width="100%"
              height="300"
              @mousedown="onCanvasMouseDown"
            >
              <!-- 边 -->
              <g v-for="(edge, ei) in renderedEdges" :key="'se' + ei">
                <line
                  :x1="edge.x1"
                  :y1="edge.y1"
                  :x2="edge.x2"
                  :y2="edge.y2"
                  :class="['graph-edge-line', edge.hasCondition ? 'conditional' : '']"
                  @click="selectEdge(edge.index)"
                />
                <text
                  v-if="edge.hasCondition"
                  :x="(edge.x1 + edge.x2) / 2"
                  :y="(edge.y1 + edge.y2) / 2 - 5"
                  class="graph-edge-label"
                  @click="selectEdge(edge.index)"
                >
                  {{ edge.condition }}
                </text>
              </g>
              <!-- 节点 -->
              <g
                v-for="(node, ni) in graphNodes"
                :key="'sn' + ni"
                :transform="`translate(${node.x ?? 50}, ${node.y ?? 50})`"
                @mousedown.stop="onNodeDragStart($event, ni)"
                @click="selectNode(ni)"
              >
                <rect
                  :width="nodeVisualWidth(node)"
                  :height="nodeVisualHeight(node)"
                  :rx="node.type === 'START' || node.type === 'END' ? 25 : 6"
                  :class="[
                    'graph-node-rect',
                    `node-type-${node.type.toLowerCase()}`,
                    selectedNodeIndex === ni ? 'selected' : '',
                  ]"
                />
                <text
                  :x="nodeVisualWidth(node) / 2"
                  :y="nodeVisualHeight(node) / 2"
                  class="graph-node-label"
                >
                  {{ nodeVisualLabel(node) }}
                </text>
              </g>
            </svg>
          </div>

          <!-- 节点列表 -->
          <div class="graph-section">
            <div class="graph-section-title">节点</div>
            <div v-for="(node, ni) in graphNodes" :key="'gn' + ni" class="graph-node-card">
              <div class="node-head">
                <span class="node-title">节点 {{ ni + 1 }}</span>
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
                    删除
                  </ElButton>
                </div>
              </div>
              <ElFormItem label="节点ID" label-width="80px">
                <ElInput v-model="node.id" style="width: 200px" @change="syncEdges()" />
              </ElFormItem>
              <ElFormItem label="类型" label-width="80px">
                <ElSelect v-model="node.type" style="width: 240px" @change="onNodeTypeChange(node)">
                  <ElOption label="开始 (START)" value="START" />
                  <ElOption label="结束 (END)" value="END" />
                  <ElOption label="审批 (TASK)" value="TASK" />
                  <ElOption label="条件分支 (EXCLUSIVE_GATEWAY)" value="EXCLUSIVE_GATEWAY" />
                  <ElOption label="并行分裂 (FORK)" value="PARALLEL_GATEWAY_FORK" />
                  <ElOption label="并行汇聚 (JOIN)" value="PARALLEL_GATEWAY_JOIN" />
                  <ElOption label="子流程 (SUBPROCESS)" value="SUBPROCESS" />
                </ElSelect>
              </ElFormItem>

              <!-- TASK 节点：审批策略 + 审批人列表 -->
              <template v-if="node.type === 'TASK'">
                <ElFormItem label="审批策略" label-width="80px">
                  <ElRadioGroup v-model="node.strategy">
                    <ElRadio value="ALL">会签（全员通过）</ElRadio>
                    <ElRadio value="ANY">或签（一人通过）</ElRadio>
                  </ElRadioGroup>
                </ElFormItem>
                <div class="approver-list">
                  <div v-for="(ap, ai) in node.approvers" :key="ai" class="approver-row">
                    <ElSelect
                      v-model="ap.type"
                      style="width: 150px"
                      @change="onApproverTypeChange(ap)"
                    >
                      <ElOption label="指定用户" value="USER" />
                      <ElOption label="申请人主管" value="LEADER" />
                      <ElOption label="职位持有者" value="POSITION" />
                    </ElSelect>
                    <ElSelect
                      v-if="ap.type === 'USER' && users.length"
                      :model-value="ap.id"
                      filterable
                      placeholder="搜索并选择用户"
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
                      placeholder="搜索并选择职位"
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
                      :placeholder="ap.type === 'USER' ? '用户ID' : '职位ID'"
                      style="width: 160px"
                    />
                    <span v-else class="hint-inline">按提交人自动解析</span>
                    <ElButton
                      size="small"
                      text
                      type="danger"
                      :disabled="(node.approvers?.length ?? 0) <= 1"
                      @click="node.approvers?.splice(ai, 1)"
                    >
                      移除
                    </ElButton>
                  </div>
                  <ElButton
                    size="small"
                    plain
                    @click="node.approvers?.push({ type: 'USER', id: undefined })"
                  >
                    + 添加审批人
                  </ElButton>
                </div>
              </template>

              <!-- SUBPROCESS 节点：子流程定义引用 -->
              <template v-if="node.type === 'SUBPROCESS'">
                <ElFormItem label="子流程定义" label-width="80px">
                  <ElInput
                    v-model="node.subprocessDefinition"
                    placeholder="子流程 code:version（如 LEAVE:1）"
                    style="width: 300px"
                  />
                </ElFormItem>
              </template>
            </div>
            <div class="node-toolbar">
              <ElButton size="small" plain @click="addGraphNode('TASK')">+ 审批节点</ElButton>
              <ElButton size="small" plain @click="addGraphNode('EXCLUSIVE_GATEWAY')">
                + 条件分支
              </ElButton>
              <ElButton size="small" plain @click="addGraphNode('PARALLEL_GATEWAY_FORK')">
                + 并行分裂
              </ElButton>
              <ElButton size="small" plain @click="addGraphNode('PARALLEL_GATEWAY_JOIN')">
                + 并行汇聚
              </ElButton>
              <ElButton size="small" plain @click="addGraphNode('SUBPROCESS')">+ 子流程</ElButton>
              <ElButton size="small" plain @click="addGraphNode('START')">+ 开始</ElButton>
              <ElButton size="small" plain @click="addGraphNode('END')">+ 结束</ElButton>
            </div>
          </div>

          <!-- 边列表 -->
          <div class="graph-section">
            <div class="graph-section-title">连线</div>
            <div v-for="(edge, ei) in graphEdges" :key="'ge' + ei" class="graph-edge-row">
              <ElSelect
                v-model="edge.from"
                placeholder="起点"
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
              <ElSelect v-model="edge.to" placeholder="终点" style="width: 180px">
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
                placeholder='条件表达式或 "default"'
                style="width: 220px"
              />
              <ElButton size="small" text type="danger" @click="graphEdges.splice(ei, 1)">
                删除
              </ElButton>
            </div>
            <ElButton size="small" plain @click="graphEdges.push({ from: '', to: '' })">
              + 添加连线
            </ElButton>
          </div>

          <div v-if="graphNodes.length" class="json-preview">
            <div class="preview-title">生成配置预览</div>
            <code>{{ nodesToConfig() }}</code>
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
      <ElFormItem label="申请表单">
        <div class="editor-switch">
          <ElRadioGroup v-model="formMode" size="small" @change="onFormModeChange">
            <ElRadioButton value="visual">可视化</ElRadioButton>
            <ElRadioButton value="json">JSON</ElRadioButton>
          </ElRadioGroup>
        </div>

        <div v-if="formMode === 'visual'" class="form-editor">
          <ElTable v-if="formFieldDrafts.length" :data="formFieldDrafts" border size="small">
            <ElTableColumn label="字段名 key" width="130">
              <template #default="{ row }">
                <ElInput v-model="row.key" placeholder="如 reason" size="small" />
              </template>
            </ElTableColumn>
            <ElTableColumn label="显示名 label" width="130">
              <template #default="{ row }">
                <ElInput v-model="row.label" placeholder="如 事由" size="small" />
              </template>
            </ElTableColumn>
            <ElTableColumn label="类型" width="120">
              <template #default="{ row }">
                <ElSelect v-model="row.type" size="small">
                  <ElOption label="单行文本" value="text" />
                  <ElOption label="多行文本" value="textarea" />
                  <ElOption label="数字" value="number" />
                  <ElOption label="日期" value="date" />
                  <ElOption label="下拉选择" value="select" />
                </ElSelect>
              </template>
            </ElTableColumn>
            <ElTableColumn label="必填" width="60" align="center">
              <template #default="{ row }">
                <ElCheckbox v-model="row.required" size="small" />
              </template>
            </ElTableColumn>
            <ElTableColumn label="选项（select 用，逗号分隔）" min-width="160">
              <template #default="{ row }">
                <ElInput
                  v-if="row.type === 'select'"
                  v-model="row.options"
                  placeholder="普通,紧急"
                  size="small"
                />
                <span v-else class="hint-inline">-</span>
              </template>
            </ElTableColumn>
            <ElTableColumn label="操作" width="60" align="center">
              <template #default="{ $index }">
                <ElButton
                  size="small"
                  text
                  type="danger"
                  @click="formFieldDrafts.splice($index, 1)"
                >
                  删
                </ElButton>
              </template>
            </ElTableColumn>
          </ElTable>
          <ElButton
            size="small"
            plain
            class="add-field-btn"
            @click="
              formFieldDrafts.push({
                key: '',
                label: '',
                type: 'text',
                required: false,
                options: '',
              })
            "
          >
            + 添加字段
          </ElButton>
          <div class="hint">不配置表单时，移动端提交该流程回退为自由 JSON 输入。</div>
          <div v-if="formFieldDrafts.some((f) => f.key.trim())" class="json-preview">
            <div class="preview-title">生成配置预览</div>
            <code>{{ fieldsToSchema() }}</code>
          </div>
        </div>

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
  ElCheckbox,
  ElInput,
  ElInputNumber,
  ElOption,
  ElRadioButton,
  ElRadioGroup,
  ElRadio,
  ElSelect,
  ElTable,
  ElTableColumn,
} from "element-plus";
import { ref, reactive, computed, watch, onMounted, onBeforeUnmount } from "vue";

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

const emit = defineEmits(["success"]);

const { mutateAsync: createDefinition } = useCreateWorkflowDefinition();

const visible = ref(false);
const loading = ref(false);
const formRef = ref<FormInstance>();

/** 审批人草稿。LEADER 无需 id（按提交人动态解析）。 */
interface ApproverDraft {
  type: "USER" | "LEADER" | "POSITION";
  id?: number;
}

/** 图节点草稿。TASK 节点额外携带审批人配置。 */
interface GraphNodeDraft {
  id: string;
  type:
    | "START"
    | "END"
    | "TASK"
    | "EXCLUSIVE_GATEWAY"
    | "PARALLEL_GATEWAY_FORK"
    | "PARALLEL_GATEWAY_JOIN"
    | "SUBPROCESS";
  strategy?: "ALL" | "ANY";
  approvers?: ApproverDraft[];
  subprocessDefinition?: string;
  x?: number;
  y?: number;
}

/** 图边草稿。EXCLUSIVE_GATEWAY 出边的 condition 为表达式或 "default"。 */
interface GraphEdgeDraft {
  from: string;
  to: string;
  condition?: string;
}

/** 表单字段草稿（options 为逗号分隔字符串，仅 select 用）。 */
interface FieldDraft {
  key: string;
  label: string;
  type: "text" | "textarea" | "number" | "date" | "select";
  required: boolean;
  options: string;
}

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
          callback(new Error("form_schema 必须是合法的 JSON"));
          return;
        }
        for (const f of fields) {
          if (!f.key || !f.label) {
            callback(new Error("每个字段需要 key 与 label"));
            return;
          }
          if (!["text", "textarea", "number", "date", "select"].includes(f.type || "text")) {
            callback(new Error(`字段 ${f.key} 的 type 仅支持 text/textarea/number/date/select`));
            return;
          }
          if (f.type === "select" && (!Array.isArray(f.options) || f.options.length === 0)) {
            callback(new Error(`字段 ${f.key} 为 select 类型，必须提供非空 options 数组`));
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
          callback(new Error("node_config 必须是合法的 JSON"));
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
            callback(new Error("图格式 nodes 必须是非空数组"));
            return;
          }
          for (const node of parsed.nodes) {
            if (!node.id || !node.type) {
              callback(new Error("每个节点必须有 id 和 type"));
              return;
            }
            if (node.type === "TASK") {
              const approvers = Array.isArray(node.approvers) ? node.approvers : [];
              if (approvers.length === 0) {
                callback(new Error(`TASK 节点 ${node.id} 至少需要一个审批人`));
                return;
              }
              for (const approver of approvers) {
                if (!["USER", "LEADER", "POSITION"].includes(approver.type)) {
                  callback(new Error(`非法审批人类型 ${approver.type}`));
                  return;
                }
                if (approver.type !== "LEADER" && !approver.id) {
                  callback(new Error(`USER / POSITION 类型审批人必须提供 id`));
                  return;
                }
              }
              const strategy = node.strategy ?? "ALL";
              if (strategy !== "ALL" && strategy !== "ANY") {
                callback(new Error(`非法审批策略 ${strategy}（仅 ALL 会签 / ANY 或签）`));
                return;
              }
            }
            if (
              node.type === "SUBPROCESS" &&
              (!node.subprocessDefinition || !String(node.subprocessDefinition).includes(":"))
            ) {
              callback(new Error(`SUBPROCESS 节点 ${node.id} 必须填写子流程引用（code:version）`));
              return;
            }
          }
          callback();
          return;
        }

        // 旧数组格式
        if (Array.isArray(parsed)) {
          if (parsed.length === 0) {
            callback(new Error("node_config 必须是非空节点数组"));
            return;
          }
          for (const node of parsed) {
            const strategy = node.strategy ?? "ALL";
            if (strategy !== "ALL" && strategy !== "ANY") {
              callback(new Error(`非法审批策略 ${strategy}（仅 ALL 会签 / ANY 或签）`));
              return;
            }
            const approvers =
              Array.isArray(node.approvers) && node.approvers.length > 0
                ? node.approvers
                : node.approver_type === "USER" && node.approver
                  ? [{ type: "USER", id: node.approver }]
                  : [];
            if (approvers.length === 0) {
              callback(
                new Error("每个节点至少需要一个审批人（approvers 或旧格式 approver_type+approver）")
              );
              return;
            }
            for (const approver of approvers) {
              if (!["USER", "LEADER", "POSITION"].includes(approver.type)) {
                callback(
                  new Error(`非法审批人类型 ${approver.type}（仅 USER / LEADER / POSITION）`)
                );
                return;
              }
              if (approver.type !== "LEADER" && !approver.id) {
                callback(new Error("USER / POSITION 类型审批人必须提供 id"));
                return;
              }
            }
          }
          callback();
          return;
        }

        callback(new Error("node_config 必须是图格式对象或节点数组"));
      },
      trigger: "blur",
    },
  ],
};

const title = computed(() =>
  $t("common.modal.create", { moduleName: $t("pages.oa.definition.title") })
);

onMounted(async () => {
  // 选择器数据源尽力加载：失败不阻塞编辑（回退手填 ID）。
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
});

// ============ 图节点编辑 ============

const NODE_TYPE_LABELS: Record<string, string> = {
  START: "开始",
  END: "结束",
  TASK: "审批",
  EXCLUSIVE_GATEWAY: "条件分支",
  PARALLEL_GATEWAY_FORK: "并行分裂",
  PARALLEL_GATEWAY_JOIN: "并行汇聚",
  SUBPROCESS: "子流程",
};

function nodeTypeLabel(type: string): string {
  return NODE_TYPE_LABELS[type] ?? type;
}

let nodeCounter = 0;

function addGraphNode(type: GraphNodeDraft["type"]) {
  const prefix =
    type === "START"
      ? "start"
      : type === "END"
        ? "end"
        : type === "EXCLUSIVE_GATEWAY"
          ? "gw"
          : type === "PARALLEL_GATEWAY_FORK"
            ? "fork"
            : type === "PARALLEL_GATEWAY_JOIN"
              ? "join"
              : type === "SUBPROCESS"
                ? "sub"
                : "node";
  const id = `${prefix}_${nodeCounter++}`;
  const node: GraphNodeDraft = { id, type };
  if (type === "TASK") {
    node.strategy = "ALL";
    node.approvers = [{ type: "LEADER" }];
  }
  if (type === "SUBPROCESS") {
    node.subprocessDefinition = "";
  }
  graphNodes.value.push(node);
  ensureLayout();
}

function onNodeTypeChange(node: GraphNodeDraft) {
  // 切换类型时调整字段
  if (node.type === "TASK") {
    if (!node.approvers) node.approvers = [{ type: "LEADER" }];
    if (!node.strategy) node.strategy = "ALL";
    node.subprocessDefinition = undefined;
  } else if (node.type === "SUBPROCESS") {
    node.approvers = undefined;
    node.strategy = undefined;
    if (!node.subprocessDefinition) node.subprocessDefinition = "";
  } else {
    node.approvers = undefined;
    node.strategy = undefined;
    node.subprocessDefinition = undefined;
  }
}

function onApproverTypeChange(ap: ApproverDraft) {
  if (ap.type === "LEADER") ap.id = undefined;
  else if (!ap.id) ap.id = 1;
}

/** 节点 id 变更后，删除引用了旧 id 的边。 */
function syncEdges() {
  const validIds = new Set(graphNodes.value.map((n) => n.id));
  graphEdges.value = graphEdges.value.filter((e) => validIds.has(e.from) && validIds.has(e.to));
}

function isExclusiveEdge(edge: GraphEdgeDraft): boolean {
  const fromNode = graphNodes.value.find((n) => n.id === edge.from);
  return fromNode?.type === "EXCLUSIVE_GATEWAY";
}

function onEdgeFromChange(edge: GraphEdgeDraft) {
  // 从 EXCLUSIVE_GATEWAY 出发的边默认 condition 为 default
  if (isExclusiveEdge(edge) && !edge.condition) {
    edge.condition = "default";
  } else if (!isExclusiveEdge(edge)) {
    edge.condition = undefined;
  }
}

// ============ SVG 画布渲染与拖拽 ============

const selectedNodeIndex = ref<number | null>(null);
const selectedEdgeIndex = ref<number | null>(null);

const NODE_VISUAL_SIZE: Record<string, { w: number; h: number }> = {
  START: { w: 50, h: 50 },
  END: { w: 50, h: 50 },
  TASK: { w: 90, h: 40 },
  EXCLUSIVE_GATEWAY: { w: 60, h: 60 },
  PARALLEL_GATEWAY_FORK: { w: 60, h: 60 },
  PARALLEL_GATEWAY_JOIN: { w: 60, h: 60 },
  SUBPROCESS: { w: 90, h: 40 },
};

function nodeVisualWidth(node: GraphNodeDraft): number {
  return NODE_VISUAL_SIZE[node.type]?.w ?? 60;
}

function nodeVisualHeight(node: GraphNodeDraft): number {
  return NODE_VISUAL_SIZE[node.type]?.h ?? 60;
}

function nodeVisualLabel(node: GraphNodeDraft): string {
  return NODE_TYPE_LABELS[node.type] ?? node.type;
}

/** 为缺少坐标的节点分配默认位置（环形布局）。已有坐标的节点不动。 */
function ensureLayout() {
  const nodes = graphNodes.value;
  const n = nodes.length;
  if (n === 0) return;
  const cx = 400;
  const cy = 160;
  const radius = Math.min(280, 50 + n * 35);
  let unassigned = 0;
  for (let i = 0; i < n; i++) {
    if (nodes[i].x === undefined || nodes[i].y === undefined) {
      unassigned++;
    }
  }
  if (unassigned === 0) return;
  let placed = 0;
  for (let i = 0; i < n; i++) {
    if (nodes[i].x === undefined || nodes[i].y === undefined) {
      const angle = (2 * Math.PI * placed) / unassigned;
      nodes[i].x = Math.round(cx + radius * Math.cos(angle));
      nodes[i].y = Math.round(cy + radius * Math.sin(angle));
      placed++;
    }
  }
}

const renderedEdges = computed(() => {
  const result: {
    x1: number;
    y1: number;
    x2: number;
    y2: number;
    hasCondition: boolean;
    condition: string;
    index: number;
  }[] = [];
  for (let i = 0; i < graphEdges.value.length; i++) {
    const edge = graphEdges.value[i];
    const fromNode = graphNodes.value.find((n) => n.id === edge.from);
    const toNode = graphNodes.value.find((n) => n.id === edge.to);
    if (!fromNode || !toNode) continue;
    result.push({
      x1: fromNode.x ?? 50,
      y1: fromNode.y ?? 50,
      x2: toNode.x ?? 50,
      y2: toNode.y ?? 50,
      hasCondition: typeof edge.condition === "string" && edge.condition.length > 0,
      condition: typeof edge.condition === "string" ? edge.condition : "",
      index: i,
    });
  }
  return result;
});

function selectNode(ni: number) {
  selectedNodeIndex.value = ni;
  selectedEdgeIndex.value = null;
}

function selectEdge(ei: number) {
  selectedEdgeIndex.value = ei;
  selectedNodeIndex.value = null;
}

function onCanvasMouseDown() {
  selectedNodeIndex.value = null;
  selectedEdgeIndex.value = null;
}

let dragInfo: {
  idx: number;
  startMX: number;
  startMY: number;
  startNX: number;
  startNY: number;
} | null = null;

function onNodeDragStart(event: MouseEvent, ni: number) {
  event.preventDefault();
  const node = graphNodes.value[ni];
  if (!node) return;
  selectedNodeIndex.value = ni;
  dragInfo = {
    idx: ni,
    startMX: event.clientX,
    startMY: event.clientY,
    startNX: node.x ?? 0,
    startNY: node.y ?? 0,
  };
  window.addEventListener("mousemove", onNodeDragMove);
  window.addEventListener("mouseup", onNodeDragEnd);
}

function onNodeDragMove(event: MouseEvent) {
  if (!dragInfo) return;
  const node = graphNodes.value[dragInfo.idx];
  if (!node) return;
  const nx = dragInfo.startNX + (event.clientX - dragInfo.startMX);
  const ny = dragInfo.startNY + (event.clientY - dragInfo.startMY);
  node.x = Math.max(0, Math.min(750, nx));
  node.y = Math.max(0, Math.min(270, ny));
}

function onNodeDragEnd() {
  dragInfo = null;
  window.removeEventListener("mousemove", onNodeDragMove);
  window.removeEventListener("mouseup", onNodeDragEnd);
}

// ============ 序列化 ============

/** 图节点+边草稿 → 流程图 JSON（version:2 图格式）。 */
function nodesToConfig(): string {
  const validNodes = graphNodes.value.filter((n) => n.id);
  if (!validNodes.length) return "";

  const nodes = validNodes.map((n) => {
    const node: Record<string, unknown> = { id: n.id, type: n.type };
    if (n.type === "TASK" && n.approvers && n.approvers.length > 0) {
      node.approvers = n.approvers.map((a) =>
        a.type === "LEADER" ? { type: a.type } : { type: a.type, id: a.id ?? 0 }
      );
      node.strategy = n.strategy;
    }
    if (n.type === "SUBPROCESS" && n.subprocessDefinition) {
      node.subprocessDefinition = n.subprocessDefinition;
    }
    return node;
  });

  const validIds = new Set(validNodes.map((n) => n.id));
  const edges = graphEdges.value
    .filter((e) => validIds.has(e.from) && validIds.has(e.to))
    .map((e) => {
      const edge: Record<string, unknown> = { from: e.from, to: e.to };
      if (e.condition) edge.condition = e.condition;
      return edge;
    });

  return JSON.stringify({ version: 2, nodes, edges });
}

/** 流程图 JSON → 图节点+边草稿。
 *  支持图格式（version:2）和旧数组格式（自动转线性图）。 */
function configToNodes(json: string) {
  const nodes: GraphNodeDraft[] = [];
  const edges: GraphEdgeDraft[] = [];
  try {
    const parsed = JSON.parse(json);

    // 图格式 version:2
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed) && parsed.version === 2) {
      if (Array.isArray(parsed.nodes)) {
        for (const n of parsed.nodes) {
          const type = n.type as GraphNodeDraft["type"];
          if (!type || !NODE_TYPE_LABELS[type]) continue;
          const node: GraphNodeDraft = { id: String(n.id), type };
          if (type === "TASK" && Array.isArray(n.approvers) && n.approvers.length > 0) {
            node.approvers = n.approvers
              .filter((a: any) => ["USER", "LEADER", "POSITION"].includes(a.type))
              .map((a: any) => ({ type: a.type as ApproverDraft["type"], id: a.id }));
            node.strategy = n.strategy === "ANY" ? "ANY" : "ALL";
          }
          if (type === "SUBPROCESS" && typeof n.subprocessDefinition === "string") {
            node.subprocessDefinition = n.subprocessDefinition;
          }
          nodes.push(node);
        }
      }
      if (Array.isArray(parsed.edges)) {
        for (const e of parsed.edges) {
          if (!e.from || !e.to) continue;
          const edge: GraphEdgeDraft = { from: String(e.from), to: String(e.to) };
          if (typeof e.condition === "string") edge.condition = e.condition;
          edges.push(edge);
        }
      }
      return { nodes, edges };
    }

    // 旧数组格式 → 自动转线性图
    if (!Array.isArray(parsed)) return { nodes, edges };
    const taskNodes: GraphNodeDraft[] = [];
    for (const node of parsed) {
      const approvers: ApproverDraft[] =
        Array.isArray(node.approvers) && node.approvers.length > 0
          ? node.approvers
              .filter((a: any) => ["USER", "LEADER", "POSITION"].includes(a.type))
              .map((a: any) => ({ type: a.type, id: a.id }))
          : node.approver_type === "USER" && node.approver
            ? [{ type: "USER" as const, id: node.approver }]
            : [];
      if (approvers.length === 0) continue;
      taskNodes.push({
        id: `node_${taskNodes.length}`,
        type: "TASK" as const,
        strategy: node.strategy === "ANY" ? "ANY" : "ALL",
        approvers,
      });
    }
    if (!taskNodes.length) return { nodes, edges };

    // start → task0 → … → taskN → end
    const startNode: GraphNodeDraft = { id: "start", type: "START" };
    const endNode: GraphNodeDraft = { id: "end", type: "END" };
    nodes.push(startNode, ...taskNodes, endNode);
    edges.push({ from: "start", to: taskNodes[0].id });
    for (let i = 0; i < taskNodes.length - 1; i++) {
      edges.push({ from: taskNodes[i].id, to: taskNodes[i + 1].id });
    }
    edges.push({ from: taskNodes[taskNodes.length - 1].id, to: "end" });
  } catch {
    /* 非法 JSON：返回已解析部分 */
  }
  return { nodes, edges };
}

/** 字段草稿 → form_schema JSON。key 为空的行跳过；select 解析逗号选项。 */
function fieldsToSchema(): string {
  const fields = formFieldDrafts.value
    .filter((f) => f.key.trim())
    .map((f) => {
      const item: Record<string, unknown> = {
        key: f.key.trim(),
        label: f.label.trim() || f.key.trim(),
        type: f.type,
        required: f.required,
      };
      if (f.type === "select") {
        item.options = f.options
          .split(/[,，]/)
          .map((o) => o.trim())
          .filter(Boolean);
      }
      return item;
    });
  return fields.length ? JSON.stringify(fields) : "";
}

/** JSON 文本 → 字段草稿。 */
function schemaToFields(json: string) {
  const drafts: FieldDraft[] = [];
  try {
    const parsed = JSON.parse(json);
    const list = Array.isArray(parsed) ? parsed : [parsed];
    for (const f of list) {
      if (!f?.key) continue;
      drafts.push({
        key: String(f.key),
        label: String(f.label ?? f.key),
        type: (["text", "textarea", "number", "date", "select"].includes(f.type)
          ? f.type
          : "text") as FieldDraft["type"],
        required: f.required === true,
        options: Array.isArray(f.options) ? f.options.join(",") : "",
      });
    }
  } catch {
    /* 非法 JSON：返回已解析部分 */
  }
  return drafts;
}

// ============ 模式切换 ============

function onNodeModeChange(mode: string | number | boolean | undefined) {
  if (mode === "visual") {
    const result = configToNodes(formData.node_config);
    if (result.nodes.length) {
      graphNodes.value = result.nodes;
      graphEdges.value = result.edges;
      ensureLayout();
    }
  } else {
    const cfg = nodesToConfig();
    if (cfg) formData.node_config = cfg;
  }
}

function onFormModeChange(mode: string | number | boolean | undefined) {
  if (mode === "visual") {
    const drafts = schemaToFields(formData.form_schema);
    if (drafts.length) formFieldDrafts.value = drafts;
  } else {
    formData.form_schema = fieldsToSchema();
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

function open(_row?: any) {
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
  if (!validNodes.length) return "至少需要一个节点";

  const idSet = new Set<string>();
  for (const node of validNodes) {
    if (idSet.has(node.id)) return `节点ID重复：${node.id}`;
    idSet.add(node.id);
  }

  for (let i = 0; i < validNodes.length; i++) {
    const node = validNodes[i];
    if (node.type === "TASK") {
      if (!node.approvers || node.approvers.length === 0)
        return `节点 ${node.id} 至少需要一个审批人`;
      for (const ap of node.approvers) {
        if (ap.type !== "LEADER" && (!ap.id || ap.id <= 0)) {
          return `节点 ${node.id} 的${ap.type === "USER" ? "用户" : "职位"}必须选择`;
        }
      }
    }
    if (node.type === "SUBPROCESS") {
      if (!node.subprocessDefinition || !node.subprocessDefinition.includes(":")) {
        return `节点 ${node.id} 必须填写子流程引用（code:version 格式）`;
      }
    }
  }

  // 表单字段校验
  const keys = new Set<string>();
  for (const f of formFieldDrafts.value) {
    if (!f.key.trim()) continue;
    if (keys.has(f.key.trim())) return `字段 key 重复：${f.key.trim()}`;
    keys.add(f.key.trim());
    if (f.type === "select") {
      const opts = f.options
        .split(/[,，]/)
        .map((o) => o.trim())
        .filter(Boolean);
      if (!opts.length) return `字段 ${f.key} 为下拉选择，必须提供选项`;
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
    if (nodeMode.value === "visual") formData.node_config = nodesToConfig();
    if (formMode.value === "visual") formData.form_schema = fieldsToSchema();
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

onBeforeUnmount(() => {
  window.removeEventListener("mousemove", onNodeDragMove);
  window.removeEventListener("mouseup", onNodeDragEnd);
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

/* ============ SVG 画布 ============ */
.graph-canvas-container {
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  background: var(--el-fill-color-lighter);
  overflow: hidden;
  margin-bottom: 12px;
}

.graph-canvas {
  display: block;
  cursor: default;
  user-select: none;
}

.graph-edge-line {
  stroke: var(--el-border-color);
  stroke-width: 2;
  cursor: pointer;
}

.graph-edge-line.conditional {
  stroke: var(--el-color-warning);
  stroke-dasharray: 6 4;
}

.graph-edge-line:hover {
  stroke-width: 3;
}

.graph-edge-label {
  fill: var(--el-color-warning);
  font-size: 10px;
  text-anchor: middle;
  cursor: pointer;
  pointer-events: none;
}

.graph-node-rect {
  stroke-width: 2;
  cursor: grab;
}

.node-type-start {
  fill: #e8f5e9;
  stroke: #4caf50;
}

.node-type-end {
  fill: #ffebee;
  stroke: #f44336;
}

.node-type-task {
  fill: #e3f2fd;
  stroke: #2196f3;
}

.node-type-exclusive_gateway {
  fill: #fff3e0;
  stroke: #ff9800;
}

.node-type-parallel_gateway_fork {
  fill: #f3e5f5;
  stroke: #9c27b0;
}

.node-type-parallel_gateway_join {
  fill: #f3e5f5;
  stroke: #9c27b0;
}

.node-type-subprocess {
  fill: #efebe9;
  stroke: #795548;
}

.graph-node-rect.selected {
  stroke-width: 4;
  filter: drop-shadow(0 0 4px rgba(0, 0, 0, 0.3));
}

.graph-node-label {
  fill: var(--el-text-color-primary);
  font-size: 11px;
  text-anchor: middle;
  dominant-baseline: middle;
  pointer-events: none;
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

.hint-inline {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

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
