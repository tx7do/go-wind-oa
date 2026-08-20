/**
 * 认证相关 Composable
 * 替代 authentication.store.ts，提供登录/登出/注册/验证码/获取用户信息/权限码等功能
 */
import { ref } from "vue";
import { router } from "@/router";

import { DEFAULT_HOME_PATH } from "@/constants";
import { resetAllStores, useAccessStore, useAppUserStore } from "@/stores";

import { ElNotification } from "element-plus";
import CryptoJS from "crypto-js";

import {
  login as authLogin,
  logout as authLogout,
  generateCaptcha as authGenerateCaptcha,
} from "@/api/composables";
import { i18n } from "@/core/i18n";
import { setCaptchaHeaders } from "@/core/transport/rest";
import {
  startRefreshTimer,
  stopRefreshTimer,
  connectSSEServer,
  logoutToLoginPage,
} from "@/composables/use-token-refresh";

import { createI18nGetErrorMsg } from "@/composables/use-request-error-msg";

const t = i18n.global.t;
const getErrorMsg = createI18nGetErrorMsg();

// ==============================
// 网络异常标记类
// ==============================

class NetworkError extends Error {
  constructor() {
    super("Network Error");
    this.name = "NetworkError";
  }
}

export { NetworkError };

// ==============================
// 登录加载状态（模块级单例）
// ==============================

const loginLoading = ref(false);

// ==============================
// 加密工具
// ==============================

function encryptData(data: string, key: string, iv: string): string {
  const keyHex = CryptoJS.enc.Utf8.parse(key);
  const ivHex = CryptoJS.enc.Utf8.parse(iv);
  const encrypted = CryptoJS.AES.encrypt(data, keyHex, {
    iv: ivHex,
    mode: CryptoJS.mode.CBC,
    padding: CryptoJS.pad.Pkcs7,
  });
  return encrypted.toString();
}

function encryptPassword(password: string): string {
  const key = import.meta.env.VITE_AES_KEY;
  if (!key) {
    throw new Error("VITE_AES_KEY is not set in environment");
  }
  return encryptData(password, key, key);
}

// ==============================
// 常量
// ==============================

const ACCESS_TOKEN_REFRESH_INTERVAL = 90 * 60 * 1000;
const REFRESH_TOKEN_REFRESH_INTERVAL = 12 * 60 * 60 * 1000;

// ==============================
// 核心业务逻辑
// ==============================

// OA 后端鉴权转发仅提供 Login/Logout/RefreshToken/GenerateCaptcha/VerifyCaptcha，
// 无 getMe / getMyPermissionCode / registerUser（依赖 identity/permission 域，
// OA v1 未含）。userInfo 从本地保存的 accessToken JWT payload 解出（uid/sub/tid/roc），
// 仅作展示与角色填充用——鉴权以服务端 ValidateToken 为准，前端解码不可信。
// accessCodes 无来源，恒为空数组（OA 路由未使用 authority 过滤）。
function decodeJwtPayload(token: string): Record<string, any> | null {
  try {
    const part = token.split(".")[1];
    if (!part) return null;
    const normalized = part.replace(/-/g, "+").replace(/_/g, "/");
    const padded = normalized + "=".repeat((4 - (normalized.length % 4)) % 4);
    return JSON.parse(decodeURIComponent(escape(atob(padded))));
  } catch {
    return null;
  }
}

function fetchUserInfo(): UserInfo | null {
  const accessStore = useAccessStore();
  const token = accessStore.accessToken ?? "";
  const claims = token ? decodeJwtPayload(String(token)) : null;

  const username = claims?.sub ? String(claims.sub) : "";
  return {
    id: claims?.uid ? Number(claims.uid) : 0,
    username,
    nickname: username,
    realname: username,
    avatar: "",
    description: "",
    homePath: DEFAULT_HOME_PATH,
    token: "",
    roles: Array.isArray(claims?.roc) ? claims.roc.map(String) : [],
    tenantId: claims?.tid ? Number(claims.tid) : 0,
  };
}

async function fetchAccessCodes() {
  return { codes: [] };
}

