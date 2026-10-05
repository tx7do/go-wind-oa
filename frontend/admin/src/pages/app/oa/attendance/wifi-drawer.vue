<template>
  <ProModal
    v-model:visible="visible"
    :title="$t('pages.oa.attendance.wifiDrawer.createTitle')"
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
import { reactive, ref } from "vue";
import ProModal from "@/components/Pro/ProModal/index.vue";
import { useUpsertWifiFingerprint } from "@/api/composables";
import { $t } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";
import { useOaDrawer } from "../use-oa-drawer";

const emit = defineEmits(["success"]);

const { visible, open, close } = useOaDrawer();
const saving = ref(false);
const form = reactive({ ssid: "", bssid: "" });

const upsertMutation = useUpsertWifiFingerprint({
  onSuccess: () => {
    ElMessage.success($t("common.notification.saveSuccess"));
    close();
    emit("success");
  },
  onError: (err: Error) => ElMessage.error(err.message || $t("common.notification.saveFailed")),
});

const handleClose = close;

function save() {
  saving.value = true;
  upsertMutation.mutate(
    { ssid: form.ssid, bssid: form.bssid },
    { onSettled: () => (saving.value = false) }
  );
}

defineExpose({ open });
</script>
