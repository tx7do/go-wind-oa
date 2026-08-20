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
          placeholder='[{"approvers":[{"type":"USER","id":123},{"type":"LEADER"}],"strategy":"ALL"}] strategy=ALL 会签(默认)/ANY 或签；type=USER 指定用户/LEADER 申请人主管/POSITION 职位持有者'
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
