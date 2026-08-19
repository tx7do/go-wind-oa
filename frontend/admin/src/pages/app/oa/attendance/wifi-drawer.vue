<template>
  <ProModal
    v-model:visible="visible"
    :title="title"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <ElForm ref="formRef" :model="formData" :rules="formRules" label-width="140px">
      <ElFormItem :label="$t('pages.oa.attendance.fieldSsid')" prop="ssid">
        <ElInput v-model="formData.ssid" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.oa.attendance.fieldBssid')" prop="bssid">
        <ElInput v-model="formData.bssid" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="handleClose">{{ $t("pages.oa.attendance.cancel") }}</ElButton>
        <ElButton type="primary" :loading="loading" @click="handleSubmit">
          {{ $t("pages.oa.attendance.submit") }}
        </ElButton>
      </div>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import { ElMessage, type FormInstance, type FormRules } from "element-plus";
import { ref, reactive, computed, watch } from "vue";

import { useCreateAttendanceWifi } from "@/api/composables";
import { $t } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";
import ProModal from "@/components/Pro/ProModal/index.vue";

const emit = defineEmits(["success"]);

const { mutateAsync: createWifi } = useCreateAttendanceWifi();

const visible = ref(false);
const loading = ref(false);
const formRef = ref<FormInstance>();

const formData = reactive({
  ssid: "",
  bssid: "",
});

const formRules: FormRules = {
  ssid: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  bssid: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
};

const title = computed(() => $t("pages.oa.attendance.createWifi"));

function resetForm() {
  formData.ssid = "";
  formData.bssid = "";
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

async function handleSubmit() {
  if (!formRef.value) return;

  const valid = await formRef.value.validate().then(
    () => true,
    () => false
  );
  if (!valid) return;

  try {
    loading.value = true;
    await createWifi({ data: formData });
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
</style>
