<template>
  <ElSplitter>
    <ElSplitterPanel :size="25" :min-size="20" :max-size="25">
      <OrgList />
    </ElSplitterPanel>
    <ElSplitterPanel :size="75">
      <UserList />
    </ElSplitterPanel>
  </ElSplitter>
</template>

<script lang="ts" setup>
import { ElSplitter, ElSplitterPanel } from "element-plus";

import OrgList from "./org-list.vue";
import UserList from "./user-list.vue";
</script>
