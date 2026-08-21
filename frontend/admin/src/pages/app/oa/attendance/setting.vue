<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ElCard shadow="never" class="flex-1">
      <div class="hint">
        工时设置每租户一行，用于考勤结算判定迟到/早退。默认 09:00–18:00。
      </div>
      <ElForm label-width="120px" class="setting-form">
        <ElFormItem label="上班时间">
          <ElTimePicker
            v-model="form.workStartTime"
            value-format="HH:mm"
            format="HH:mm"
            :clearable="false"
            style="width: 200px"
          />
        </ElFormItem>
        <ElFormItem label="下班时间">
          <ElTimePicker
            v-model="form.workEndTime"
            value-format="HH:mm"
            format="HH:mm"
            :clearable="false"
            style="width: 200px"
          />
        </ElFormItem>
        <ElFormItem>
          <ElButton type="primary" :loading="saving" @click="save">保存</ElButton>
        </ElFormItem>
      </ElForm>
    </ElCard>
  </div>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref } from "vue";
import { ElButton, ElCard, ElForm, ElFormItem, ElMessage, ElTimePicker } from "element-plus";

import { useGetAttendanceSetting, useUpdateAttendanceSetting } from "@/api/composables";

const form = reactive({ workStartTime: "09:00", workEndTime: "18:00" });
const saving = ref(false);
const loading = ref(false);

const getQuery = useGetAttendanceSetting({ enabled: false });

const updateMutation = useUpdateAttendanceSetting({
  onSuccess: () => ElMessage.success("已保存"),
  onError: (err: Error) => ElMessage.error(err.message || "保存失败"),
});

async function load() {
  loading.value = true;
  try {
    const resp = await getQuery.refetch();
    const data = resp.data as { workStartTime?: string; workEndTime?: string } | undefined;
    if (data?.workStartTime) form.workStartTime = data.workStartTime;
    if (data?.workEndTime) form.workEndTime = data.workEndTime;
  } finally {
    loading.value = false;
  }
}

function save() {
  saving.value = true;
  updateMutation.mutate(
    { workStartTime: form.workStartTime, workEndTime: form.workEndTime },
    { onSettled: () => (saving.value = false) }
  );
}

onMounted(load);
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
.hint {
  margin-bottom: 16px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.setting-form {
  max-width: 480px;
}
</style>
