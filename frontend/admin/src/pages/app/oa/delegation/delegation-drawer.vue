<template>
  <ProModal
    v-model:visible="visible"
    title="设置审批委托"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <div class="hint">
      设置后，系统会将原本分配给委托人的待办审批任务自动转发给被委托人（代理人）。一人仅可指定一名代理人。
    </div>
    <ElForm label-width="100px">
      <ElFormItem label="委托人">
        <ElSelect
          v-model="form.delegatorUserId"
          filterable
          placeholder="搜索并选择委托人"
          style="width: 320px"
        >
          <ElOption
            v-for="u in users"
            :key="u.id"
            :label="userDisplayName(u)"
            :value="u.id as number"
          />
        </ElSelect>
      </ElFormItem>
      <ElFormItem label="被委托人">
        <ElSelect
          v-model="form.delegateUserId"
          filterable
          placeholder="搜索并选择被委托人"
          style="width: 320px"
        >
          <ElOption
            v-for="u in users"
            :key="u.id"
            :label="userDisplayName(u)"
            :value="u.id as number"
          />
        </ElSelect>
      </ElFormItem>
    </ElForm>
    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="handleClose">取消</ElButton>
        <ElButton type="primary" :loading="creating" @click="submit">保存</ElButton>
      </div>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import { ElButton, ElForm, ElFormItem, ElInput, ElMessage, ElSelect, ElOption } from "element-plus";
import { reactive, ref, onMounted } from "vue";
import ProModal from "@/components/Pro/ProModal/index.vue";
import { useSetWorkflowDelegation, fetchUsers, userDisplayName } from "@/api/composables";
import { DRAWER_WIDTH } from "@/constants";
import type { identityservicev1_User } from "@/api/generated/admin/service/v1";

const emit = defineEmits(["success"]);

const visible = ref(false);
const creating = ref(false);
const form = reactive({ delegatorUserId: 0, delegateUserId: 0 });
const users = ref<identityservicev1_User[]>([]);

const createMutation = useSetWorkflowDelegation({
  onSuccess: () => {
    ElMessage.success("已保存");
    visible.value = false;
    emit("success");
  },
  onError: (err: Error) => ElMessage.error(err.message || "保存失败"),
});

onMounted(async () => {
  try {
    const resp = await fetchUsers();
    users.value = resp.items ?? [];
  } catch {
    users.value = [];
  }
});

function open() {
  form.delegatorUserId = 0;
  form.delegateUserId = 0;
  visible.value = true;
}

function handleClose() {
  visible.value = false;
}

function submit() {
  if (!form.delegatorUserId || !form.delegateUserId) {
    ElMessage.warning("请选择委托人和被委托人");
    return;
  }
  if (form.delegatorUserId === form.delegateUserId) {
    ElMessage.warning("委托人和被委托人不能相同");
    return;
  }
  creating.value = true;
  createMutation.mutate(
    {
      data: {
        delegatorUserId: form.delegatorUserId,
        delegateUserId: form.delegateUserId,
      },
    },
    { onSettled: () => (creating.value = false) }
  );
}

defineExpose({ open });
</script>

<style lang="scss" scoped>
.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 16px;
}
</style>
