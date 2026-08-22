import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const system: RouteRecordRaw[] = [
  {
    path: "/system",
    name: "System",
    component: Layout,
    redirect: "/system/menus",
    meta: {
      order: 2005,
      icon: "lucide:settings",
      title: "routes.system.moduleName",
      keepAlive: true,
    },
    children: [
      {
        path: "dict",
        name: "DictManagement",
        meta: {
          order: 3,
          icon: "lucide:library-big",
          title: "routes.system.dict",
        },
        component: () => import("@/pages/app/system/dict/index.vue"),
      },

      {
        path: "files",
        name: "FileManagement",
        meta: {
          order: 4,
          icon: "lucide:file-search",
          title: "routes.system.file",
        },
        component: () => import("@/pages/app/system/file/index.vue"),
      },

      {
        path: "tasks",
        name: "TaskManagement",
        meta: {
          order: 5,
          icon: "lucide:list-todo",
          title: "routes.system.task",
        },
        component: () => import("@/pages/app/system/task/index.vue"),
      },

      {
        path: "login-policies",
        name: "LoginPolicyManagement",
        meta: {
          order: 6,
          icon: "lucide:shield-x",
          title: "routes.system.loginPolicy",
        },
        component: () => import("@/pages/app/system/login_policy/index.vue"),
      },

      {
        path: "languages",
        name: "LanguageManagement",
        meta: {
          order: 7,
          icon: "lucide:globe",
          title: "routes.system.language",
        },
        component: () => import("@/pages/app/system/language/index.vue"),
      },
    ],
  },
];

export default system;
