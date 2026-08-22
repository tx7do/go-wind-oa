/**
 * API Hooks 索引文件。
 *
 * 基座管理域（用户/租户/权限/系统/日志/站内信/分析页）composables 已从
 * go-wind-admin vue-element 模板恢复；OA 业务域为 oa.ts。全量恢复（plan/mfa/redis-cache-monitor 后端已补齐）。
 *
 * auth：转发至 admin-service AuthenticationService（见
 * docs/oa-mobile-design.md §5.1，鉴权缺口已解决）。
 */
export * from "./shared";
export * from "./auth";
export * from "./oa";

// 管理门户相关
export * from "./admin-portal";

// 用户相关
export * from "./user";

// 用户个人资料
export * from "./user-profile";

// 租户管理
export * from "./tenant";

// 组织人员管理 (OPM)
export * from "./org-unit";
export * from "./position";

// 权限管理
export * from "./permission";
export * from "./permission-group";
export * from "./role";

// 系统管理
export * from "./menu";
export * from "./api";
export * from "./dict";
export * from "./file";
export * from "./file-transfer";
export * from "./task";
export * from "./login-policy";
export * from "./language";

// 日志审计
export * from "./login-audit-log";
export * from "./api-audit-log";
export * from "./operation-audit-log";
export * from "./data-access-audit-log";
export * from "./permission-audit-log";
export * from "./policy-evaluation-log";

// 首页分析概览
export * from "./dashboard";

// 内部消息
export * from "./internal-message";

// MFA（多因素认证）
export * from "./mfa";

// 租户套餐
export * from "./plan";

// Redis 缓存监控
export * from "./redis-cache-monitor";
