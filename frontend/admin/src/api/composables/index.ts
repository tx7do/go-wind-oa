/**
 * API Hooks 索引文件。
 *
 * OA 前端仅含工作流（oa）+ 通用枚举工具（shared）+ 认证（auth）。
 * 原 cms 的 internal_message / tenant / permission 等模块已随基座拷贝剥离。
 *
 * auth：转发至 cms admin-service AuthenticationService（见
 * docs/oa-mobile-design.md §5.1，鉴权缺口已解决）。
 */
export * from "./shared";
export * from "./auth";
export * from "./oa";
