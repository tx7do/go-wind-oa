<template>
  <ProModal
    v-model:visible="visible"
    :title="$t('pages.oa.approval.detail.title')"
    :config="{ component: 'drawer', drawer: { size: '60%', closeOnClickModal: false } }"
  >
    <div v-if="detail" class="detail-body">
      <div class="section-title">{{ $t("pages.oa.approval.detail.progress") }}</div>
      <ElSteps :active="progressActive" finish-status="success" align-center>
        <ElStep
          v-for="(step, idx) in progressSteps"
          :key="idx"
          :title="step.title"
          :description="step.description"
        />
      </ElSteps>

      <div class="section-title">{{ $t("pages.oa.approval.detail.taskInfo") }}</div>
      <ElDescriptions :column="2" border size="small">
        <ElDescriptionsItem :label="$t('pages.oa.approval.detail.fieldTaskId')">
          {{ taskId }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.oa.approval.detail.fieldNodeId')">
          {{ detail.task?.nodeId ?? "-" }}
        </ElDescriptionsItem>
      </ElDescriptions>

      <div class="section-title">{{ $t("pages.oa.approval.detail.formData") }}</div>
      <ElDescriptions v-if="formEntries.length" :column="1" border size="small">
        <ElDescriptionsItem v-for="e in formEntries" :key="e[0]" :label="e[0]">
          {{ e[1] }}
        </ElDescriptionsItem>
      </ElDescriptions>
      <ElInput v-else :model-value="detail.formData ?? '-'" type="textarea" :rows="6" readonly />

      <div class="section-title">{{ $t("pages.oa.approval.detail.history") }}</div>
      <ElTable :data="detail.logs ?? []" border size="small">
        <ElTableColumn :label="$t('pages.oa.approval.detail.colAction')" width="110">
          <template #default="{ row }">{{ auditActionLabel(row.logAction) }}</template>
        </ElTableColumn>
        <ElTableColumn :label="$t('pages.oa.approval.detail.colTime')" width="170">
          <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
        </ElTableColumn>
        <ElTableColumn
          prop="comment"
          :label="$t('pages.oa.approval.detail.colComment')"
          min-width="160"
        />
      </ElTable>

      <div class="section-title">{{ $t("pages.oa.approval.detail.operation") }}</div>
      <ElInput
        v-model="comment"
        type="textarea"
        :rows="2"
        :placeholder="$t('pages.oa.approval.detail.commentPlaceholder')"
        style="margin-bottom: 12px"
      />
      <div class="actions">
        <ElButton type="primary" :loading="acting" @click="doAudit('APPROVE')">
          {{ $t("pages.oa.approval.detail.approve") }}
        </ElButton>
        <ElButton type="danger" plain :loading="acting" @click="doAudit('REJECT')">
          {{ $t("pages.oa.approval.detail.reject") }}
        </ElButton>
        <ElButton plain :loading="acting" @click="doForward">
          {{ $t("pages.oa.approval.detail.forward") }}
        </ElButton>
        <ElButton plain :loading="acting" @click="doAddApprover">
          {{ $t("pages.oa.approval.detail.addApprover") }}
        </ElButton>
      </div>
    </div>

    <!-- 转办/加签目标用户选择（替代手输用户ID） -->
    <ElDialog
      v-model="selectorVisible"
      :title="selectorTitle"
      width="380px"
      append-to-body
      @close="cancelSelector"
    >
      <ElSelect
        v-model="selectorValue"
        filterable
        :placeholder="$t('pages.oa.approval.detail.placeholderSelectUser')"
        style="width: 100%"
      >
        <ElOption
          v-for="u in users"
          :key="u.id"
          :label="`${userDisplayName(u)} (#${u.id})`"
          :value="u.id as number"
        />
      </ElSelect>
      <template #footer>
        <ElButton @click="cancelSelector">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :disabled="!selectorValue" @click="confirmSelector">
          {{ $t("common.button.confirm") }}
        </ElButton>
      </template>
    </ElDialog>
  </ProModal>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from "vue";
import {
  ElButton,
  ElDescriptions,
  ElDescriptionsItem,
  ElDialog,
  ElInput,
  ElMessage,
  ElOption,
  ElSelect,
  ElStep,
  ElSteps,
  ElTable,
  ElTableColumn,
} from "element-plus";

import ProModal from "@/components/Pro/ProModal/index.vue";
import {
  auditActionLabel,
  fetchTaskDetail,
  fetchUsers,
  userDisplayName,
  useAuditTask,
} from "@/api/composables";
import type {
  identityservicev1_User,
  oaservicev1_GetTaskResponse,
  oaservicev1_AuditAction,
} from "@/api/generated/admin/service/v1";
import { $t } from "@/core/i18n";
import { formatDateTime } from "@/utils/date";

const emit = defineEmits(["success"]);

const visible = ref(false);
const acting = ref(false);
const taskId = ref(0);
const detail = ref<oaservicev1_GetTaskResponse | null>(null);
const comment = ref("");

// formData 可解析为 JSON 对象时按 key-value 展示，否则原文。
const formEntries = computed<[string, string][]>(() => {
  const raw = detail.value?.formData;
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return Object.entries(parsed).map(([k, v]) => [k, String(v)] as [string, string]);
    }
  } catch {
    /* 原文展示 */
  }
  return [];
});

