<template>
  <Fallback status="500" />
</template>

<script lang="ts" setup>
import Fallback from "./fallback.vue";

defineOptions({ name: "Fallback500" });
</script>
