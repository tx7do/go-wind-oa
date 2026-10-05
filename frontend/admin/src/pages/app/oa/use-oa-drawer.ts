import { ref } from "vue";

/**
 * OA 手写抽屉的通用开关状态。
 *
 * 收拢各抽屉重复的 visible/open/close 样板（ProModal v-model:visible +
 * defineExpose({ open }) 模式）。open 的附加副作用（重置表单、拉详情等）
 * 通过 onOpen 回调注入，open 本身的参数原样透传。
 *
 * @example
 * ```ts
 * const { visible, open, close } = useOaDrawer(() => resetForm());
 * defineExpose({ open });
 * ```
 */
export function useOaDrawer(onOpen?: (...args: any[]) => void) {
  const visible = ref(false);

  function open(...args: any[]) {
    onOpen?.(...args);
    visible.value = true;
  }

  function close() {
    visible.value = false;
  }

  return { visible, open, close };
}