// 审批进度：从审批日志构建步骤列表，当前待办节点高亮。
const progressSteps = computed(() => {
  const logs = detail.value?.logs ?? [];
  const currentNode = detail.value?.task?.nodeId;
  const steps: { title: string; description: string }[] = [];
  for (const log of logs) {
    const action = auditActionLabel(log.logAction);
    steps.push({
      title: String(log.nodeId ?? "—"),
      description: action + " · " + formatDateTime(log.createdAt),
    });
  }
  if (currentNode && !steps.some((s) => s.title === currentNode)) {
    steps.push({
      title: String(currentNode),
      description: $t("pages.oa.approval.detail.pendingApproval"),
    });
  }
  return steps;
});

const progressActive = computed(() => {
  const currentNode = detail.value?.task?.nodeId;
  if (!currentNode) return -1;
  const idx = progressSteps.value.findIndex((s) => s.title === currentNode);
  return idx;
});

const auditMutation = useAuditTask({
  onSuccess: () => {
    ElMessage.success($t("common.message.success"));
    visible.value = false;
    emit("success");
  },
  onError: (err: Error) => ElMessage.error(err.message || $t("common.message.operationFailed")),
});

async function open(id: number) {
  taskId.value = id;
  comment.value = "";
  try {
    detail.value = await fetchTaskDetail(id);
    visible.value = true;
  } catch {
    detail.value = null;
    ElMessage.error($t("common.message.getDetailFailed"));
  }
}

function doAudit(action: oaservicev1_AuditAction, forwardTo?: number, additionalApprover?: number) {
  acting.value = true;
  auditMutation.mutate(
    {
      taskId: taskId.value,
      action,
      comment: comment.value,
      forwardTo,
      additionalApprover,
    },
    { onSettled: () => (acting.value = false) }
  );
}

async function doForward() {
  const uid = await promptUser($t("pages.oa.approval.detail.forward"));
  if (uid) doAudit("FORWARD", uid);
}

async function doAddApprover() {
  const uid = await promptUser($t("pages.oa.approval.detail.addApprover"));
  if (uid) doAudit("ADD_APPROVER", undefined, uid);
}

// ============ 人员选择弹窗（转办/加签共用，Promise 化） ============

// 人员列表尽力加载：失败时选择器为空，不阻塞审批主流程
const users = ref<identityservicev1_User[]>([]);
onMounted(async () => {
  try {
    const resp = await fetchUsers();
    users.value = resp.items ?? [];
  } catch {
    users.value = [];
  }
});

const selectorVisible = ref(false);
const selectorTitle = ref("");
const selectorValue = ref<number | undefined>(undefined);
let selectorResolve: ((uid: number | null) => void) | null = null;

function promptUser(title: string): Promise<number | null> {
  selectorTitle.value = title;
  selectorValue.value = undefined;
  selectorVisible.value = true;
  return new Promise((resolve) => {
    selectorResolve = resolve;
  });
}

function confirmSelector() {
  const uid = selectorValue.value ?? null;
  selectorVisible.value = false;
  selectorResolve?.(uid);
  selectorResolve = null;
}

function cancelSelector() {
  selectorVisible.value = false;
  selectorResolve?.(null);
  selectorResolve = null;
}

defineExpose({ open });
</script>

<style lang="scss" scoped>
.detail-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.section-title {
  font-size: 14px;
  font-weight: 600;
  margin-top: 8px;
}
.actions {
  display: flex;
  gap: 12px;
}
</style>
