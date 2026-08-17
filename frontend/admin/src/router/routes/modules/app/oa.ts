import type { RouteRecordRaw } from "vue-router";

import { Layout } from "@/layouts";

/**
 * OA 工作流定义管理路由模块。
 *
 * 对齐 cms internal_message.ts 路由模块模式：Layout 包裹，children 为具体页面，
 * meta.authority 限定平台管理员。accessMode=frontend 下，routes 聚合器对
 * modules 目录下所有 .ts 做 eager glob 自动收编本文件，无需改聚合器。
 */
const oa: RouteRecordRaw[] = [
  {
    path: "/oa",
    name: "OaManagement",
    redirect: "/oa/definitions",
    component: Layout,
    meta: {
      order: 2003,
      icon: "lucide:file-work",
      title: "routes.oa.moduleName",
      keepAlive: true,
      authority: ["sys:platform_admin"],
    },
    children: [
      {
        path: "definitions",
        name: "OaWorkflowDefinition",
        meta: {
          order: 1,
          icon: "lucide:file-work",
          title: "routes.oa.definition",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/pages/app/oa/definition/index.vue"),
      },
    ],
  },
];

export default oa;
