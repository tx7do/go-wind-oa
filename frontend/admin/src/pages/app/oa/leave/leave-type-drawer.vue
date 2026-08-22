<template>
  <ProModal
    v-model:visible="visible"
    title="新建请假类型"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <ElForm label-width="80px">
      <ElFormItem label="代码">
        <ElInput v-model="form.code" placeholder="ANNUAL / SICK / PERSONAL" />
      </ElFormItem>
      <ElFormItem label="名称">
        <ElInput v-model="form.name" placeholder="年假 / 病假 / 事假" />
      </ElFormItem>
    </ElForm>
    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="handleClose">取消</ElButton>
        <ElButton type="primary" :loading="creating" @click="submit">创建</ElButton>
      </div>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import { ElButton, ElForm, ElFormItem, ElInput, ElMessage } from "element-plus";
import { reactive, ref } from "vue";
import ProModal from "@/components/Pro/ProModal/index.vue";
import { useCreateLeaveType } from "@/api/composables";
import { DRAWER_WIDTH } from "@/constants";

const emit = defineEmits(["success"]);

const visible = ref(false);
const creating = ref(false);
const form = reactive({ code: "", name: "" });

const createMutation = useCreateLeaveType({
  onSuccess: () => {
    ElMessage.success("已创建");
    visible.value = false;
    emit("success");
  },
  onError: (err: Error) => ElMessage.error(err.message || "创建失败"),
});

function open() {
  visible.value = true;
}

function handleClose() {
  visible.value = false;
}

function submit() {
  if (!form.code || !form.name) {
    ElMessage.warning("请填写代码与名称");
    return;
  }
  creating.value = true;
  createMutation.mutate(
    { data: { code: form.code, name: form.name } },
    { onSettled: () => (creating.value = false) }
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
