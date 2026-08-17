import { BasicLayout, Layout } from "@/layouts";
import { generateAccessible } from "@/core/router";
import { preferences } from "@/core/preferences";
import type {
  ComponentRecordType,
  GenerateMenuAndRoutesOptions,
} from "@/core/router/types";

const forbiddenComponent = () => import("@/pages/core/error/403.vue");

async function generateAccess(options: GenerateMenuAndRoutesOptions) {
  const pageMap: ComponentRecordType = import.meta.glob("../pages/**/*.vue");

  const layoutMap: ComponentRecordType = {
    BasicLayout,
    Layout,
  };

  return await generateAccessible(preferences.app.accessMode, {
    ...options,
    // OA 采用 frontend accessMode（见 preferences/config/default.ts），
    // generateAccessible 的 frontend 分支不调用 fetchMenuListAsync，
    // 路由来自 routes/modules 下的前端模块（import.meta.glob 自动收编）。
    // fetchMenuListAsync 仅为类型完整保留，返回空数组，永不被执行。
    fetchMenuListAsync: async () => [],
    forbiddenComponent,
    layoutMap,
    pageMap,
  });
}

export { generateAccess };
