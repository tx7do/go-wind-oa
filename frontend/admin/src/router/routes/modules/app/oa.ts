import type { RouteRecordRaw } from "vue-router";

import { Layout } from "@/layouts";

/**
 * OA 管理路由模块。
 *
 * 对齐 cms internal_message.ts 路由模块模式：Layout 包裹，children 为具体页面，
 * meta.authority 限定平台管理员。accessMode=frontend 下，routes 聚合器对
 * modules 目录下所有 .ts 做 eager glob 自动收编本文件，无需改聚合器。
 *
 * 页面：
 *   - 流程定义管理
 *   - 考勤地理围栏库 CRUD
 *   - 考勤 Wi-Fi 指纹库 CRUD
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
      {
        path: "attendance-fence",
        name: "OaAttendanceFence",
        meta: {
          order: 2,
          icon: "lucide:map-pin",
          title: "routes.oa.attendanceFence",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/pages/app/oa/attendance/fence.vue"),
      },
      {
        path: "attendance-wifi",
        name: "OaAttendanceWifi",
        meta: {
          order: 3,
          icon: "lucide:wifi",
          title: "routes.oa.attendanceWifi",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/pages/app/oa/attendance/wifi.vue"),
      },
    ],
  },
];

export default oa;
