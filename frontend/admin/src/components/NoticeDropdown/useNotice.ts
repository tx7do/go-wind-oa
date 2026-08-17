/**
 * 通知中心逻辑（OA stub）。
 *
 * OA 后端无 internal_message 收件箱接口（依赖 cms internal_message 服务，
 * OA 未含）。全部方法为 no-op，list / unreadTotal 恒空。NoticeDropdown 组件
 * 据此展示空通知状态。SSE 推送对接见 docs/oa-mobile-design.md §5.3。
 */
import { ref } from "vue";

export function useNotice() {
  const list = ref<any[]>([]);
  const unreadTotal = ref(0);
  const detail = ref<any | null>(null);
  const dialogVisible = ref(false);

  async function fetchList() {
    list.value = [];
    unreadTotal.value = 0;
  }

  async function read(_item: { id?: number; messageId?: number }) {
    // no-op
  }

  async function readAll() {
    // no-op
  }

  async function clearAll() {
    // no-op
  }

  function goMore() {
    // no-op
  }

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
