/**
 * 认证 composable。
 *
 * 类型来源 @/api/generated/admin/service/v1，由 buf.admin.typescript.gen.yaml
 * 从 api/protos/admin/service/v1/i_authentication.proto 生成。admin-service
 * 的 AuthenticationService HTTP 边端将登录/登出/刷新令牌/验证码请求转发至
 * core-service 的同名 gRPC 服务。
 */
import {
  useMutation,
  useQuery,
  type UseMutationOptions,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  authenticationservicev1_GenerateCaptchaResponse,
  authenticationservicev1_LoginRequest,
  authenticationservicev1_LoginResponse,
} from "@/api/generated/admin/service/v1";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";

// 直接导出函数，供非 Vue 上下文使用
export async function login(request: authenticationservicev1_LoginRequest) {
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
  options?: UseMutationOptions<authenticationservicev1_LoginResponse, Error, authenticationservicev1_LoginRequest>
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
  options?: UseMutationOptions<authenticationservicev1_LoginResponse, Error, authenticationservicev1_LoginRequest>
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
  options?: UseQueryOptions<authenticationservicev1_GenerateCaptchaResponse, Error>
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
