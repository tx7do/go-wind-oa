/**
 * 字典缓存 Composable
 *
 * OA 后端无字典服务（依赖 cms dictservice，OA 未含）。全部方法为 no-op stub：
 * fetchAllDictEntries 空操作，缓存恒空；getDictEntries* 系列返回空数组 / 原值，
 * 保证引用此模块的页面不崩。
 */
import { ref } from "vue";

// ==============================
// 字典项缓存（模块级单例，恒空）
// ==============================

const dictEntryCache = ref<Record<string, any[]>>({});

// ==============================
// 缓存管理
// ==============================

/**
 * 获取所有字典项并缓存。
 * OA 无字典服务，no-op。
 */
export async function fetchAllDictEntries() {
  return;
}

/**
 * 获取指定 typeCode 的字典项列表。
 * OA 无字典服务，返回空数组。
 */
export function getDictEntriesByTypeCode(_typeCode: string): any[] {
  return [];
}

/**
 * 获取指定 typeCode 的字典项选项（label/value 格式）。
 * OA 无字典服务，返回空数组。
 */
export function getDictEntriesOptionsByTypeCode(_typeCode: string): { label: string; value: string }[] {
  return [];
}

/**
 * 重置缓存。
 */
export function resetCache() {
  dictEntryCache.value = {};
}

// ==============================
// 工具函数
// ==============================

/**
 * 获取字典项标签。
 * OA 无字典服务，返回空串。
 */
export function getDictEntryLabel(_row: any): string {
  return "";
}

/**
 * 通过字典项值获取字典项标签。
 * OA 无字典服务，原样返回 value。
 */
export function getDictEntryLabelByValue(value?: string, _dictEntries?: any[]): string {
  return value ?? "";
}
