<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ElCard shadow="never" class="flex-1">
      <ElTabs v-model="tab">
        <ElTabPane label="请假类型与额度" name="types">
          <div class="flex gap-3 mb-4">
            <ElButton type="primary" @click="typeDialogVisible = true">新建类型</ElButton>
            <ElButton type="primary" plain @click="balanceDialogVisible = true">授予额度</ElButton>
          </div>
          <ElTable :data="leaveTypes" border stripe v-loading="loadingTypes">
            <ElTableColumn prop="id" label="ID" width="80" />
            <ElTableColumn prop="code" label="代码" width="160" />
            <ElTableColumn prop="name" label="名称" width="200" />
            <ElTableColumn prop="remark" label="备注" min-width="200" />
            <ElTableColumn prop="created_at" label="创建时间" width="170">
              <template #default="{ row }">{{ fmtTime(row.createdAt || row.created_at) }}</template>
            </ElTableColumn>
          </ElTable>

          <div class="subtitle mt-6 mb-2">假期额度（全部用户，当年）</div>
          <ElTable :data="balances" border stripe v-loading="loadingBalances">
            <ElTableColumn prop="userId" label="用户ID" width="100" />
            <ElTableColumn prop="leaveTypeId" label="类型ID" width="100" />
            <ElTableColumn prop="year" label="年度" width="100" />
            <ElTableColumn prop="totalDays" label="总额度(天)" width="120" />
            <ElTableColumn prop="usedDays" label="已用(天)" width="120" />
          </ElTable>
        </ElTabPane>

        <ElTabPane label="请假申请" name="applications">
          <ElTable :data="applications" border stripe v-loading="loadingApps">
            <ElTableColumn prop="id" label="ID" width="80" />
            <ElTableColumn prop="createdBy" label="申请人ID" width="100" />
            <ElTableColumn prop="leaveTypeName" label="类型" width="120" />
            <ElTableColumn label="起止" min-width="200">
              <template #default="{ row }">
                {{ fmtDate(row.startDate) }} ~ {{ fmtDate(row.endDate) }}
              </template>
            </ElTableColumn>
            <ElTableColumn prop="days" label="天数" width="80" />
            <ElTableColumn prop="reason" label="事由" min-width="160" show-overflow-tooltip />
            <ElTableColumn label="状态" width="100">
              <template #default="{ row }">
                <ElTag :type="statusTag(row.leaveStatus)">{{ statusLabel(row.leaveStatus) }}</ElTag>
              </template>
            </ElTableColumn>
            <ElTableColumn prop="instanceId" label="流程实例" width="100" />
          </ElTable>
          <div class="pager">
            <ElPagination
              v-model:current-page="appPage"
              v-model:page-size="appPageSize"
              :total="appTotal"
              :page-sizes="[10, 20, 50]"
              layout="total, sizes, prev, pager, next"
              @current-change="loadAll"
              @size-change="loadAll"
            />
          </div>
        </ElTabPane>
      </ElTabs>
    </ElCard>

    <ElDialog v-model="typeDialogVisible" title="新建请假类型" width="420px">
      <ElForm label-width="80px">
        <ElFormItem label="代码">
          <ElInput v-model="newType.code" placeholder="ANNUAL / SICK / PERSONAL" />
        </ElFormItem>
        <ElFormItem label="名称">
          <ElInput v-model="newType.name" placeholder="年假 / 病假 / 事假" />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="typeDialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="creatingType" @click="createType">创建</ElButton>
      </template>
    </ElDialog>

    <ElDialog v-model="balanceDialogVisible" title="授予假期额度" width="420px">
      <ElForm label-width="90px">
        <ElFormItem label="用户ID">
          <ElInputNumber v-model="newBalance.userId" :min="1" style="width: 100%" />
        </ElFormItem>
        <ElFormItem label="类型ID">
          <ElInputNumber v-model="newBalance.leaveTypeId" :min="1" style="width: 100%" />
        </ElFormItem>
        <ElFormItem label="年度">
          <ElInputNumber v-model="newBalance.year" :min="2000" :max="2100" style="width: 100%" />
        </ElFormItem>
        <ElFormItem label="总额度(天)">
          <ElInputNumber v-model="newBalance.totalDays" :min="0" :step="0.5" style="width: 100%" />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="balanceDialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="granting" @click="grantBalance">授予</ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref } from "vue";
