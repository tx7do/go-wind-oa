<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ElCard shadow="never" class="flex-1">
      <ElForm label-width="100px">
        <ElFormItem :label="$t('pages.oa.announcement.fieldTitle')">
          <ElInput
            v-model="form.title"
            :placeholder="$t('pages.oa.announcement.placeholderTitle')"
          />
        </ElFormItem>
        <ElFormItem :label="$t('pages.oa.announcement.fieldContent')">
          <ElInput
            v-model="form.content"
            type="textarea"
            :rows="6"
            :placeholder="$t('pages.oa.announcement.placeholderContent')"
          />
        </ElFormItem>
        <ElFormItem :label="$t('pages.oa.announcement.fieldScope')">
          <ElRadioGroup v-model="form.scope">
            <ElRadio value="all">{{ $t("pages.oa.announcement.scopeAll") }}</ElRadio>
            <ElRadio value="dept">{{ $t("pages.oa.announcement.scopeDept") }}</ElRadio>
          </ElRadioGroup>
        </ElFormItem>
        <ElFormItem
          v-if="form.scope === 'dept'"
          :label="$t('pages.oa.announcement.fieldSelectDept')"
        >
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
            <div v-if="orgTree.length === 0" class="empty">
              {{ $t("pages.oa.announcement.noOrgData") }}
            </div>
          </div>
        </ElFormItem>
        <ElFormItem>
          <ElButton type="primary" :loading="sending" @click="send">
            {{ $t("pages.oa.announcement.publish") }}
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
import { $t } from "@/core/i18n";

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
    ElMessage.warning($t("pages.oa.announcement.titleContentRequired"));
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
        ElMessage.warning($t("pages.oa.announcement.deptRequired"));
        sending.value = false;
        return;
      }
      const resp = await apiClient.userService.ListUserIDsByOrgUnitIDs({
        orgUnitIds: checked,
        excludeExpired: true,
      } as any);
      const userIds = (resp as any)?.userIds ?? [];
      if (userIds.length === 0) {
        ElMessage.warning($t("pages.oa.announcement.deptNoMembers"));
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
    ElMessage.success($t("pages.oa.announcement.publishSuccess"));
    form.title = "";
    form.content = "";
  } catch (e: any) {
    ElMessage.error(e?.message || $t("pages.oa.announcement.publishFailed"));
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
