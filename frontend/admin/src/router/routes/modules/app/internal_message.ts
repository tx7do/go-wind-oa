import type { RouteRecordRaw } from "vue-router";

import { Layout } from "@/layouts";

/**
 * 消息公告模块：站内消息（管理端发送/分类）+ 公告发布。
 * 公告后端复用站内信通道（oa_announcement 落 internal_messages），菜单语义上归并同组。
 */
const internal_message: RouteRecordRaw[] = [
  {
    path: "/internal-message",
    name: "InternalMessageManagement",
    redirect: "/internal-message/messages",
    component: Layout,
    meta: {
      order: 5,
      icon: "lucide:mail",
      title: "routes.internalMessage.moduleName",
      keepAlive: true,
    },
    children: [
      {
        path: "messages",
        name: "InternalMessageList",
        meta: {
          order: 1,
          icon: "lucide:message-circle-more",
          title: "routes.internalMessage.internalMessage",
        },
        component: () => import("@/pages/app/internal_message/message/index.vue"),
      },

      {
        path: "categories",
        name: "InternalMessageCategoryManagement",
        meta: {
          order: 2,
          icon: "lucide:tags",
          title: "routes.internalMessage.internalMessageCategory",
        },
        component: () => import("@/pages/app/internal_message/category/index.vue"),
      },

      {
        path: "announcement",
        name: "OaAnnouncementManagement",
        meta: {
          order: 3,
          icon: "lucide:megaphone",
          title: "routes.oa.announcement",
        },
        component: () => import("@/pages/app/oa/announcement/index.vue"),
      },

      {
        path: "inbox",
        name: "InboxPage",
        component: () => import("@/pages/app/internal_message/inbox/index.vue"),
        meta: {
          title: "routes.profile.internalMessage",
          icon: "lucide:message-circle-more",
          hideInMenu: true,
        },
      },
    ],
  },
];

export default internal_message;
