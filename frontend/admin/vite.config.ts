import vue from "@vitejs/plugin-vue";
import { type ConfigEnv, type UserConfig, loadEnv, defineConfig, type PluginOption } from "vite";

import AutoImport from "unplugin-auto-import/vite";
import Components from "unplugin-vue-components/vite";
import { ElementPlusResolver } from "unplugin-vue-components/resolvers";

import { mockDevServerPlugin } from "vite-plugin-mock-dev-server";

import tailwindcss from "@tailwindcss/vite";
import pkg from "./package.json" with { type: "json" };

// Vite配置  https://cn.vitejs.dev/config
export default defineConfig(({ mode }: ConfigEnv): UserConfig => {
  const env = loadEnv(mode, process.cwd());
  const isProduction = mode === "production";

  return {
    resolve: {
      // Vite 8 新特性：自动读取 tsconfig.json 中的 paths 别名
      tsconfigPaths: true,
    },
    css: {
      preprocessorOptions: {
        // 定义全局 SCSS 变量
        scss: {
          additionalData: `@use "@/styles/_variables.scss" as *;`,
        },
      },
    },
    server: {
      host: "0.0.0.0",
      port: +env.VITE_APP_PORT,
      open: true,
      proxy: {
        [env.VITE_APP_BASE_API]: {
          changeOrigin: true,
          target: env.VITE_APP_API_URL,
          rewrite: (path: string) => path.replace(new RegExp("^" + env.VITE_APP_BASE_API), ""),
        },
      },
    },
    plugins: [
      vue(),
      ...(env.VITE_MOCK_DEV_SERVER === "true" ? [mockDevServerPlugin()] : []),
      tailwindcss(),
      // API 自动导入
      AutoImport({
        // 导入 Vue 函数，如：ref, reactive, toRef 等
        imports: ["vue", "@vueuse/core", "pinia", "vue-router", "vue-i18n"],
        resolvers: [
          // 导入 Element Plus函数，如：ElMessage, ElMessageBox 等
          ElementPlusResolver(),
        ],
        eslintrc: {
          enabled: false,
          filepath: "./.eslintrc-auto-import.json",
          globalsPropValue: true,
        },
        vueTemplate: true,
        // 导入函数类型声明文件路径 (false:关闭自动生成)
        dts: false,
      }),
      // 组件自动导入
      Components({
        resolvers: [
          // 导入 Element Plus 组件
          ElementPlusResolver(),
        ],
        // 指定自定义组件位置(默认:src/components)
        dirs: ["src/components", "src/**/components"],
        // 导入组件类型声明文件路径 (false:关闭自动生成)
        dts: false,
      }),
    ] as PluginOption[],
    // 预构建依赖名单必须覆盖源码实际 import 的全部第三方包（含懒加载页面）。
    // 名单外的包会在 dev 下首次进入用到它的页面时才被发现，触发 Vite
    // "optimized dependencies changed. reloading" 整页刷新；而登录令牌仅存内存，
    // 整页刷新即被登出。下列清单由 src 全量 import 扫描生成（CSS/类型导入除外）。
    optimizeDeps: {
      include: [
        // 基础框架
        "vue",
        "vue-router",
        "element-plus",
        "pinia",
        "pinia-plugin-persistedstate",
        "axios",
        "@vueuse/core",
        "vue-i18n",
        "nprogress",
        "qs",
        "path-browserify",
        "path-to-regexp",
        "@element-plus/icons-vue",
        "element-plus/es/locale/lang/en",
        "element-plus/es/locale/lang/zh-cn",
        // 状态与请求
        "@tanstack/vue-query",
        "@microsoft/fetch-event-source",
        // 图表与表格
        "echarts",
        "echarts/charts",
        "echarts/components",
        "echarts/core",
        "echarts/features",
        "echarts/renderers",
        "vxe-table",
        // 编辑器（Tiptap / Monaco / Markdown / CodeMirror / JSON）
        "@tiptap/core",
        "@tiptap/extension-code-block-lowlight",
        "@tiptap/extension-color",
        "@tiptap/extension-heading",
        "@tiptap/extension-highlight",
        "@tiptap/extension-horizontal-rule",
        "@tiptap/extension-image",
        "@tiptap/extension-link",
        "@tiptap/extension-placeholder",
        "@tiptap/extension-subscript",
        "@tiptap/extension-superscript",
        "@tiptap/extension-table",
        "@tiptap/extension-table-cell",
        "@tiptap/extension-table-header",
        "@tiptap/extension-table-row",
        "@tiptap/extension-task-item",
        "@tiptap/extension-task-list",
        "@tiptap/extension-text-align",
        "@tiptap/extension-text-style",
        "@tiptap/extension-underline",
        "@tiptap/starter-kit",
        "@tiptap/vue-3",
        "highlight.js",
        "lowlight",
        "monaco-editor",
        "md-editor-v3",
        "codemirror-editor-vue3",
        "json-editor-vue",
        // 工具库
        "dayjs",
        "dayjs/plugin/relativeTime",
        "dayjs/plugin/timezone",
        "dayjs/plugin/utc",
        "lodash.clonedeep",
        "crypto-js",
        "clsx",
        "tailwind-merge",
        "defu",
        "dompurify",
        "marked",
        "exceljs",
        "sortablejs",
        "radix-vue",
        "@iconify/vue",
      ],
    },
    // 构建配置
    build: {
      chunkSizeWarningLimit: 2000,
      reportCompressedSize: false,
      minify: isProduction ? "esbuild" : false,
      target: "esnext",
      cssCodeSplit: true,
      rollupOptions: {
        output: {
          // 手动分块策略：将大型第三方库分离到独立 chunk，优化缓存和加载
          manualChunks(id) {
            if (!id.includes("node_modules")) return;

            // Monaco Editor — 体积巨大，单独拆分
            if (id.includes("monaco-editor")) return "monaco-editor";

            // ECharts — 图表库
            if (id.includes("echarts") || id.includes("zrender")) return "echarts";

            // Element Plus — UI 组件库
            if (id.includes("element-plus") || id.includes("@element-plus")) return "element-plus";

            // Vue 核心 & 生态
            if (
              id.includes("/vue/") ||
              id.includes("\\vue\\") ||
              id.includes("/@vue/") ||
              id.includes("\\@vue\\") ||
              id.includes("/vue-router/") ||
              id.includes("\\vue-router\\") ||
              id.includes("/pinia/") ||
              id.includes("\\pinia\\") ||
              id.includes("/vue-i18n/") ||
              id.includes("\\vue-i18n\\") ||
              id.includes("/@vueuse/") ||
              id.includes("\\@vueuse\\")
            ) {
              return "vue-vendor";
            }

            // VxeTable — 表格组件
            if (id.includes("vxe-table") || id.includes("vxe-pc-ui")) return "vxe-table";

            // Tiptap 富文本编辑器
            if (id.includes("@tiptap") || id.includes("tiptap")) return "tiptap";

            // 工具库
            if (id.includes("lodash") || id.includes("dayjs") || id.includes("axios")) return "utils-vendor";
          },
          // 用于从入口点创建的块的打包输出格式[name]表示文件名,[hash]表示该文件内容hash值
          entryFileNames: "js/[name].[hash].js",
          // 用于命名代码拆分时创建的共享块的输出命名
          chunkFileNames: "js/[name].[hash].js",
          // 用于输出静态资源的命名，[ext]表示文件扩展名
          assetFileNames: (assetInfo: any) => {
            // Vite 8 / Rolldown: 添加空值保护
            if (!assetInfo.name) {
              return "assets/[name].[hash][extname]";
            }
            const info = assetInfo.name.split(".");
            let extType = info[info.length - 1];
            if (/\.(mp4|webm|ogg|mp3|wav|flac|aac)(\?.*)?$/i.test(assetInfo.name)) {
              extType = "media";
            } else if (/\.(png|jpe?g|gif|svg)(\?.*)?$/.test(assetInfo.name)) {
              extType = "img";
            } else if (/\.(woff2?|eot|ttf|otf)(\?.*)?$/i.test(assetInfo.name)) {
              extType = "fonts";
            }
            return `${extType}/[name].[hash].[ext]`;
          },
        },
      },
    },
    // 生产环境精简 __APP_INFO__：不打包完整 dependencies/devDependencies，减少包体积
    define: {
      __APP_INFO__: JSON.stringify({
        pkg: {
          name: pkg.name,
          version: pkg.version,
          engines: pkg.engines,
        },
        buildTimestamp: Date.now(),
      }),
      ...(isProduction
        ? {
            "console.log": "(() => {})",
            "console.debug": "(() => {})",
            "console.info": "(() => {})",
          }
        : {}),
    },
  };
});
