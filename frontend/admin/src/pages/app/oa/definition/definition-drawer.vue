<template>
  <ProModal
    v-model:visible="visible"
    :title="title"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <ElForm ref="formRef" :model="formData" :rules="formRules" label-width="140px">
      <ElFormItem :label="$t('pages.oa.definition.fieldName')" prop="name">
        <ElInput v-model="formData.name" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.oa.definition.fieldCode')" prop="code">
        <ElInput v-model="formData.code" :placeholder="$t('common.placeholder.input')" clearable />
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

      <ElFormItem :label="$t('pages.oa.definition.fieldDescription')" prop="description">
        <ElInput
          v-model="formData.description"
          type="textarea"
          :placeholder="$t('common.placeholder.input')"
          :rows="3"
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.oa.definition.fieldNodeConfig')" prop="node_config">
        <ElInput
          v-model="formData.node_config"
          type="textarea"
          :placeholder="$t('common.placeholder.input')"
          :rows="8"
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.oa.definition.fieldFormSchema')" prop="form_schema">
        <ElInput
          v-model="formData.form_schema"
          type="textarea"
          :placeholder="$t('common.placeholder.input')"
          :rows="8"
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
import { ref, reactive, computed, watch } from "vue";

import { useCreateWorkflowDefinition } from "@/api/composables";
import { $t } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";
import ProModal from "@/components/Pro/ProModal/index.vue";

const emit = defineEmits(["success"]);

const { mutateAsync: createDefinition } = useCreateWorkflowDefinition();

const visible = ref(false);
const loading = ref(false);
const formRef = ref<FormInstance>();

// 表单数据
const formData = reactive({
  name: "",
  code: "",
  version: 1,
  description: "",
  node_config: "",
  form_schema: "",
});

// 表单验证规则
const formRules: FormRules = {
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  code: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  version: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
};

const title = computed(() =>
  $t("common.modal.create", { moduleName: $t("pages.oa.definition.title") })
);

// 重置表单
function resetForm() {
  formData.name = "";
  formData.code = "";
  formData.version = 1;
  formData.description = "";
  formData.node_config = "";
  formData.form_schema = "";
  formRef.value?.clearValidate();
}

// 打开抽屉（仅创建模式）
function open(_row?: any) {
  visible.value = true;
  resetForm();
}

// 关闭抽屉
function handleClose() {
  visible.value = false;
  resetForm();
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value) return;

  const valid = await formRef.value.validate().then(
    () => true,
    () => false
  );
  if (!valid) return;

  try {
    loading.value = true;
    await createDefinition({ data: formData });
    ElMessage.success($t("common.notification.createSuccess"));
    emit("success");
    handleClose();
  } catch {
    ElMessage.error($t("common.notification.createFailed"));
  } finally {
    loading.value = false;
  }
}

// ProModal 关闭时自动重置表单
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
</style>
