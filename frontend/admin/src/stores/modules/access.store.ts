import type { RouteRecordRaw } from "vue-router";

import { acceptHMRUpdate, defineStore } from "pinia";
import type { MenuRecordRaw } from "@/core/router/types";

/**
 * @zh_CN 访问令牌类型
 */
type AccessToken = null | string;

/**
 * @zh_CN 访问权限相关状态定义
 */
interface AccessState {
  /**
   * 权限码
   */
  accessCodes: string[];
  /**
   * 可访问的菜单列表
   */
  accessMenus: MenuRecordRaw[];
  /**
   * 可访问的路由列表
   */
  accessRoutes: RouteRecordRaw[];
  /**
   * 登录 accessToken
   */
  accessToken: AccessToken;
  /**
   * MFA 挑战操作标识（登录二次验证阶段暂存，验证通过后清空）
   */
  mfaOperationId: string | null;
  /**
   * accessToken 过期时间戳
   */
  accessTokenExpireTime?: number;
  /**
   * 是否已经检查过权限
   */
  isAccessChecked: boolean;
  /**
   * 登录是否过期
   */
  loginExpired: boolean;

  /**
   * 登录 accessToken
   */
  refreshToken: AccessToken;

  /**
   * refreshToken 过期时间戳
   */
  refreshTokenExpireTime?: number;
}

/**
 * @zh_CN 访问权限相关状态管理
 */
export const useAccessStore = defineStore("core-access", {
  actions: {
    $reset() {
      this.accessToken = null;
      this.refreshToken = null;
      this.accessCodes = [];
      this.accessMenus = [];
      this.accessRoutes = [];
      this.isAccessChecked = false;
      this.loginExpired = false;
      this.accessTokenExpireTime = undefined;
      this.refreshTokenExpireTime = undefined;
    },
    /**
     * @zh_CN 检查 accessToken 是否过期
     */
    checkAccessTokenExpired(): boolean {
      if (!this.accessTokenExpireTime) {
        return true;
      }
      const now = Date.now();
      return now >= this.accessTokenExpireTime;
    },
    /**
     * @zh_CN 检查 refreshToken 是否过期
     */
    checkRefreshTokenExpired(): boolean {
      if (!this.refreshTokenExpireTime) {
        return true;
      }
      const now = Date.now();
      return now >= this.refreshTokenExpireTime;
    },
    setAccessCodes(codes: string[]) {
      this.accessCodes = codes;
    },
    setAccessMenus(menus: MenuRecordRaw[]) {
      this.accessMenus = menus;
    },
    setAccessRoutes(routes: RouteRecordRaw[]) {
      this.accessRoutes = routes;
    },
    setAccessToken(token: AccessToken) {
      this.accessToken = token;
    },
    setAccessTokenExpireTime(accessTokenExpireTime: number) {
      this.accessTokenExpireTime = accessTokenExpireTime;
    },
    setIsAccessChecked(isAccessChecked: boolean) {
      this.isAccessChecked = isAccessChecked;
    },

    setLoginExpired(loginExpired: boolean) {
      this.loginExpired = loginExpired;
    },

    setRefreshToken(token: AccessToken) {
      this.refreshToken = token;
    },

    setRefreshTokenExpireTime(refreshTokenExpireTime: number) {
      this.refreshTokenExpireTime = refreshTokenExpireTime;
    },
  },
  persist: {
    // Token（access/refresh 及其过期时间）持久化到 localStorage，刷新/重开浏览器后
    // 守卫凭 accessToken 直接恢复会话：userInfo 由 JWT 本地解码重建，令牌定时刷新
    // 与 SSE 在守卫的 getUserPermissionCodes 里重启（isAccessChecked 不持久化，
    // 刷新后必为 false，恰好触发这条恢复链）。登出走 $reset，插件会把清空后的
    // 状态同步写回 storage。会话态字段（mfaOperationId/loginExpired/isAccessChecked）
    // 仍不持久化。
    // 注：曾刻意仅存内存（防 XSS 从 storage 读 token），因"任何整页刷新即被登出"
    // 的体验问题（F5、Vite 依赖预构建 reload 等都会触发）改回持久化。
    pick: [
      "accessCodes",
      "accessToken",
      "accessTokenExpireTime",
      "refreshToken",
      "refreshTokenExpireTime",
    ],
  },
  state: (): AccessState => ({
    accessCodes: [],
    accessMenus: [],
    accessRoutes: [],
    accessToken: null,
    mfaOperationId: null,
    accessTokenExpireTime: undefined,
    isAccessChecked: false,
    loginExpired: false,
    refreshToken: null,
    refreshTokenExpireTime: undefined,
  }),
});

// 解决热更新问题
const hot = import.meta.hot;
if (hot) {
  hot.accept(acceptHMRUpdate(useAccessStore, hot));
}
