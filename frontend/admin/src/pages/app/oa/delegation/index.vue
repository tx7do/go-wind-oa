<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd">
      <template #operation="scope: any">
        <ElButton size="small" type="danger" link @click="handleDelete(scope.row)">
          {{ $t("common.delete") }}
        </ElButton>
      </template>
    </ProPage>

    <DelegationDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElButton, ElMessage, ElMessageBox } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import DelegationDrawer from "./delegation-drawer.vue";
import {
  fetchListWorkflowDelegations,
  useDeleteWorkflowDelegation,
  fetchUsers,
  userDisplayName,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { queryClient } from "@/plugins/vue-query";
import { $t } from "@/core/i18n";
import type { identityservicev1_User } from "@/api/generated/admin/service/v1";

const pageRef = ref();
const drawerRef = ref();

const deleteMutation = useDeleteWorkflowDelegation({
  onSuccess: () => {
    ElMessage.success($t("common.notification.deleteSuccess"));
    queryClient.invalidateQueries({ queryKey: ["listWorkflowDelegations"] });
    pageRef.value?.refresh();
  },
  onError: (err: Error) => ElMessage.error(err.message || $t("common.notification.deleteFailed")),
});

const userMap = ref<Map<number, identityservicev1_User>>(new Map());

async function loadUsers() {
  try {
    const resp = await fetchUsers();
    const m = new Map<number, identityservicev1_User>();
    for (const u of resp.items ?? []) {
      m.set(u.id as number, u);
    }
    userMap.value = m;
  } catch {
    userMap.value = new Map();
  }
}

loadUsers();

function fmtTime(v?: string) {
  return v ? String(v).replace("T", " ").slice(0, 19) : "-";
}

function handleAdd() {
  drawerRef.value?.open();
}

function handleSuccess() {
  pageRef.value?.refresh();
}

function handleDelete(row: any) {
  ElMessageBox.confirm("确定删除此委托记录？", "删除确认", {
    confirmButtonText: $t("common.button.confirm"),
    cancelButtonText: $t("common.cancel"),
    type: "warning",
  })
    .then(() => {
      deleteMutation.mutate({ id: row.id as number });
    })
    .catch(() => {});
}

const pageConfig = computed<ProPageConfig>(() => ({
  table: {
    listAction: async () => {
      const result = await fetchListWorkflowDelegations(new PaginationQuery());
      return { items: (result as any)?.items ?? [], total: 0 };
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    pagination: false,
    tableAttrs: { border: true, stripe: true },
    columns: [
      { prop: "id", label: "ID", width: 80 },
      {
        prop: "delegatorUserId",
        label: "委托人",
        width: 200,
        formatter: (row: any) => {
          const u = userMap.value.get(row.delegatorUserId as number);
          return u ? userDisplayName(u) : `#${row.delegatorUserId}`;
        },
      },
      {
        prop: "delegateUserId",
        label: "被委托人（代理人）",
        width: 200,
        formatter: (row: any) => {
          const u = userMap.value.get(row.delegateUserId as number);
          return u ? userDisplayName(u) : `#${row.delegateUserId}`;
        },
      },
      {
        prop: "createdAt",
        label: "创建时间",
        width: 170,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "operation",
        label: "操作",
        width: 100,
        slotName: "operation",
      },
    ],
  },
}));
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
