<template>
  <ProModal
    v-model:visible="visible"
    :title="$t('pages.oa.attendance.holidayDrawer.title')"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <ElForm label-width="90px">
      <ElFormItem :label="$t('pages.oa.attendance.holidayDrawer.fieldDate')">
        <ElDatePicker
          v-model="form.date"
          type="date"
          value-format="YYYY-MM-DD"
          style="width: 100%"
        />
      </ElFormItem>
      <ElFormItem :label="$t('pages.oa.attendance.holidayDrawer.fieldType')">
        <ElRadioGroup v-model="form.holidayType">
          <ElRadio value="HOLIDAY">
            {{ $t("pages.oa.attendance.holidayDrawer.statutoryHoliday") }}
          </ElRadio>
          <ElRadio value="WORKDAY">
            {{ $t("pages.oa.attendance.holidayDrawer.makeupWorkday") }}
          </ElRadio>
        </ElRadioGroup>
      </ElFormItem>
      <ElFormItem :label="$t('pages.oa.attendance.holidayDrawer.fieldName')">
        <ElInput
          v-model="form.name"
          :placeholder="$t('pages.oa.attendance.holidayDrawer.namePlaceholder')"
        />
      </ElFormItem>
    </ElForm>
    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="handleClose">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="saving" @click="save">
          {{ $t("common.button.save") }}
        </ElButton>
      </div>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import {
  ElButton,
  ElForm,
  ElFormItem,
  ElInput,
  ElDatePicker,
  ElMessage,
  ElRadio,
  ElRadioGroup,
} from "element-plus";
import { reactive, ref } from "vue";
import ProModal from "@/components/Pro/ProModal/index.vue";
import { useUpsertHoliday } from "@/api/composables";
import type { oaservicev1_Holiday_HolidayType } from "@/api/generated/admin/service/v1";
import { DRAWER_WIDTH } from "@/constants";
import { $t } from "@/core/i18n";

const emit = defineEmits(["success"]);

const visible = ref(false);
const saving = ref(false);
const form = reactive({ date: "", holidayType: "HOLIDAY", name: "" });

const upsertMutation = useUpsertHoliday({
  onSuccess: () => {
    ElMessage.success($t("common.notification.saveSuccess"));
    visible.value = false;
    emit("success");
  },
  onError: (err: Error) => ElMessage.error(err.message || $t("common.notification.saveFailed")),
});

function open() {
  visible.value = true;
}

function handleClose() {
  visible.value = false;
}

function save() {
  if (!form.date) {
    ElMessage.warning($t("pages.oa.attendance.holidayDrawer.dateRequired"));
    return;
  }
  saving.value = true;
  upsertMutation.mutate(
    {
      date: `${form.date}T00:00:00Z`,
      holidayType: form.holidayType as oaservicev1_Holiday_HolidayType,
      name: form.name,
    },
    { onSettled: () => (saving.value = false) }
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
</style>
