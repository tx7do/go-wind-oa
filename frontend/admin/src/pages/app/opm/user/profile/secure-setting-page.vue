<template>
  <div class="page-container">
    <div class="setting-list">
      <div v-for="item in secureSettingList" :key="item.key" class="setting-item">
        <div class="setting-item-content">
          <span class="item-title">{{ item.title }}</span>
          <ElLink v-if="item.extra" type="primary" underline="never" class="item-link">
            {{ item.extra }}
          </ElLink>
        </div>
        <div class="item-description">{{ item.description }}</div>
      </div>

      <!-- MFA（TOTP）绑定/解绑：真实功能组件，绑定后登录需二次验证 -->
      <div class="setting-item">
        <MfaManagement />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { $t } from "@/core/i18n";

import MfaManagement from "./MfaManagement.vue";

interface SettingItem {
  key: string;
  title: string;
  description: string;
  extra?: string;
}

const secureSettingList: SettingItem[] = [
  {
    key: "1",
    title: $t("pages.user.secureSetting.accountPassword"),
    description: $t("pages.user.secureSetting.accountPasswordDesc"),
    extra: $t("common.button.edit"),
  },
  {
    key: "2",
    title: $t("pages.user.secureSetting.securePhone"),
    description: $t("pages.user.secureSetting.securePhoneDesc"),
    extra: $t("common.button.edit"),
  },
  {
    key: "3",
    title: $t("pages.user.secureSetting.secureQuestion"),
    description: $t("pages.user.secureSetting.secureQuestionDesc"),
    extra: $t("common.button.edit"),
  },
  {
    key: "4",
    title: $t("pages.user.secureSetting.backupEmail"),
    description: $t("pages.user.secureSetting.backupEmailDesc"),
    extra: $t("common.button.edit"),
  },
];
</script>

<style lang="scss" scoped>
.page-container {
  width: 100%;
  max-width: 800px;
}

.setting-list {
  padding-top: 20px;
}

.setting-item {
  padding: 16px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);

  &:last-child {
    border-bottom: none;
  }
}

.setting-item-content {
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

.item-link {
  flex-shrink: 0;
}

.item-description {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.5;
}
</style>
