<template>
  <Fallback status="401" />
</template>

<script lang="ts" setup>
import Fallback from "./fallback.vue";

defineOptions({ name: "Fallback401" });
</script>
