<template>
  <ProModal
    v-model:visible="visible"
    title="报销明细"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <ElTable :data="items" border size="small">
      <ElTableColumn prop="category" label="类别" width="120" />
      <ElTableColumn prop="amount" label="金额" width="120" />
      <ElTableColumn label="费用日期" width="120">
        <template #default="{ row }">{{ fmtDate(row.expenseDate) }}</template>
      </ElTableColumn>
      <ElTableColumn prop="description" label="说明" min-width="160" />
      <ElTableColumn prop="invoiceFileId" label="发票文件ID" width="110" />
    </ElTable>
  </ProModal>
</template>

<script lang="ts" setup>
import { ElTable, ElTableColumn } from "element-plus";
import { ref } from "vue";
import ProModal from "@/components/Pro/ProModal/index.vue";
import { DRAWER_WIDTH } from "@/constants";

const visible = ref(false);
const items = ref<any[]>([]);

function fmtDate(v?: string) {
  return v ? String(v).slice(0, 10) : "-";
}

function open(rowItems: any[]) {
  items.value = rowItems ?? [];
  visible.value = true;
}

defineExpose({ open });
</script>
