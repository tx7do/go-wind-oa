import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const opm: RouteRecordRaw[] = [
  {
    path: "/opm",
    name: "OrganizationalPersonnelManagement",
    component: Layout,
    redirect: "/opm/users",
    meta: {
      order: 2001,
      icon: "lucide:users",
      title: "routes.opm.moduleName",
      keepAlive: true,
    },
    children: [
      {
        path: "org-units",
        name: "OrgUnitManagement",
        meta: {
          order: 1,
          icon: "lucide:layers",
          title: "routes.opm.orgUnit",
        },
        component: () => import("@/pages/app/opm/org_unit/index.vue"),
      },

      {
        path: "positions",
        name: "PositionManagement",
        meta: {
          order: 2,
          icon: "lucide:briefcase",
          title: "routes.opm.position",
        },
        component: () => import("@/pages/app/opm/position/index.vue"),
      },

      {
        path: "users",
        name: "UserManagement",
        meta: {
          order: 3,
          icon: "lucide:user",
          title: "routes.opm.user",
        },
        component: () => import("@/pages/app/opm/user/list/index.vue"),
      },
      {
        path: "users/detail/:id",
        name: "UserDetail",
        meta: {
          hideInMenu: true,
          title: "routes.opm.userDetail",
        },
        component: () => import("@/pages/app/opm/user/detail/index.vue"),
      },
      {
        path: "profile",
        name: "UserProfile",
        component: () => import("@/pages/app/opm/user/profile/index.vue"),
        meta: {
          title: "routes.profile.settings",
          icon: "lucide:user-pen",
          hideInMenu: true,
        },
      },
    ],
  },
];

export default opm;
