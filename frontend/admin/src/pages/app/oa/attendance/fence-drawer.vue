<template>
  <ProModal
    v-model:visible="visible"
    title="新建围栏"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <ElForm label-width="90px">
      <ElFormItem label="名称">
        <ElInput v-model="form.name" />
      </ElFormItem>
      <ElFormItem label="纬度">
        <ElInputNumber v-model="form.latitude" :precision="6" :step="0.000001" :controls="false" style="width: 100%" />
      </ElFormItem>
      <ElFormItem label="经度">
        <ElInputNumber v-model="form.longitude" :precision="6" :step="0.000001" :controls="false" style="width: 100%" />
      </ElFormItem>
      <ElFormItem label="半径(米)">
        <ElInputNumber v-model="form.radiusMeters" :min="1" :precision="0" :step="1" :controls="false" style="width: 100%" />
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
import { ElButton, ElForm, ElFormItem, ElInput, ElInputNumber, ElMessage } from "element-plus";
import { reactive, ref } from "vue";
import ProModal from "@/components/Pro/ProModal/index.vue";
import { useUpsertGeofence } from "@/api/composables";
import { DRAWER_WIDTH } from "@/constants";

const emit = defineEmits(["success"]);

const visible = ref(false);
const saving = ref(false);
const form = reactive({ name: "", latitude: 0, longitude: 0, radiusMeters: 100 });

const upsertMutation = useUpsertGeofence({
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
  saving.value = true;
  upsertMutation.mutate(
    { name: form.name, latitude: form.latitude, longitude: form.longitude, radiusMeters: form.radiusMeters },
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
