<template>
  <ProModal
    v-model:visible="visible"
    :title="title"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <ElForm ref="formRef" :model="formData" :rules="formRules" label-width="140px">
      <ElFormItem :label="$t('pages.oa.attendance.fieldName')" prop="name">
        <ElInput v-model="formData.name" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.oa.attendance.fieldLongitude')" prop="longitude">
        <ElInputNumber
          v-model="formData.longitude"
          controls-position="right"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.oa.attendance.fieldLatitude')" prop="latitude">
        <ElInputNumber
          v-model="formData.latitude"
          controls-position="right"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.oa.attendance.fieldRadius')" prop="radius">
        <ElInputNumber
          v-model="formData.radius"
          :min="1"
          controls-position="right"
          style="width: 100%"
        />
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
import type { oaservicev1_AttendanceFence } from "@/api/generated/admin/service/v1";

import {
  useCreateAttendanceFence,
  useUpdateAttendanceFence,
} from "@/api/composables";
import { $t } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";
import ProModal from "@/components/Pro/ProModal/index.vue";

const emit = defineEmits(["success"]);

const { mutateAsync: createFence } = useCreateAttendanceFence();
const { mutateAsync: updateFence } = useUpdateAttendanceFence();

const visible = ref(false);
const loading = ref(false);
const formRef = ref<FormInstance>();
const editingId = ref<number | null>(null);

const formData = reactive({
  name: "",
  longitude: 0,
  latitude: 0,
  radius: 1,
});

const formRules: FormRules = {
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  longitude: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  latitude: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  radius: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
};

const title = computed(() =>
  editingId.value === null
    ? $t("pages.oa.attendance.createFence")
    : $t("pages.oa.attendance.editFence")
);

function resetForm() {
  formData.name = "";
  formData.longitude = 0;
  formData.latitude = 0;
  formData.radius = 1;
  editingId.value = null;
  formRef.value?.clearValidate();
}

function open(row?: oaservicev1_AttendanceFence) {
  visible.value = true;
  resetForm();
  if (row) {
    editingId.value = (row.id as number) ?? null;
    formData.name = row.name ?? "";
    formData.longitude = (row.longitude as number) ?? 0;
    formData.latitude = (row.latitude as number) ?? 0;
    formData.radius = (row.radius as number) ?? 1;
  }
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
    if (editingId.value === null) {
      await createFence({ data: formData });
      ElMessage.success($t("common.notification.createSuccess"));
    } else {
      await updateFence({
        id: editingId.value,
        data: formData,
        updateMask: "name,longitude,latitude,radius",
      });
      ElMessage.success($t("common.notification.updateSuccess"));
    }
    emit("success");
    handleClose();
  } catch {
    ElMessage.error(
      editingId.value === null
        ? $t("common.notification.createFailed")
        : $t("common.notification.updateFailed")
    );
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