import {
  ElButton,
  ElCard,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElMessage,
  ElPagination,
  ElTable,
  ElTableColumn,
  ElTabPane,
  ElTabs,
  ElTag,
} from "element-plus";

import {
  useCreateLeaveType,
  useGrantLeaveBalance,
  useListLeaveApplications,
  useListLeaveBalances,
  useListLeaveTypes,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";

const tab = ref("types");

const leaveTypes = ref<any[]>([]);
const loadingTypes = ref(false);
const typesQuery = useListLeaveTypes(new PaginationQuery(), { enabled: false });

const balances = ref<any[]>([]);
const loadingBalances = ref(false);
const balancesQuery = useListLeaveBalances(reactive({ userId: 0, year: 0 }), {
  enabled: false,
});

const applications = ref<any[]>([]);
const loadingApps = ref(false);
const appPage = ref(1);
const appPageSize = ref(10);
const appTotal = ref(0);
const appsQuery = useListLeaveApplications(
  reactive({
    userId: 0,
    status: undefined,
    page: computed(() => appPage.value),
    pageSize: computed(() => appPageSize.value),
  }),
  { enabled: false }
);

async function loadAll() {
  loadingTypes.value = true;
  loadingBalances.value = true;
  loadingApps.value = true;
  try {
    const t = await typesQuery.refetch();
    leaveTypes.value = t.data?.items ?? [];
    const b = await balancesQuery.refetch();
    balances.value = b.data?.items ?? [];
    const a = await appsQuery.refetch();
    applications.value = a.data?.items ?? [];
    appTotal.value = Number(a.data?.total ?? 0);
  } finally {
    loadingTypes.value = false;
    loadingBalances.value = false;
    loadingApps.value = false;
  }
}

const typeDialogVisible = ref(false);
const creatingType = ref(false);
const newType = reactive({ code: "", name: "" });
const createTypeMutation = useCreateLeaveType({
  onSuccess: () => {
    ElMessage.success("已创建");
    typeDialogVisible.value = false;
    newType.code = "";
    newType.name = "";
    loadAll();
  },
  onError: (err: Error) => ElMessage.error(err.message || "创建失败"),
});

function createType() {
  if (!newType.code || !newType.name) {
    ElMessage.warning("请填写代码与名称");
    return;
  }
  creatingType.value = true;
  createTypeMutation.mutate(
    { data: { code: newType.code, name: newType.name } },
    { onSettled: () => (creatingType.value = false) }
  );
}

const balanceDialogVisible = ref(false);
const granting = ref(false);
const newBalance = reactive({
  userId: 1,
  leaveTypeId: 1,
  year: new Date().getFullYear(),
  totalDays: 10,
});
const grantMutation = useGrantLeaveBalance({
  onSuccess: () => {
    ElMessage.success("已授予");
    balanceDialogVisible.value = false;
    loadAll();
  },
  onError: (err: Error) => ElMessage.error(err.message || "授予失败"),
});

function grantBalance() {
  granting.value = true;
  grantMutation.mutate(
    {
      userId: newBalance.userId,
      leaveTypeId: newBalance.leaveTypeId,
      year: newBalance.year,
      totalDays: newBalance.totalDays,
    },
    { onSettled: () => (granting.value = false) }
  );
}

function fmtDate(v?: string) {
  return v ? String(v).slice(0, 10) : "-";
}

function fmtTime(v?: string) {
  return v ? String(v).replace("T", " ").slice(0, 19) : "-";
}

function statusLabel(s?: string): string {
  switch (s) {
    case "APPROVED": return "已通过";
    case "REJECTED": return "已驳回";
    case "WITHDRAWN": return "已撤回";
    default: return "审批中";
  }
}

function statusTag(s?: string): "success" | "danger" | "info" | "warning" {
  switch (s) {
    case "APPROVED": return "success";
    case "REJECTED": return "danger";
    case "WITHDRAWN": return "info";
    default: return "warning";
  }
}

onMounted(loadAll);
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
.subtitle {
  font-size: 14px;
  font-weight: 600;
}
.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}
</style>
