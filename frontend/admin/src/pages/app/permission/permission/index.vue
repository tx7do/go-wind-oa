<template>
  <ElSplitter>
    <ElSplitterPanel :size="40" :min-size="30" :max-size="50">
      <PermissionGroupList />
    </ElSplitterPanel>
    <ElSplitterPanel :size="60">
      <PermissionList />
    </ElSplitterPanel>
  </ElSplitter>
</template>

<script lang="ts" setup>
import { ElSplitter, ElSplitterPanel } from "element-plus";

import PermissionGroupList from "./permission-group-list.vue";
import PermissionList from "./permission-list.vue";
</script>
