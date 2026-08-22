<template>
  <div class="page-container">
    <div class="notify-list">
      <div v-for="item in msgNotifyList" :key="item.key" class="notify-item">
        <div class="notify-item-content">
          <span class="item-title">{{ item.title }}</span>
          <ElSwitch
            v-model="item.checked"
            :active-text="$t('common.switch.active')"
            :inactive-text="$t('common.switch.inactive')"
            class="item-switch"
          />
        </div>
        <div class="item-description">{{ item.description }}</div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { reactive } from "vue";

import { $t } from "@/core/i18n";

interface NotifyItem {
  key: string;
  title: string;
  description: string;
  checked: boolean;
}

const msgNotifyList = reactive<NotifyItem[]>([
  {
    key: "1",
    title: $t("pages.user.msgNotify.accountPassword"),
    description: $t("pages.user.msgNotify.accountPasswordDesc"),
    checked: true,
  },
  {
    key: "2",
    title: $t("pages.user.msgNotify.systemMessage"),
    description: $t("pages.user.msgNotify.systemMessageDesc"),
    checked: true,
  },
  {
    key: "3",
    title: $t("pages.user.msgNotify.todoTask"),
    description: $t("pages.user.msgNotify.todoTaskDesc"),
    checked: true,
  },
]);
</script>

<style lang="scss" scoped>
.page-container {
  width: 100%;
  max-width: 800px;
}

.notify-list {
  padding-top: 20px;
}

.notify-item {
  padding: 16px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);

  &:last-child {
    border-bottom: none;
  }
}

.notify-item-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  margin-bottom: 8px;
}

.item-title {
  flex: 1;
  font-size: 14px;
  font-weight: 500;
}

.item-switch {
  flex-shrink: 0;
}

.item-description {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.5;
}
</style>