async function login(
  params: Record<string, any>,
  onSuccess?: () => Promise<void> | void
): Promise<{ userInfo: null | UserInfo } | null> {
  let userInfo: null | UserInfo = null;
  try {
    loginLoading.value = true;

    // 若表单携带验证码，先设置一次性 Header（由 transport.unary 消费）
    if (params.captchaId && params.captchaCode) {
      setCaptchaHeaders(params.captchaId, params.captchaCode);
    }

    const resp = await authLogin({
      username: params.username,
      password: encryptPassword(params.password),
      tenant_code: params.tenant_code,
      grant_type: "password",
    });

    const accessToken = (resp as any).access_token;
    const refresh_token = (resp as any).refresh_token;
    let expiresAt = (resp as any).expires_in;
    let refreshExpiresAt = (resp as any).refresh_expires_in;

    const accessStore = useAccessStore();

    const expiresAtSec = Number(expiresAt);
    expiresAt =
      !Number.isFinite(expiresAtSec) || expiresAtSec <= 0
        ? Date.now() + ACCESS_TOKEN_REFRESH_INTERVAL
        : Date.now() + Math.floor(expiresAtSec * 1000);

    const refreshExpiresAtSec = Number(refreshExpiresAt);
    refreshExpiresAt =
      !Number.isFinite(refreshExpiresAtSec) || refreshExpiresAtSec <= 0
        ? Date.now() + REFRESH_TOKEN_REFRESH_INTERVAL
        : Date.now() + Math.floor(refreshExpiresAtSec * 1000);

    if (accessToken) {
      accessStore.setAccessToken(accessToken);
      accessStore.setAccessTokenExpireTime(expiresAt);

      if (refresh_token) {
        accessStore.setRefreshToken(refresh_token);
        accessStore.setRefreshTokenExpireTime(refreshExpiresAt);
        startRefreshTimer();
      }

      const [fetchUserInfoResult, fetchAccessCodeResult] = await Promise.all([
        fetchUserInfo(),
        fetchAccessCodes(),
      ]);

      userInfo = fetchUserInfoResult;
      if (!userInfo) {
        throw new Error(t("core.authentication.loginFailedDesc"));
      }

      const userStore = useAppUserStore();
      userStore.setUserInfo(userInfo);
      accessStore.setAccessCodes(fetchAccessCodeResult.codes ?? []);

      if (accessStore.loginExpired) {
        accessStore.setLoginExpired(false);
      } else {
        if (onSuccess) {
          await onSuccess();
        } else {
          await router.push(userInfo.homePath || DEFAULT_HOME_PATH);
        }
      }

      if (userInfo?.realname) {
        ElNotification({
          title: t("core.authentication.loginSuccess"),
          message: `${t("core.authentication.loginSuccessDesc")}:${userInfo?.realname}`,
          type: "success",
          duration: 3000,
        });
      }
    }
  } catch (error) {
    await _doLogout();

    // 使用 i18n 翻译错误消息（与 RequestClient 的 getErrorMsg 一致）
    const errorMsg = getErrorMsg(error);
    ElNotification({
      title: t("core.authentication.loginFailed"),
      message: errorMsg,
      type: "error",
    });
    return null;
  } finally {
    loginLoading.value = false;
  }

  return { userInfo };
}

async function _doLogout(redirect: boolean = true) {
  console.log("_doLogout");
  stopRefreshTimer();
  resetAllStores();
  const accessStore = useAccessStore();
  accessStore.setLoginExpired(false);
  loginLoading.value = false;
  await logoutToLoginPage(redirect);
}

async function logout(redirect: boolean = true) {
  const accessStore = useAccessStore();
  try {
    if (accessStore.accessToken !== null && accessStore.accessToken !== "") {
      await authLogout();
    }
  } catch {
    // 忽略错误
  }
  await _doLogout(redirect);
}

async function register(_username: string, _password: string) {
  // OA 鉴权转发层未含 RegisterUser，注册暂不可用。
  throw new Error(t("core.authentication.registerUnavailable"));
}

async function getCaptcha() {
  return await authGenerateCaptcha();
}

/**
 * 判断是否为网络异常（非业务错误）
 * 网络异常的特征：AxiosError 且 code 为 ERR_NETWORK，或 message 包含 Network Error
 */
function isNetworkError(error: unknown): boolean {
  if (!error || typeof error !== "object") return false;
  const err = error as Record<string, unknown>;
  // AxiosError: code === 'ERR_NETWORK'
  if (err.code === "ERR_NETWORK") return true;
  // 兜底：message 检测
  if (typeof err.message === "string" && err.message.includes("Network Error")) return true;
  return false;
}

async function getUserPermissionCodes() {
  const accessStore = useAccessStore();
  const userStore = useAppUserStore();

  if (userStore.userInfo === null || accessStore.accessCodes === null) {
    try {
      const [fetchUserInfoResult, fetchAccessCodeResult] = await Promise.all([
        fetchUserInfo(),
        fetchAccessCodes(),
      ]);
      if (fetchUserInfoResult === null || fetchAccessCodeResult === null) {
        console.warn("setupAccessGuard failed fetch user info:", fetchUserInfoResult);
        return false;
      }
      userStore.setUserInfo(fetchUserInfoResult);

      // 只存权限码，角色码由 userStore.userRoles 管理
      const codes = fetchAccessCodeResult ? (fetchAccessCodeResult.codes ?? []) : [];
      accessStore.setAccessCodes(codes);
    } catch (error: unknown) {
      // 网络异常：抛出特定标记，让路由守卫跳转错误页而非白屏
      if (isNetworkError(error)) {
        throw new NetworkError();
      }
      // 其他错误（如 401/403）：标记为认证失败
      return false;
    }
  }

  startRefreshTimer();
  connectSSEServer();

  // 返回 true 表示成功获取权限数据
  return true;
}

// ==============================
// 导出
// ==============================

export function useAuth() {
  return {
    loginLoading,
    login,
    logout,
    register,
    getCaptcha,
    fetchUserInfo,
    fetchAccessCodes,
    getUserPermissionCodes,
  };
}
