<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ElCard shadow="never" class="flex-1">
      <ElForm label-width="100px">
        <ElFormItem label="标题">
          <ElInput v-model="form.title" placeholder="公告标题" />
        </ElFormItem>
        <ElFormItem label="内容">
          <ElInput
            v-model="form.content"
            type="textarea"
            :rows="6"
            placeholder="公告内容"
          />
        </ElFormItem>
        <ElFormItem label="发布范围">
          <ElRadioGroup v-model="form.scope">
            <ElRadio value="all">全员</ElRadio>
            <ElRadio value="dept">按部门</ElRadio>
          </ElRadioGroup>
        </ElFormItem>
        <ElFormItem v-if="form.scope === 'dept'" label="选择部门">
          <div class="tree-wrap">
            <ElTree
              ref="treeRef"
              :data="orgTree"
              :props="{ label: 'name', children: 'children' }"
              node-key="id"
              show-checkbox
              check-strictly
              default-expand-all
            />
            <div v-if="orgTree.length === 0" class="empty">暂无组织数据</div>
          </div>
        </ElFormItem>
        <ElFormItem>
          <ElButton type="primary" :loading="sending" @click="send">
            发布公告
          </ElButton>
        </ElFormItem>
      </ElForm>
    </ElCard>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, onMounted } from "vue";
import {
  ElButton,
  ElCard,
  ElForm,
  ElFormItem,
  ElInput,
  ElRadio,
  ElRadioGroup,
  ElTree,
  ElMessage,
} from "element-plus";
import { apiClient } from "@/api/client";

const form = reactive({
  title: "",
  content: "",
  scope: "all" as "all" | "dept",
});
const orgTree = ref<any[]>([]);
const treeRef = ref<InstanceType<typeof ElTree>>();
const sending = ref(false);

onMounted(async () => {
  try {
    const resp = await apiClient.orgUnitService.List({
      page: 1,
      pageSize: 999,
      noPaging: true,
      sorting: undefined,
    } as any);
    orgTree.value = (resp as any)?.items ?? [];
  } catch {
    orgTree.value = [];
  }
});

async function send() {
  if (!form.title || !form.content) {
    ElMessage.warning("请填写标题与内容");
    return;
  }
  sending.value = true;
  try {
    if (form.scope === "all") {
      await apiClient.internalMessageService.SendMessage({
        type: "NOTIFICATION" as any,
        title: form.title,
        content: form.content,
        targetAll: true,
        targetUserIds: undefined,
      } as any);
    } else {
      const checked = treeRef.value?.getCheckedKeys(false) as number[];
      if (!checked || checked.length === 0) {
        ElMessage.warning("请至少选择一个部门");
        sending.value = false;
        return;
      }
      const resp = await apiClient.userService.ListUserIDsByOrgUnitIDs({
        orgUnitIds: checked,
        excludeExpired: true,
      } as any);
      const userIds = (resp as any)?.userIds ?? [];
      if (userIds.length === 0) {
        ElMessage.warning("所选部门无在職成员");
        sending.value = false;
        return;
      }
      await apiClient.internalMessageService.SendMessage({
        type: "NOTIFICATION" as any,
        title: form.title,
        content: form.content,
        targetAll: false,
        targetUserIds: userIds,
      } as any);
    }
    ElMessage.success("公告已发布");
    form.title = "";
    form.content = "";
  } catch (e: any) {
    ElMessage.error(e?.message || "发布失败");
  } finally {
    sending.value = false;
  }
}
</script>

<style scoped>
.tree-wrap {
  width: 100%;
  max-height: 300px;
  overflow: auto;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  padding: 8px;
}
.empty {
  padding: 16px;
  color: var(--el-text-color-secondary);
  text-align: center;
}
</style>
