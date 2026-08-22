<template>
  <ElSplitter>
    <ElSplitterPanel :size="40" :min-size="30" :max-size="50">
      <PlanList />
    </ElSplitterPanel>
    <ElSplitterPanel :size="60">
      <PlanQuotaList />
    </ElSplitterPanel>
  </ElSplitter>
</template>

<script lang="ts" setup>
import { ElSplitter, ElSplitterPanel } from "element-plus";

import PlanList from "./plan-list.vue";
import PlanQuotaList from "./plan-quota-list.vue";
</script>
