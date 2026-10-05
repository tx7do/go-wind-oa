<template>
  <ProModal
    v-model:visible="visible"
    :title="$t('pages.oa.delegation.drawer.title')"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <div class="hint">
      {{ $t("pages.oa.delegation.drawer.hint") }}
    </div>
    <ElForm label-width="100px">
      <ElFormItem :label="$t('pages.oa.delegation.drawer.fieldDelegator')">
        <ElSelect
          v-model="form.delegatorUserId"
          filterable
          :placeholder="$t('pages.oa.delegation.drawer.placeholderDelegator')"
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
      <ElFormItem :label="$t('pages.oa.delegation.drawer.fieldDelegate')">
        <ElSelect
          v-model="form.delegateUserId"
          filterable
          :placeholder="$t('pages.oa.delegation.drawer.placeholderDelegate')"
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
        <ElButton @click="handleClose">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="creating" @click="submit">
          {{ $t("common.button.save") }}
        </ElButton>
      </div>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import { ElButton, ElForm, ElFormItem, ElMessage, ElSelect, ElOption } from "element-plus";
import { reactive, ref, onMounted } from "vue";
import ProModal from "@/components/Pro/ProModal/index.vue";
import { useSetWorkflowDelegation, fetchUsers, userDisplayName } from "@/api/composables";
import { DRAWER_WIDTH } from "@/constants";
import type { identityservicev1_User } from "@/api/generated/admin/service/v1";
import { $t } from "@/core/i18n";

const emit = defineEmits(["success"]);

const visible = ref(false);
const creating = ref(false);
const form = reactive({ delegatorUserId: 0, delegateUserId: 0 });
const users = ref<identityservicev1_User[]>([]);

const createMutation = useSetWorkflowDelegation({
  onSuccess: () => {
    ElMessage.success($t("common.notification.saveSuccess"));
    visible.value = false;
    emit("success");
  },
  onError: (err: Error) => ElMessage.error(err.message || $t("common.notification.saveFailed")),
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
    ElMessage.warning($t("pages.oa.delegation.drawer.selectRequired"));
    return;
  }
  if (form.delegatorUserId === form.delegateUserId) {
    ElMessage.warning($t("pages.oa.delegation.drawer.sameRequired"));
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
