/**
 * 通知中心逻辑。
 *
 * 收件箱经 admin BFF 的 InternalMessageRecipientService.ListUserInbox 拉取当前
 * 管理员收件箱（收件人过滤在 core 侧按 viewer 强制）。BFF 在返回前已为每行回填
 * 标题/正文（GetMessage），故列表项无需前端再逐条请求。
 *
 * 实时刷新：admin 登录后 globalSSEClient 已连接（见 use-token-refresh.ts），后端
 * 在自身 SendMessage 路径写 recipient 后向收件人的 admin 会话推 "notification"
 * 事件。此处订阅该事件以刷新列表。注意：工作流引擎经 core 进程内 SendMessage 落库
 * 的通知不触发 SSE（core gRPC-only 无 SSE server），此类通知需等下次手动/页面
 * 重载拉取——见 docs/oa-workflow-design.md §5.5、§12。
 *
 * 已读/删除经 MarkNotificationAsRead / DeleteNotificationFromInbox（recipientIds
 * 为收件人行 ID，userId 留空由 core viewer 侧校验所有权）。
 */
import { onMounted, onUnmounted, ref } from "vue";
import { apiClient } from "@/api/client";
import { globalSSEClient } from "@/core/transport/sse";
import type { SSEEventHandler } from "@/core/transport/sse";

// BFF 运行时回填的收件箱行（title/content 不在 TS 类型里，故本地声明）。
type InboxRow = {
  id?: number;
  messageId?: number;
  title?: string;
  content?: string;
  createdAt?: string;
  readAt?: string;
};

// 详情视图（供弹窗渲染，字段映射自 GetMessage 返回的 InternalMessage）。
type NoticeDetail = {
  title?: string;
  content?: string;
  publisherName?: string;
  publishTime?: string;
};

export function useNotice() {
  const list = ref<InboxRow[]>([]);
  const unreadTotal = ref(0);
  const detail = ref<NoticeDetail | null>(null);
  const dialogVisible = ref(false);

  async function fetchList() {
    try {
      const resp = (await apiClient.internalMessageRecipientService.ListUserInbox({
        page: 1,
        pageSize: 20,
        sorting: undefined,
      })) as { items?: InboxRow[] };
      const rows = resp.items ?? [];
      list.value = rows;
      unreadTotal.value = rows.reduce(
        (n, r) => n + (r.readAt ? 0 : 1),
        0
      );
    } catch {
      list.value = [];
      unreadTotal.value = 0;
    }
  }

  // SSE "notification" 事件：后端向本会话推了新收件行，刷新列表。
  const onNotification: SSEEventHandler<unknown> = () => {
    fetchList();
  };

  async function read(item: InboxRow) {
    if (item.id == null) return;
    try {
      await apiClient.internalMessageRecipientService.MarkNotificationAsRead({
        recipientIds: [item.id],
        userId: undefined,
      });
    } catch {
      /* 标记失败不阻塞详情查看 */
    }
    await fetchList();

    // 详情弹窗：单条 GetMessage 取完整消息（含 title/senderName/createdAt）。
    if (item.messageId == null) {
      detail.value = null;
      return;
    }
    try {
      const msg = (await apiClient.internalMessageService.GetMessage({
        id: item.messageId,
      })) as {
        title?: string;
        content?: string;
        senderName?: string;
        createdAt?: string;
      };
      detail.value = {
        title: msg.title,
        content: msg.content,
        publisherName: msg.senderName,
        publishTime: msg.createdAt,
      };
      dialogVisible.value = true;
    } catch {
      detail.value = null;
    }
  }

  async function readAll() {
    const ids = list.value
      .filter((r) => !r.readAt && r.id != null)
      .map((r) => r.id as number);
    if (ids.length === 0) return;
    try {
      await apiClient.internalMessageRecipientService.MarkNotificationAsRead({
        recipientIds: ids,
        userId: undefined,
      });
      await fetchList();
    } catch {
      /* 静默 */
    }
  }

  async function clearAll() {
    const ids = list.value
      .filter((r) => r.id != null)
      .map((r) => r.id as number);
    if (ids.length === 0) return;
    try {
      await apiClient.internalMessageRecipientService
        .DeleteNotificationFromInbox({ recipientIds: ids, userId: undefined });
      await fetchList();
    } catch {
      /* 静默 */
    }
  }

  function goMore() {
    // 无独立通知管理页，暂为 no-op。
  }

  onMounted(() => {
    fetchList();
    globalSSEClient.on("notification", onNotification);
  });

  onUnmounted(() => {
    globalSSEClient.off("notification", onNotification);
  });

  return {
    list,
    unreadTotal,
    detail,
    dialogVisible,
    fetchList,
    read,
    readAll,
    clearAll,
    goMore,
  };
}
