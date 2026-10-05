<template>
  <ProModal
    v-model:visible="visible"
    :title="$t('pages.oa.attendance.createFence')"
    :config="{ component: 'drawer', drawer: { size: WIDE_DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <ElForm label-width="90px">
      <ElFormItem :label="$t('pages.oa.attendance.fieldName')">
        <ElInput v-model="form.name" />
      </ElFormItem>
      <ElFormItem :label="$t('pages.oa.attendance.fenceDrawer.fieldLocation')">
        <AmapCirclePicker v-model="geoValue" />
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
import { ElButton, ElForm, ElFormItem, ElInput, ElMessage } from "element-plus";
import { reactive, ref, watch } from "vue";
import ProModal from "@/components/Pro/ProModal/index.vue";
import AmapCirclePicker from "@/components/AmapCirclePicker/index.vue";
import { useUpsertGeofence } from "@/api/composables";
import { $t } from "@/core/i18n";
import { useOaDrawer } from "../use-oa-drawer";

// 围栏抽屉含地图，需比通用抽屉更宽；不改全局 DRAWER_WIDTH 以免影响其他抽屉。
const WIDE_DRAWER_WIDTH = "720px";

const emit = defineEmits(["success"]);

const { visible, open, close } = useOaDrawer();
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
    ElMessage.success($t("common.notification.saveSuccess"));
    close();
    emit("success");
  },
  onError: (err: Error) => ElMessage.error(err.message || $t("common.notification.saveFailed")),
});

const handleClose = close;

function save() {
  if (!form.name) {
    ElMessage.warning($t("pages.oa.attendance.fenceDrawer.nameRequired"));
    return;
  }
  if (!form.latitude || !form.longitude) {
    ElMessage.warning($t("pages.oa.attendance.fenceDrawer.locationRequired"));
    return;
  }
  saving.value = true;
  upsertMutation.mutate(
    {
      name: form.name,
      latitude: form.latitude,
      longitude: form.longitude,
      radiusMeters: form.radiusMeters,
    },
    { onSettled: () => (saving.value = false) }
  );
}

defineExpose({ open });
</script>
