<template>
  <ProModal
    v-model:visible="visible"
    :title="$t('pages.oa.leave.typeDrawer.createTitle')"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <ElForm label-width="80px">
      <ElFormItem :label="$t('pages.oa.leave.typeDrawer.fieldCode')">
        <ElInput v-model="form.code" placeholder="ANNUAL / SICK / PERSONAL" />
      </ElFormItem>
      <ElFormItem :label="$t('pages.oa.leave.typeDrawer.fieldName')">
        <ElInput
          v-model="form.name"
          :placeholder="$t('pages.oa.leave.typeDrawer.namePlaceholder')"
        />
      </ElFormItem>
    </ElForm>
    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="handleClose">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="creating" @click="submit">
          {{ $t("pages.oa.leave.typeDrawer.create") }}
        </ElButton>
      </div>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import { ElButton, ElForm, ElFormItem, ElInput, ElMessage } from "element-plus";
import { reactive, ref } from "vue";
import ProModal from "@/components/Pro/ProModal/index.vue";
import { useCreateLeaveType } from "@/api/composables";
import { DRAWER_WIDTH } from "@/constants";
import { useOaDrawer } from "../use-oa-drawer";
import { $t } from "@/core/i18n";

const emit = defineEmits(["success"]);

const { visible, open, close } = useOaDrawer();
const creating = ref(false);
const form = reactive({ code: "", name: "" });

const createMutation = useCreateLeaveType({
  onSuccess: () => {
    ElMessage.success($t("pages.oa.leave.typeDrawer.createSuccess"));
    close();
    emit("success");
  },
  onError: (err: Error) => ElMessage.error(err.message || $t("common.notification.createFailed")),
});

const handleClose = close;

function submit() {
  if (!form.code || !form.name) {
    ElMessage.warning($t("pages.oa.leave.typeDrawer.codeNameRequired"));
    return;
  }
  creating.value = true;
  createMutation.mutate(
    { data: { code: form.code, name: form.name } },
    { onSettled: () => (creating.value = false) }
  );
}

defineExpose({ open });
</script>
