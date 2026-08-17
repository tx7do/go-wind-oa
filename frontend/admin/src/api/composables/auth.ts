/**
 * 认证 composable。
 *
 * 对齐 go-wind-admin/frontend/admin/vue-element/src/api/composables/auth.ts，
 * 差异：
 *  - 类型来源改为 @/api/generated/oa/v1（由 buf.vue-element.oa.typescript.gen.yaml
 *    从 OA backend api/protos/oa/v1/authentication.proto 生成）；
 *  - 删除 RegisterUser（依赖 cms user.proto，OA 鉴权转发层未含）。
 *
 * OA 后端 AuthenticationService 经 oa→cms proto 翻译（wire-compatible，
 * proto.Marshal/Unmarshal）转发至 cms admin-service 同名 RPC，故登录/登出/
 * 刷新令牌/验证码均与 go-wind-admin 同款语义。鉴权缺口已闭合——见
 * docs/oa-mobile-design.md §“鉴权缺口”（已更新为已解决）。
 */
import {
  useMutation,
  useQuery,
  type UseMutationOptions,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  GenerateCaptchaResponse,
  LoginRequest,
  LoginResponse,
} from "@/api/generated/oa/v1";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";

// 直接导出函数，供非 Vue 上下文使用
export async function login(request: LoginRequest) {
  return apiClient.authenticationService.Login(request);
}

export async function logout() {
  return apiClient.authenticationService.Logout({});
}

export async function generateCaptcha() {
  return apiClient.authenticationService.GenerateCaptcha({});
}

export async function refreshToken(refreshToken: string) {
  return apiClient.authenticationService.RefreshToken({
    grant_type: "refresh_token",
    refresh_token: refreshToken ?? "",
  });
}

// ------------------------------
// 登录（Mutation）
// ------------------------------
export function useLogin(
  options?: UseMutationOptions<LoginResponse, Error, LoginRequest>
) {
  return useMutation({
    mutationFn: (req) => login(req),
    ...options,
  });
}

// ------------------------------
// 登录（Mutation - GET）
// ------------------------------
export const loginMutation = queryClient.getMutationCache().build(queryClient, {
  mutationKey: ["login"],
  mutationFn: login,
  retry: 0,
});

// ------------------------------
// 登出（Mutation）
// ------------------------------
export function useLogout(options?: UseMutationOptions<{}, Error, {}>) {
  return useMutation({
    mutationFn: () => logout(),
    ...options,
  });
}

// ------------------------------
// 登出（Mutation - GET）
// ------------------------------
export const logoutMutation = queryClient.getMutationCache().build(queryClient, {
  mutationKey: ["logout"],
  mutationFn: logout,
  retry: 0,
});

// ------------------------------
// 刷新 Token（Mutation）
// ------------------------------
export function useRefreshToken(
  options?: UseMutationOptions<LoginResponse, Error, LoginRequest>
) {
  return useMutation({
    mutationFn: (req) => refreshToken(req.refresh_token ?? ""),
    ...options,
  });
}

// ------------------------------
// 刷新 Token（Mutation - GET）
// ------------------------------
export const refreshTokenMutation = queryClient
  .getMutationCache()
  .build(queryClient, {
    mutationKey: ["refreshToken"],
    mutationFn: refreshToken,
    retry: 0,
  });

// ------------------------------
// 获取验证码（Query - GET）
// ------------------------------
export function useGenerateCaptcha(
  options?: UseQueryOptions<GenerateCaptchaResponse, Error>
) {
  return useQuery({
    queryKey: ["captcha"],
    queryFn: () => generateCaptcha(),
    ...options,
  });
}

// ==============================================
// 获取验证码 【给 Store / 外部调用】不带 Hook 的方法
// ==============================================
export async function fetchGenerateCaptcha() {
  return queryClient.fetchQuery({
    queryKey: ["generateCaptcha"],
    queryFn: () => generateCaptcha(),
    staleTime: 0,
    retry: 0,
  });
}
