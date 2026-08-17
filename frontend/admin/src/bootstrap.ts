import { createApp } from "vue";

// ===== 样式导入 =====
import "element-plus/dist/index.css";
import "element-plus/theme-chalk/dark/css-vars.css";
import "vxe-table/lib/style.css";
import "@/styles/index.scss";
import "@/styles/tailwind.css";
import "animate.css";

import { setupDirective } from "@/directives";
import { setupI18n } from "@/core/i18n";
import { setupRouter } from "@/router";
import { initStores } from "@/stores/setup";
import { registerGlobComp } from "@/registerGlobComp";
import { initPreferences } from "@/core/preferences";
import { RequestClient } from "@/core/transport/rest";
import { logoutToLoginPage, refreshToken } from "@/composables/use-token-refresh";
import { useAccessStore } from "@/stores";
import { createI18nGetErrorMsg } from "@/composables/use-request-error-msg";
import { i18n } from "@/core/i18n";

import App from "./App.vue";
import { setupVueQuery } from "@/plugins/vue-query";

async function bootstrap(namespace: string) {
  const app = createApp(App);

  // 抑制 Element Plus ElForm labelWidth="auto" ResizeObserver 的无害警告
  // 该警告在布局过渡期（侧边栏宽度变化、布局切换等）时触发，属于正常现象。
  // 可能报告 width 0（容器未准备好）或 width NaN（中间态的无效计算值），两者均需过滤。
  const originalWarn = console.warn;
  console.warn = (...args: unknown[]) => {
    const msg = args[0];
    if (
      msg &&
      typeof msg === "object" &&
      (String(msg).includes("[ElForm] unexpected width 0") ||
        String(msg).includes("[ElForm] unexpected width NaN"))
    ) {
      return;
    }
    originalWarn.apply(console, args);
  };

  // 初始化偏好设置
  await initPreferences({
    namespace,
    overrides: {
      app: {
        name: import.meta.env.VITE_APP_TITLE || "GoWind Admin",
        version: import.meta.env.VITE_APP_VERSION || "0.0.0",
        enableTenant: import.meta.env.VITE_APP_TENANT_ENABLED === "true",
      },
    },
  });

  // 注册全局组件
  registerGlobComp(app);

  setupVueQuery(app);

  // 注册自定义指令
  setupDirective(app);

  // 配置 pinia-store
  await initStores(app, { namespace });

  // 注入 RequestClient 回调（业务层 → 基础设施层）
  // 必须在 initStores 之后，因为 getToken 依赖 accessStore
  const accessStore = useAccessStore();
  RequestClient.init(import.meta.env.VITE_APP_API_URL, {
    getToken: () => accessStore.accessToken,
    getLocale: () => i18n.global.locale.value,
    refreshToken,
    onReAuthenticate: logoutToLoginPage,
    onError: (msg) => console.error("[RequestClient]", msg),
    getErrorMsg: createI18nGetErrorMsg(),
  });

  // 配置路由及路由守卫
  setupRouter(app);

  // 国际化 i18n 配置
  await setupI18n(app);

  // 挂载应用
  app.mount("#app");
}

export { bootstrap };
