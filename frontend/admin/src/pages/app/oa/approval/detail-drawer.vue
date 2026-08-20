<template>
  <ProModal
    v-model:visible="visible"
    title="审批详情"
    :config="{ component: 'drawer', drawer: { size: '60%', closeOnClickModal: false } }"
  >
    <div v-if="detail" class="detail-body">
      <div class="section-title">任务信息</div>
      <ElDescriptions :column="2" border size="small">
        <ElDescriptionsItem label="任务ID">{{ taskId }}</ElDescriptionsItem>
        <ElDescriptionsItem label="节点索引">{{ detail.task?.nodeIndex ?? '-' }}</ElDescriptionsItem>
      </ElDescriptions>

      <div class="section-title">申请表单数据</div>
      <ElDescriptions v-if="formEntries.length" :column="1" border size="small">
        <ElDescriptionsItem v-for="e in formEntries" :key="e[0]" :label="e[0]">
          {{ e[1] }}
        </ElDescriptionsItem>
      </ElDescriptions>
      <ElInput v-else :model-value="detail.formData ?? '-'" type="textarea" :rows="6" readonly />

      <div class="section-title">审批历史</div>
      <ElTable :data="detail.logs ?? []" border size="small">
        <ElTableColumn label="动作" width="110">
          <template #default="{ row }">{{ auditActionLabel(row.logAction) }}</template>
        </ElTableColumn>
        <ElTableColumn label="时间" width="170">
          <template #default="{ row }">{{ fmtTime(row.createdAt) }}</template>
        </ElTableColumn>
        <ElTableColumn prop="comment" label="意见" min-width="160" />
      </ElTable>

      <div class="section-title">审批操作</div>
      <ElInput
        v-model="comment"
        type="textarea"
        :rows="2"
        placeholder="审批意见（可选）"
        style="margin-bottom: 12px"
      />
      <div class="actions">
        <ElButton type="primary" :loading="acting" @click="doAudit('APPROVE')">通过</ElButton>
        <ElButton type="danger" plain :loading="acting" @click="doAudit('REJECT')">驳回</ElButton>
        <ElButton plain :loading="acting" @click="doForward">转办</ElButton>
      </div>
    </div>
  </ProModal>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import {
  ElButton,
  ElDescriptions,
  ElDescriptionsItem,
  ElInput,
  ElMessage,
  ElMessageBox,
  ElTable,
  ElTableColumn,
} from "element-plus";

import ProModal from "@/components/Pro/ProModal/index.vue";
import { auditActionLabel, fetchTaskDetail, useAuditTask } from "@/api/composables";
import type {
  oaservicev1_GetTaskResponse,
  oaservicev1_AuditAction,
} from "@/api/generated/admin/service/v1";

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

const auditMutation = useAuditTask({
  onSuccess: () => {
    ElMessage.success("操作成功");
    visible.value = false;
    emit("success");
  },
  onError: (err: Error) => ElMessage.error(err.message || "操作失败"),
});

async function open(id: number) {
  taskId.value = id;
  comment.value = "";
  try {
    detail.value = await fetchTaskDetail(id);
    visible.value = true;
  } catch {
    detail.value = null;
  }
}

function doAudit(action: oaservicev1_AuditAction, forwardTo?: number) {
  acting.value = true;
  auditMutation.mutate(
    { taskId: taskId.value, action, comment: comment.value, forwardTo },
    { onSettled: () => (acting.value = false) }
  );
}

async function doForward() {
  try {
    const { value } = await ElMessageBox.prompt("请输入转办目标用户ID", "转办", {
      inputPattern: /^[1-9]\d*$/,
      inputErrorMessage: "请输入正整数用户ID",
    });
    doAudit("FORWARD", Number(value));
  } catch {
    /* 用户取消 */
  }
}

function fmtTime(v?: string) {
  return v ? String(v).replace("T", " ").slice(0, 19) : "-";
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
