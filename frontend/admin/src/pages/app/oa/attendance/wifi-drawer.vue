<template>
  <ProModal
    v-model:visible="visible"
    title="新建 Wi-Fi 指纹"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <ElForm label-width="90px">
      <ElFormItem label="SSID">
        <ElInput v-model="form.ssid" />
      </ElFormItem>
      <ElFormItem label="BSSID">
        <ElInput v-model="form.bssid" />
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
import { reactive, ref } from "vue";
import ProModal from "@/components/Pro/ProModal/index.vue";
import { useUpsertWifiFingerprint } from "@/api/composables";
import { DRAWER_WIDTH } from "@/constants";

const emit = defineEmits(["success"]);

const visible = ref(false);
const saving = ref(false);
const form = reactive({ ssid: "", bssid: "" });

const upsertMutation = useUpsertWifiFingerprint({
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
    { ssid: form.ssid, bssid: form.bssid },
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
