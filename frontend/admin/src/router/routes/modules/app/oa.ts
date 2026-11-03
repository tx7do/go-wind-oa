import type { RouteRecordRaw } from "vue-router";

import { Layout } from "@/layouts";

/**
 * OA 业务路由模块——按业务域拆为四个平级菜单分组（对齐上游 base 的两级菜单模式，
 * 不做三级嵌套）：
 *
 *   审批管理  /approval    审批中心、流程定义、审批委托
 *   考勤管理  /attendance  记录、设置、节假日、地理围栏、Wi-Fi 指纹
 *   假期管理  /leave       请假申请、类型配置、额度授予
 *   行政申请  /affairs     报销、出差、加班、用印、外出
 *
 * 约定：
 * - 路由 name 保持历史名（OaXxx）不变，避免破坏 KeepAlive include 与命名跳转；
 * - 同级 order 全局唯一（业务区 1-5、平台区 2001+），杜绝撞号导致的排序漂移；
 * - 不设 meta.authority：OA v1 后端无角色/权限表（authz 为 noop），登录即可访问；
 *   待 identity/permission 域落地后再恢复 authority 过滤。
 */
const approval: RouteRecordRaw[] = [
  {
    path: "/approval",
    name: "OaApprovalModule",
    redirect: "/approval/center",
    component: Layout,
    meta: {
      order: 1,
      icon: "lucide:clipboard-check",
      title: "routes.approval.moduleName",
      keepAlive: true,
    },
    children: [
      {
        path: "center",
        name: "OaApprovalCenter",
        meta: {
          order: 1,
          icon: "lucide:clipboard-check",
          title: "routes.oa.approval",
        },
        component: () => import("@/pages/app/oa/approval/index.vue"),
      },
      {
        path: "definitions",
        name: "OaWorkflowDefinition",
        meta: {
          order: 2,
          icon: "lucide:file-cog",
          title: "routes.oa.definition",
        },
        component: () => import("@/pages/app/oa/definition/index.vue"),
      },
      {
        path: "delegations",
        name: "OaWorkflowDelegation",
        meta: {
          order: 3,
          icon: "lucide:user-check",
          title: "routes.oa.delegation",
        },
        component: () => import("@/pages/app/oa/delegation/index.vue"),
      },
    ],
  },
];

const attendance: RouteRecordRaw[] = [
  {
    path: "/attendance",
    name: "OaAttendanceModule",
    redirect: "/attendance/records",
    component: Layout,
    meta: {
      order: 2,
      icon: "lucide:calendar-check",
      title: "routes.attendance.moduleName",
      keepAlive: true,
    },
    children: [
      {
        path: "records",
        name: "OaAttendanceRecords",
        meta: {
          order: 1,
          icon: "lucide:calendar-check",
          title: "routes.oa.attendanceRecords",
        },
        component: () => import("@/pages/app/oa/attendance/records.vue"),
      },
      {
        path: "setting",
        name: "OaAttendanceSetting",
        meta: {
          order: 2,
          icon: "lucide:settings",
          title: "routes.oa.attendanceSetting",
        },
        component: () => import("@/pages/app/oa/attendance/setting.vue"),
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
        path: "fences",
        name: "OaAttendanceFences",
        meta: {
          order: 4,
          icon: "lucide:map-pin",
          title: "routes.oa.attendanceFence",
        },
        component: () => import("@/pages/app/oa/attendance/fences.vue"),
      },
      {
        path: "wifis",
        name: "OaAttendanceWifis",
        meta: {
          order: 5,
          icon: "lucide:wifi",
          title: "routes.oa.attendanceWifi",
        },
        component: () => import("@/pages/app/oa/attendance/wifis.vue"),
      },
    ],
  },
];

const leave: RouteRecordRaw[] = [
  {
    path: "/leave",
    name: "OaLeaveModule",
    redirect: "/leave/applications",
    component: Layout,
    meta: {
      order: 3,
      icon: "lucide:plane-takeoff",
      title: "routes.leave.moduleName",
      keepAlive: true,
    },
    children: [
      {
        path: "applications",
        name: "OaLeaveApplications",
        meta: {
          order: 1,
          icon: "lucide:plane-takeoff",
          title: "routes.oa.leaveApplications",
        },
        component: () => import("@/pages/app/oa/leave/applications.vue"),
      },
      {
        path: "types",
        name: "OaLeaveTypes",
        meta: {
          order: 2,
          icon: "lucide:tags",
          title: "routes.oa.leaveTypes",
        },
        component: () => import("@/pages/app/oa/leave/types.vue"),
      },
      {
        path: "balances",
        name: "OaLeaveBalances",
        meta: {
          order: 3,
          icon: "lucide:wallet",
          title: "routes.oa.leaveBalances",
        },
        component: () => import("@/pages/app/oa/leave/balances.vue"),
      },
    ],
  },
];

const affairs: RouteRecordRaw[] = [
  {
    path: "/affairs",
    name: "OaAffairsModule",
    redirect: "/affairs/expense",
    component: Layout,
    meta: {
      order: 4,
      icon: "lucide:receipt",
      title: "routes.affairs.moduleName",
      keepAlive: true,
    },
    children: [
      {
        path: "expense",
        name: "OaExpenseManagement",
        meta: {
          order: 1,
          icon: "lucide:receipt",
          title: "routes.oa.expense",
        },
        component: () => import("@/pages/app/oa/expense/index.vue"),
      },
      {
        path: "business-trip",
        name: "OaBusinessTripManagement",
        meta: {
          order: 2,
          icon: "lucide:luggage",
          title: "routes.oa.businessTrip",
        },
        component: () => import("@/pages/app/oa/business_trip/index.vue"),
      },
      {
        path: "overtime",
        name: "OaOvertimeManagement",
        meta: {
          order: 3,
          icon: "lucide:clock-alert",
          title: "routes.oa.overtime",
        },
        component: () => import("@/pages/app/oa/overtime/index.vue"),
      },
      {
        path: "seal-application",
        name: "OaSealApplicationManagement",
        meta: {
          order: 4,
          icon: "lucide:stamp",
          title: "routes.oa.sealApplication",
        },
        component: () => import("@/pages/app/oa/seal_application/index.vue"),
      },
      {
        path: "outing",
        name: "OaOutingManagement",
        meta: {
          order: 5,
          icon: "lucide:door-open",
          title: "routes.oa.outing",
        },
        component: () => import("@/pages/app/oa/outing/index.vue"),
      },
    ],
  },
];

export default [...approval, ...attendance, ...leave, ...affairs];
