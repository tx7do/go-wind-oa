<template>
  <ProModal
    v-model:visible="visible"
    title="授予假期额度"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <ElForm label-width="90px">
      <ElFormItem label="用户ID">
        <ElInputNumber v-model="form.userId" :min="1" style="width: 100%" />
      </ElFormItem>
      <ElFormItem label="类型ID">
        <ElInputNumber v-model="form.leaveTypeId" :min="1" style="width: 100%" />
      </ElFormItem>
      <ElFormItem label="年度">
        <ElInputNumber v-model="form.year" :min="2000" :max="2100" style="width: 100%" />
      </ElFormItem>
      <ElFormItem label="总额度(天)">
        <ElInputNumber v-model="form.totalDays" :min="0" :step="0.5" style="width: 100%" />
      </ElFormItem>
    </ElForm>
    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="handleClose">取消</ElButton>
        <ElButton type="primary" :loading="granting" @click="submit">授予</ElButton>
      </div>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import { ElButton, ElForm, ElFormItem, ElInputNumber, ElMessage } from "element-plus";
import { reactive, ref } from "vue";
import ProModal from "@/components/Pro/ProModal/index.vue";
import { useGrantLeaveBalance } from "@/api/composables";
import { DRAWER_WIDTH } from "@/constants";

const emit = defineEmits(["success"]);

const visible = ref(false);
const granting = ref(false);
const form = reactive({
  userId: 1,
  leaveTypeId: 1,
  year: new Date().getFullYear(),
  totalDays: 10,
});

const grantMutation = useGrantLeaveBalance({
  onSuccess: () => {
    ElMessage.success("已授予");
    visible.value = false;
    emit("success");
  },
  onError: (err: Error) => ElMessage.error(err.message || "授予失败"),
});

function open() {
  visible.value = true;
}

function handleClose() {
  visible.value = false;
}

function submit() {
  granting.value = true;
  grantMutation.mutate(
    {
      userId: form.userId,
      leaveTypeId: form.leaveTypeId,
      year: form.year,
      totalDays: form.totalDays,
    },
    { onSettled: () => (granting.value = false) }
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
