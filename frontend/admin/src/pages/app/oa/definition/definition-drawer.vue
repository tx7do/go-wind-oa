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
        <ElInput v-model="formData.remark" placeholder="流程用途说明（可选）" :rows="2" type="textarea" />
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
          <!-- 流程画布：开始 → 节点 → 结束，实时预览，点击节点定位卡片 -->
          <div class="flow-canvas" :class="{ dragging: dragIndex !== null }">
            <div class="flow-pill flow-start">开始</div>
            <template v-for="(node, ni) in nodeDrafts" :key="'c' + ni">
              <div class="flow-arrow">→</div>
              <div
                class="flow-chip"
                :class="{ active: activeNode === ni }"
                @click="focusNode(ni)"
              >
                <div class="chip-title">
                  <span class="chip-name">{{ ni + 1 }}</span>
                  <span class="chip-strategy" :class="node.strategy === 'ANY' ? 'any' : 'all'">
                    {{ node.strategy === "ANY" ? "或签" : "会签" }}
                  </span>
                </div>
                <div class="chip-approvers">
                  {{ node.approvers.map(approverLabel).join("、") || "未配置" }}
                </div>
              </div>
            </template>
            <div class="flow-arrow">→</div>
            <div class="flow-pill flow-end">结束</div>
          </div>

          <!-- 节点模板快加 + 添加节点 -->
          <div class="node-toolbar">
            <ElDropdown @command="addNodeByTemplate">
              <ElButton type="primary" plain size="small">
                + 从模板添加节点<ElIcon class="el-icon--right"><ArrowDown /></ElIcon>
              </ElButton>
              <template #dropdown>
                <ElDropdownMenu>
                  <ElDropdownItem command="leader-all">主管审批 · 会签</ElDropdownItem>
                  <ElDropdownItem command="leader-any">主管审批 · 或签</ElDropdownItem>
                  <ElDropdownItem command="user-all">指定用户 · 会签</ElDropdownItem>
                  <ElDropdownItem command="position-all">职位持有者 · 会签</ElDropdownItem>
                </ElDropdownMenu>
              </template>
            </ElDropdown>
            <ElButton size="small" plain @click="addNode">+ 空白节点</ElButton>
          </div>

          <!-- 节点卡片（可拖拽排序） -->
          <div
            v-for="(node, ni) in nodeDrafts"
            :key="'n' + ni"
            class="node-card"
            :class="{ 'drag-over': dragOverIndex === ni, dragging: dragIndex === ni }"
            :data-node="ni"
            draggable="true"
            @dragstart="onDragStart(ni)"
            @dragover.prevent="onDragOver(ni)"
            @dragleave="dragOverIndex = null"
            @drop.prevent="onDrop(ni)"
            @dragend="onDragEnd"
          >
            <div class="node-head">
              <span class="node-title">
                <ElIcon class="drag-handle"><Rank /></ElIcon>
                节点 {{ ni + 1 }}
              </span>
              <div class="node-ops">
                <ElButton size="small" text :disabled="ni === 0" @click="moveNode(ni, -1)">上移</ElButton>
                <ElButton size="small" text :disabled="ni === nodeDrafts.length - 1" @click="moveNode(ni, 1)">下移</ElButton>
                <ElButton size="small" text type="danger" :disabled="nodeDrafts.length <= 1" @click="nodeDrafts.splice(ni, 1)">删除</ElButton>
              </div>
            </div>

            <ElFormItem label="审批策略" label-width="80px">
              <ElRadioGroup v-model="node.strategy">
                <ElRadio value="ALL">会签（全员通过）</ElRadio>
                <ElRadio value="ANY">或签（一人通过）</ElRadio>
              </ElRadioGroup>
            </ElFormItem>

            <div class="approver-list">
              <div v-for="(ap, ai) in node.approvers" :key="ai" class="approver-row">
                <ElSelect v-model="ap.type" style="width: 150px" @change="onApproverTypeChange(ap)">
                  <ElOption label="指定用户" value="USER" />
                  <ElOption label="申请人主管" value="LEADER" />
                  <ElOption label="职位持有者" value="POSITION" />
                </ElSelect>

                <!-- 用户选择器：按姓名选择（列表加载失败回退手填 ID） -->
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
                <!-- 职位选择器 -->
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

                <ElButton size="small" text type="danger" :disabled="node.approvers.length <= 1" @click="node.approvers.splice(ai, 1)">移除</ElButton>
              </div>
              <ElButton size="small" plain @click="node.approvers.push({ type: 'USER', id: undefined })">+ 添加审批人</ElButton>
            </div>
          </div>

          <div v-if="nodeDrafts.length" class="json-preview">
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
                <ElInput v-if="row.type === 'select'" v-model="row.options" placeholder="普通,紧急" size="small" />
                <span v-else class="hint-inline">-</span>
              </template>
            </ElTableColumn>
            <ElTableColumn label="操作" width="60" align="center">
              <template #default="{ $index }">
                <ElButton size="small" text type="danger" @click="formFieldDrafts.splice($index, 1)">删</ElButton>
              </template>
            </ElTableColumn>
          </ElTable>
          <ElButton size="small" plain class="add-field-btn" @click="formFieldDrafts.push({ key: '', label: '', type: 'text', required: false, options: '' })">
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
  ArrowDown,
  Rank,
} from "@element-plus/icons-vue";
import {
  ElButton,
  ElCheckbox,
  ElDropdown,
  ElDropdownItem,
  ElDropdownMenu,
  ElIcon,
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
import { ref, reactive, computed, watch, onMounted } from "vue";

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

/** 节点草稿：策略 + 审批人列表。 */
interface NodeDraft {
  strategy: "ALL" | "ANY";
  approvers: ApproverDraft[];
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
const nodeDrafts = ref<NodeDraft[]>([]);
const formFieldDrafts = ref<FieldDraft[]>([]);

// 审批人选择器数据源（用户/职位，列表加载失败时选择器回退手填 ID）。
const users = ref<identityservicev1_User[]>([]);
const positions = ref<identityservicev1_Position[]>([]);

// 拖拽排序与画布定位状态。
const dragIndex = ref<number | null>(null);
const dragOverIndex = ref<number | null>(null);
const activeNode = ref<number | null>(null);

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
          if ((f.type === "select") && (!Array.isArray(f.options) || f.options.length === 0)) {
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
        let nodes: any[];
        try {
          nodes = JSON.parse(value);
        } catch {
          callback(new Error("node_config 必须是合法的 JSON 数组"));
          return;
        }
        if (!Array.isArray(nodes) || nodes.length === 0) {
          callback(new Error("node_config 必须是非空节点数组"));
          return;
        }
        for (const node of nodes) {
          const strategy = node.strategy ?? "ALL";
          if (strategy !== "ALL" && strategy !== "ANY") {
            callback(new Error(`非法审批策略 ${strategy}（仅 ALL 会签 / ANY 或签）`));
            return;
          }
          const approvers = Array.isArray(node.approvers) && node.approvers.length > 0
            ? node.approvers
            : (node.approver_type === "USER" && node.approver ? [{ type: "USER", id: node.approver }] : []);
          if (approvers.length === 0) {
            callback(new Error("每个节点至少需要一个审批人（approvers 或旧格式 approver_type+approver）"));
            return;
          }
          for (const approver of approvers) {
            if (!["USER", "LEADER", "POSITION"].includes(approver.type)) {
              callback(new Error(`非法审批人类型 ${approver.type}（仅 USER / LEADER / POSITION）`));
              return;
            }
            if (approver.type !== "LEADER" && !approver.id) {
              callback(new Error("USER / POSITION 类型审批人必须提供 id"));
              return;
            }
          }
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

// ============ 节点编辑 ============

function addNode() {
  nodeDrafts.value.push({ strategy: "ALL", approvers: [{ type: "LEADER" }] });
}

/** 模板快加：常用节点形态一键创建。 */
function addNodeByTemplate(command: string) {
  switch (command) {
    case "leader-all":
      nodeDrafts.value.push({ strategy: "ALL", approvers: [{ type: "LEADER" }] });
      break;
    case "leader-any":
      nodeDrafts.value.push({ strategy: "ANY", approvers: [{ type: "LEADER" }] });
      break;
    case "user-all":
      nodeDrafts.value.push({ strategy: "ALL", approvers: [{ type: "USER", id: undefined }] });
      break;
    case "position-all":
      nodeDrafts.value.push({ strategy: "ALL", approvers: [{ type: "POSITION", id: undefined }] });
      break;
  }
}

function moveNode(index: number, delta: number) {
  const target = index + delta;
  if (target < 0 || target >= nodeDrafts.value.length) return;
  const list = nodeDrafts.value;
  [list[index], list[target]] = [list[target], list[index]];
}

function onApproverTypeChange(ap: ApproverDraft) {
  if (ap.type === "LEADER") ap.id = undefined;
  else if (!ap.id) ap.id = 1;
}

// ============ 拖拽排序（HTML5 原生） ============

function onDragStart(index: number) {
  dragIndex.value = index;
}

function onDragOver(index: number) {
  if (dragIndex.value === null || dragIndex.value === index) return;
  dragOverIndex.value = index;
}

function onDrop(index: number) {
  const from = dragIndex.value;
  if (from === null || from === index) {
    onDragEnd();
    return;
  }
  const list = nodeDrafts.value;
  const [moved] = list.splice(from, 1);
  list.splice(index, 0, moved);
  onDragEnd();
}

function onDragEnd() {
  dragIndex.value = null;
  dragOverIndex.value = null;
}

// ============ 画布 ============

/** 画布节点 chip 的审批人短标签。 */
function approverLabel(ap: ApproverDraft): string {
  switch (ap.type) {
    case "LEADER":
      return "主管";
    case "USER": {
      const u = users.value.find((x) => x.id === ap.id);
      return u ? userDisplayName(u) : ap.id ? `用户#${ap.id}` : "未选用户";
    }
    case "POSITION": {
      const p = positions.value.find((x) => x.id === ap.id);
      return p ? positionDisplayName(p) : ap.id ? `职位#${ap.id}` : "未选职位";
    }
  }
}

/** 点击画布节点：高亮并滚动到对应卡片。 */
function focusNode(index: number) {
  activeNode.value = index;
  const el = document.querySelector(`.node-card[data-node="${index}"]`);
  el?.scrollIntoView({ behavior: "smooth", block: "center" });
}

// ============ 序列化 ============

/** 节点草稿 → node_config JSON（新格式）。 */
function nodesToConfig(): string {
  const nodes = nodeDrafts.value
    .filter((n) => n.approvers.length > 0)
    .map((n) => ({
      approvers: n.approvers.map((a) => (a.type === "LEADER" ? { type: a.type } : { type: a.type, id: a.id ?? 0 })),
      strategy: n.strategy,
    }));
  return JSON.stringify(nodes);
}

/** JSON 文本 → 节点草稿（兼容旧单人格式），用于切到可视化模式时解析。 */
function configToNodes(json: string) {
  const drafts: NodeDraft[] = [];
  try {
    const parsed = JSON.parse(json);
    if (!Array.isArray(parsed)) return drafts;
    for (const node of parsed) {
      const approvers: ApproverDraft[] = Array.isArray(node.approvers) && node.approvers.length > 0
        ? node.approvers
            .filter((a: any) => ["USER", "LEADER", "POSITION"].includes(a.type))
            .map((a: any) => ({ type: a.type, id: a.id }))
        : (node.approver_type === "USER" && node.approver ? [{ type: "USER" as const, id: node.approver }] : []);
      if (approvers.length === 0) continue;
      drafts.push({
        strategy: node.strategy === "ANY" ? "ANY" : "ALL",
        approvers,
      });
    }
  } catch {
    /* 非法 JSON：返回已解析部分 */
  }
  return drafts;
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
        type: (["text", "textarea", "number", "date", "select"].includes(f.type) ? f.type : "text") as FieldDraft["type"],
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
    const drafts = configToNodes(formData.node_config);
    if (drafts.length) nodeDrafts.value = drafts;
    if (!nodeDrafts.value.length) addNode();
  } else {
    formData.node_config = nodeDrafts.value.length ? nodesToConfig() : formData.node_config;
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
  nodeDrafts.value = [];
  formFieldDrafts.value = [];
  activeNode.value = null;
  addNode();
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

/** 可视化模式校验：每节点至少一审批人，USER/POSITION 必须有 id，select 必须有选项。 */
function validateDrafts(): string | null {
  if (!nodeDrafts.value.length) return "至少需要一个审批节点";
  for (let i = 0; i < nodeDrafts.value.length; i++) {
    const node = nodeDrafts.value[i];
    if (!node.approvers.length) return `节点 ${i + 1} 至少需要一个审批人`;
    for (const ap of node.approvers) {
      if (ap.type !== "LEADER" && (!ap.id || ap.id <= 0)) {
        return `节点 ${i + 1} 的${ap.type === "USER" ? "用户" : "职位"}必须选择`;
      }
    }
  }
  const keys = new Set<string>();
  for (const f of formFieldDrafts.value) {
    if (!f.key.trim()) continue;
    if (keys.has(f.key.trim())) return `字段 key 重复：${f.key.trim()}`;
    keys.add(f.key.trim());
    if (f.type === "select") {
      const opts = f.options.split(/[,，]/).map((o) => o.trim()).filter(Boolean);
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

/* ============ 流程画布 ============ */
.flow-canvas {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 12px;
  background: var(--el-fill-color-lighter);
  border-radius: 6px;
  overflow-x: auto;

  &.dragging {
    outline: 1px dashed var(--el-color-primary-light-5);
  }
}

.flow-pill {
  flex-shrink: 0;
  padding: 6px 14px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;

  &.flow-start {
    background: var(--el-color-success-light-8);
    color: var(--el-color-success);
    border: 1px solid var(--el-color-success-light-5);
  }

  &.flow-end {
    background: var(--el-fill-color);
    color: var(--el-text-color-secondary);
    border: 1px solid var(--el-border-color);
  }
}

.flow-arrow {
  flex-shrink: 0;
  color: var(--el-text-color-placeholder);
  font-size: 14px;
}

.flow-chip {
  flex-shrink: 0;
  min-width: 84px;
  max-width: 150px;
  padding: 6px 10px;
  border-radius: 6px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-color-primary-light-7);
  cursor: pointer;
  transition: box-shadow 0.2s, border-color 0.2s;

  &:hover,
  &.active {
    border-color: var(--el-color-primary);
    box-shadow: 0 0 0 2px var(--el-color-primary-light-8);
  }

  .chip-title {
    display: flex;
    align-items: center;
    gap: 6px;

    .chip-name {
      font-weight: 700;
      font-size: 12px;
    }

    .chip-strategy {
      font-size: 10px;
      padding: 0 6px;
      border-radius: 999px;

      &.all {
        background: var(--el-color-primary-light-9);
        color: var(--el-color-primary);
      }

      &.any {
        background: var(--el-color-warning-light-9);
        color: var(--el-color-warning);
      }
    }
  }

  .chip-approvers {
    margin-top: 2px;
    font-size: 11px;
    color: var(--el-text-color-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}

/* ============ 节点卡片 ============ */
.node-toolbar {
  display: flex;
  gap: 8px;
}

.node-card {
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  padding: 12px;
  background: var(--el-bg-color);
  transition: border-color 0.15s, opacity 0.15s;

  &.drag-over {
    border-color: var(--el-color-primary);
    border-style: dashed;
  }

  &.dragging {
    opacity: 0.45;
  }
}

.node-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  cursor: grab;
}

.node-title {
  font-weight: 600;
  font-size: 13px;
  display: inline-flex;
  align-items: center;
  gap: 4px;

  .drag-handle {
    color: var(--el-text-color-placeholder);
  }
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
