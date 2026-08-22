<template>
  <div class="page-container">
    <div class="account-list">
      <div v-for="item in accountBindList" :key="item.key" class="account-item">
        <div class="account-item-content">
          <!-- 左侧：图标和标题 -->
          <div class="item-left">
            <IconifyIcon
              :icon="item.avatar"
              :width="28"
              :height="28"
              :color="item.color"
              class="item-avatar"
            />
            <div class="item-info">
              <span class="item-title">{{ item.title }}</span>
              <span class="item-description">{{ item.description }}</span>
            </div>
          </div>
          <!-- 右侧：操作链接 -->
          <ElLink
            v-if="item.extra"
            type="primary"
            underline="never"
            :disabled="item.disabled"
            class="item-link"
          >
            {{ item.extra }}
          </ElLink>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { Icon as IconifyIcon } from "@iconify/vue";
import { $t } from "@/core/i18n";

interface AccountBindItem {
  key: string;
  title: string;
  description: string;
  extra: string;
  avatar: string;
  color: string;
  status: "bound" | "pending" | "unbound";
  boundTime?: string;
  isPrimary?: boolean;
  disabled?: boolean;
  platform?: string;
  required?: boolean;
}

// 动态标题和描述，使用国际化
const accountBindList = ref<AccountBindItem[]>([
  {
    key: "email",
    title: $t("pages.user.accountBind.email"),
    description: $t("pages.user.accountBind.emailDesc"),
    extra: $t("common.button.edit"),
    avatar: "ri:mail-fill",
    color: "#5470c6",
    status: "bound",
    boundTime: "2023-09-01T10:00:00Z",
    isPrimary: true,
    required: true,
    platform: "email",
  },
  {
    key: "phone",
    title: $t("pages.user.accountBind.phone"),
    description: $t("pages.user.accountBind.phoneDesc"),
    extra: $t("common.button.edit"),
    avatar: "ri:smartphone-fill",
    color: "#722ed1",
    status: "bound",
    boundTime: "2023-09-01T10:05:00Z",
    isPrimary: true,
    required: true,
    platform: "phone",
  },
  {
    key: "github",
    title: $t("pages.user.accountBind.github"),
    description: $t("pages.user.accountBind.githubDesc"),
    extra: $t("pages.user.accountBind.manage"),
    avatar: "fa-brands:github",
    color: "#333",
    status: "bound",
    boundTime: "2023-10-15T09:20:00Z",
    isPrimary: true,
    disabled: true,
    platform: "github",
  },
  {
    key: "wechat",
    title: $t("pages.user.accountBind.wechat"),
    description: $t("pages.user.accountBind.wechatDesc"),
    extra: $t("pages.user.accountBind.unbind"),
    avatar: "ri:wechat-fill",
    color: "#2dc26b",
    status: "bound",
    boundTime: "2024-01-20T14:35:00Z",
    isPrimary: false,
    platform: "wechat",
  },
  {
    key: "weibo",
    title: $t("pages.user.accountBind.weibo"),
    description: $t("pages.user.accountBind.weiboDesc"),
    extra: $t("pages.user.accountBind.bindNow"),
    avatar: "ri:weibo-fill",
    color: "#e6162d",
    status: "unbound",
    platform: "weibo",
  },
  {
    key: "dingtalk",
    title: $t("pages.user.accountBind.dingtalk"),
    description: $t("pages.user.accountBind.dingtalkDesc"),
    extra: $t("pages.user.accountBind.bind"),
    avatar: "ri:dingding-fill",
    color: "#2eabff",
    status: "unbound",
    platform: "dingtalk",
  },
]);
</script>

<style lang="scss" scoped>
.page-container {
  width: 100%;
  max-width: 800px;
}

.account-list {
  padding-top: 20px;
}

.account-item {
  padding: 16px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);

  &:last-child {
    border-bottom: none;
  }
}

.account-item-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.item-left {
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 0;
}

.item-avatar {
  margin-right: 16px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.item-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.item-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  line-height: 1.4;
}

.item-description {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.item-link {
  flex-shrink: 0;
  font-size: 14px;
}
</style>
