import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const permission: RouteRecordRaw[] = [
  {
    path: "/permission",
    name: "PermissionManagement",
    component: Layout,
    redirect: "/permission/codes",
    meta: {
      order: 2002,
      icon: "lucide:shield-check",
      title: "routes.permission.moduleName",
      keepAlive: true,
    },
    children: [
      {
        path: "codes",
        name: "PermissionPointManagement",
        meta: {
          order: 1,
          icon: "lucide:shield-ellipsis",
          title: "routes.permission.code",
        },
        component: () => import("@/pages/app/permission/permission/index.vue"),
      },

      {
        path: "menus",
        name: "MenuManagement",
        meta: {
          order: 2,
          icon: "lucide:square-menu",
          title: "routes.permission.menu",
        },
        component: () => import("@/pages/app/permission/menu/index.vue"),
      },

      {
        path: "apis",
        name: "APIManagement",
        meta: {
          order: 3,
          icon: "lucide:route",
          title: "routes.permission.api",
        },
        component: () => import("@/pages/app/permission/api/index.vue"),
      },

      {
        path: "roles",
        name: "RoleManagement",
        meta: {
          order: 4,
          icon: "lucide:shield-user",
          title: "routes.permission.role",
        },
        component: () => import("@/pages/app/permission/role/index.vue"),
      },
    ],
  },
];

export default permission;
