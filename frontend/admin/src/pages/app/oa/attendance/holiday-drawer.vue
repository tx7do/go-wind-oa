<template>
  <ProModal
    v-model:visible="visible"
    title="设置节假日/调休"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <ElForm label-width="90px">
      <ElFormItem label="日期">
        <ElDatePicker v-model="form.date" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
      </ElFormItem>
      <ElFormItem label="类型">
        <ElRadioGroup v-model="form.holidayType">
          <ElRadio value="HOLIDAY">法定假日（休息）</ElRadio>
          <ElRadio value="WORKDAY">调休上班</ElRadio>
        </ElRadioGroup>
      </ElFormItem>
      <ElFormItem label="名称">
        <ElInput v-model="form.name" placeholder="如：国庆节 / 调休上班" />
      </ElFormItem>
    </ElForm>
    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="handleClose">取消</ElButton>
        <ElButton type="primary" :loading="saving" @click="save">保存</ElButton>
      </div>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import { ElButton, ElForm, ElFormItem, ElInput, ElDatePicker, ElMessage, ElRadio, ElRadioGroup } from "element-plus";
import { reactive, ref } from "vue";
import ProModal from "@/components/Pro/ProModal/index.vue";
import { useUpsertHoliday } from "@/api/composables";
import type { oaservicev1_Holiday_HolidayType } from "@/api/generated/admin/service/v1";
import { DRAWER_WIDTH } from "@/constants";

const emit = defineEmits(["success"]);

const visible = ref(false);
const saving = ref(false);
const form = reactive({ date: "", holidayType: "HOLIDAY", name: "" });

const upsertMutation = useUpsertHoliday({
  onSuccess: () => {
    ElMessage.success("已保存");
    visible.value = false;
    emit("success");
  },
  onError: (err: Error) => ElMessage.error(err.message || "保存失败"),
});

function open() {
  visible.value = true;
}

function handleClose() {
  visible.value = false;
}

function save() {
  if (!form.date) {
    ElMessage.warning("请选择日期");
    return;
  }
  saving.value = true;
  upsertMutation.mutate(
    { date: `${form.date}T00:00:00Z`, holidayType: form.holidayType as oaservicev1_Holiday_HolidayType, name: form.name },
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
