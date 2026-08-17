import { $t, $te } from "@/core/i18n";
import type { HttpResponse } from "@/core/transport/rest";

/**
 * 创建带 i18n 翻译的错误消息提取函数
 * 按优先级：reason(翻译) → message → code(翻译) → 兜底
 */
export function createI18nGetErrorMsg() {
  const prefix = "common.request.";
  return (error: unknown): string => {
    const errStr = String(error ?? "");
    if (errStr.includes("Network Error")) return $t(prefix + "error.networkError");
    if (
      error &&
      typeof error === "object" &&
      "message" in error &&
      String(error.message).includes("timeout")
    ) {
      return $t(prefix + "error.timeout");
    }

    // 优先从 axios error.response.data 提取，兼容 request() 解包后的业务对象
    let resData: HttpResponse | undefined;
    if (
      error &&
      typeof error === "object" &&
      "response" in error &&
      error.response &&
      typeof error.response === "object" &&
      "data" in error.response
    ) {
      resData = error.response.data as HttpResponse;
    } else if (
      error &&
      typeof error === "object" &&
      ("reason" in error || "code" in error || "message" in error)
    ) {
      // request() 方法 catch 中 throw error.response.data 的场景
      resData = error as unknown as HttpResponse;
    }

    if (!resData) return $t(prefix + "error.unknownError");

    const { reason, message, code } = resData;
    if (reason) {
      const key = `${prefix}reason.${reason}`;
      if ($te(key)) return $t(key);
    }
    if (message?.trim()) return message.trim();
    if (code) {
      const statusKey = `${prefix}status.${code}`;
      if ($te(statusKey)) return $t(statusKey);
    }
    return $t(prefix + "error.unknownError");
  };
}
