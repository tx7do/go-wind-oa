<template>
  <ProModal
    v-model:visible="visible"
    :title="$t('pages.oa.expense.detail.title')"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <ElTable :data="items" border size="small">
      <ElTableColumn
        prop="category"
        :label="$t('pages.oa.expense.detail.colCategory')"
        width="120"
      />
      <ElTableColumn prop="amount" :label="$t('pages.oa.expense.detail.colAmount')" width="120" />
      <ElTableColumn :label="$t('pages.oa.expense.detail.colExpenseDate')" width="120">
        <template #default="{ row }">{{ formatDate(row.expenseDate) }}</template>
      </ElTableColumn>
      <ElTableColumn
        prop="description"
        :label="$t('pages.oa.expense.detail.colDescription')"
        min-width="160"
      />
      <ElTableColumn
        prop="invoiceFileId"
        :label="$t('pages.oa.expense.detail.colInvoiceFileId')"
        width="110"
      />
    </ElTable>
  </ProModal>
</template>

<script lang="ts" setup>
import { ElTable, ElTableColumn } from "element-plus";
import { ref } from "vue";
import ProModal from "@/components/Pro/ProModal/index.vue";
import { DRAWER_WIDTH } from "@/constants";
import { useOaDrawer } from "../use-oa-drawer";
import { formatDate } from "@/utils/date";

const { visible, open } = useOaDrawer((rowItems: any[]) => {
  items.value = rowItems ?? [];
});
const items = ref<any[]>([]);

defineExpose({ open });
</script>
