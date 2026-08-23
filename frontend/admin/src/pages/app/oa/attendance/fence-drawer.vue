<template>
  <ProModal
    v-model:visible="visible"
    title="新建围栏"
    :config="{ component: 'drawer', drawer: { size: WIDE_DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <ElForm label-width="90px">
      <ElFormItem label="名称">
        <ElInput v-model="form.name" />
      </ElFormItem>
      <ElFormItem label="位置">
        <AmapCirclePicker v-model="geoValue" />
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
import { ElButton, ElForm, ElFormItem, ElInput, ElMessage } from "element-plus";
import { reactive, ref, watch } from "vue";
import ProModal from "@/components/Pro/ProModal/index.vue";
import AmapCirclePicker from "@/components/AmapCirclePicker/index.vue";
import { useUpsertGeofence } from "@/api/composables";

// 围栏抽屉含地图，需比通用抽屉更宽；不改全局 DRAWER_WIDTH 以免影响其他抽屉。
const WIDE_DRAWER_WIDTH = "720px";

const emit = defineEmits(["success"]);

const visible = ref(false);
const saving = ref(false);
const form = reactive({ name: "", latitude: 0, longitude: 0, radiusMeters: 100 });

// 地图组件的双向绑定值。center 为 [经度, 纬度]，与表单的 longitude/latitude 对应。
const geoValue = ref<{ center: [number, number] | null; radius: number }>({
  center: null,
  radius: 100,
});

// 地图圈选 → 回填表单字段。
watch(
  geoValue,
  (v) => {
    if (v.center) {
      form.longitude = v.center[0];
      form.latitude = v.center[1];
    }
    form.radiusMeters = v.radius;
  },
  { deep: true }
);

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
  if (!form.name) {
    ElMessage.warning("请填写围栏名称");
    return;
  }
  if (!form.latitude || !form.longitude) {
    ElMessage.warning("请在地图上圈选围栏位置");
    return;
  }
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
