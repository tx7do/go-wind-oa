<template>
  <ElSplitter>
    <ElSplitterPanel :size="40" :min-size="30" :max-size="50">
      <DictTypeList />
    </ElSplitterPanel>
    <ElSplitterPanel :size="60">
      <DictEntryList />
    </ElSplitterPanel>
  </ElSplitter>
</template>

<script lang="ts" setup>
import { ElSplitter, ElSplitterPanel } from "element-plus";

import DictEntryList from "./dict-entry-list.vue";
import DictTypeList from "./dict-type-list.vue";
</script>
