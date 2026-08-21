<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ElCard shadow="never" class="flex-1 directory-card">
      <div class="directory-layout">
        <div class="tree-panel">
          <div class="panel-title">组织架构</div>
          <ElTree
            :data="orgTree"
            :props="{ label: 'name', children: 'children' }"
            node-key="id"
            default-expand-all
            highlight-current
            @node-click="onNodeClick"
          />
        </div>
        <div class="member-panel">
          <div class="panel-title">成员列表</div>
          <ElTable :data="members" border stripe v-loading="loading">
            <ElTableColumn prop="id" label="ID" width="70" />
            <ElTableColumn prop="nickname" label="昵称" width="120" />
            <ElTableColumn prop="realname" label="姓名" width="120" />
            <ElTableColumn label="部门" min-width="180">
              <template #default="{ row }">
                {{ joinArr(row.orgUnitNames) }}
              </template>
            </ElTableColumn>
            <ElTableColumn label="职位" min-width="180">
              <template #default="{ row }">
                {{ joinArr(row.positionNames) }}
              </template>
            </ElTableColumn>
          </ElTable>
        </div>
      </div>
    </ElCard>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted } from "vue";
import { ElCard, ElTable, ElTableColumn, ElTree } from "element-plus";
import { apiClient } from "@/api/client";

const orgTree = ref<any[]>([]);
const members = ref<any[]>([]);
const loading = ref(false);

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
  await loadMembers();
});

async function loadMembers() {
  loading.value = true;
  try {
    const resp = await apiClient.userService.List({
      page: 1,
      pageSize: 999,
      noPaging: true,
      sorting: undefined,
    } as any);
    members.value = (resp as any)?.items ?? [];
  } catch {
    members.value = [];
  } finally {
    loading.value = false;
  }
}

function onNodeClick(_data: any) {
  // 组织树当前仅作导航展示，成员列表展示全租户用户（含 orgUnitNames 标注所属部门）。
}

function joinArr(arr: any): string {
  if (!arr || !Array.isArray(arr)) return "";
  return (arr as string[]).join("、");
}
</script>

<style scoped>
.directory-card {
  height: 100%;
}
.directory-layout {
  display: flex;
  gap: 12px;
  height: 100%;
  min-height: 0;
}
.tree-panel {
  width: 280px;
  flex-shrink: 0;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  overflow: auto;
  padding: 8px;
}
.member-panel {
  flex: 1;
  min-width: 0;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  overflow: auto;
  padding: 8px;
}
.panel-title {
  font-weight: 600;
  padding: 4px 8px 8px;
  color: var(--el-text-color-primary);
}
</style>
