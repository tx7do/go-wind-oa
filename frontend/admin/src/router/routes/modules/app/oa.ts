import type { RouteRecordRaw } from "vue-router";

import { Layout } from "@/layouts";

/**
 * OA 管理路由模块。
 *
 * 对齐 cms internal_message.ts 路由模块模式：Layout 包裹，children 为具体页面。
 * accessMode=frontend 下，routes 聚合器对 modules 目录下所有 .ts 做 eager glob
 * 自动收编本文件，无需改聚合器。
 *
 * 不设 meta.authority：OA v1 后端无角色/权限表（authz 为 noop），登录即视为
 * 可访问后台；待 identity/permission 域落地后再恢复 authority 过滤。
 *
 * 页面：
 *   - 流程定义管理
 *   - 考勤记录 / 设置 / 结算
 *   - 请假管理（类型 / 额度 / 申请单）
 *   - 报销申请管理
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
    },
    children: [
      {
        path: "definitions",
        name: "OaWorkflowDefinition",
        meta: {
          order: 1,
          icon: "lucide:file-work",
          title: "routes.oa.definition",
        },
        component: () => import("@/pages/app/oa/definition/index.vue"),
      },
      {
        path: "approvals",
        name: "OaApprovalCenter",
        meta: {
          order: 2,
          icon: "lucide:stamp",
          title: "routes.oa.approval",
        },
        component: () => import("@/pages/app/oa/approval/index.vue"),
      },
      {
        path: "attendance-records",
        name: "OaAttendanceRecords",
        meta: {
          order: 3,
          icon: "lucide:calendar-check",
          title: "routes.oa.attendanceRecords",
        },
        component: () => import("@/pages/app/oa/attendance/records.vue"),
      },
      {
        path: "holidays",
        name: "OaHolidays",
        meta: {
          order: 3,
          icon: "lucide:calendar-days",
          title: "routes.oa.holidays",
        },
        component: () => import("@/pages/app/oa/attendance/holidays.vue"),
      },
      {
        path: "leave",
        name: "OaLeaveManagement",
        meta: {
          order: 4,
          icon: "lucide:plane-takeoff",
          title: "routes.oa.leave",
        },
        component: () => import("@/pages/app/oa/leave/index.vue"),
      },
      {
        path: "expense",
        name: "OaExpenseManagement",
        meta: {
          order: 5,
          icon: "lucide:receipt",
          title: "routes.oa.expense",
        },
        component: () => import("@/pages/app/oa/expense/index.vue"),
      },
      {
        path: "business-trip",
        name: "OaBusinessTripManagement",
        meta: {
          order: 6,
          icon: "lucide:suitcase",
          title: "routes.oa.businessTrip",
        },
        component: () => import("@/pages/app/oa/business_trip/index.vue"),
      },
      {
        path: "overtime",
        name: "OaOvertimeManagement",
        meta: {
          order: 7,
          icon: "lucide:clock-alert",
          title: "routes.oa.overtime",
        },
        component: () => import("@/pages/app/oa/overtime/index.vue"),
      },
      {
        path: "seal-application",
        name: "OaSealApplicationManagement",
        meta: {
          order: 8,
          icon: "lucide:stamp",
          title: "routes.oa.sealApplication",
        },
        component: () => import("@/pages/app/oa/seal_application/index.vue"),
      },
      {
        path: "outing",
        name: "OaOutingManagement",
        meta: {
          order: 9,
          icon: "lucide:door-open",
          title: "routes.oa.outing",
        },
        component: () => import("@/pages/app/oa/outing/index.vue"),
      },
    ],
  },
];

export default oa;
