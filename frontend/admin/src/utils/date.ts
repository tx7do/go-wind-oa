import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import timezone from "dayjs/plugin/timezone";
import utc from "dayjs/plugin/utc";

dayjs.extend(utc);
dayjs.extend(timezone);
dayjs.extend(relativeTime);

const dateUtil = dayjs;

// 统一以上海时区展示时间。注意：setDefault 仅影响 dayjs.tz(input) 的解析，
// 不影响 dayjs(input)（按浏览器本地时区）。因此 formatDate 内必须使用
// dateUtil.tz(...) 才能让该默认时区真正生效，否则跨时区客户端会显示偏差。
const DEFAULT_TZ = "Asia/Shanghai";
dayjs.tz.setDefault(DEFAULT_TZ);
dateUtil.tz.setDefault(DEFAULT_TZ);

export { dateUtil };

export function formatDate(time?: number | string | Date | null, format = "YYYY-MM-DD") {
  if (time === null || time === undefined || time === "") {
    return "";
  }
  if (isDate(time)) {
    return dateUtil.tz(time, DEFAULT_TZ).format(format);
  }

  try {
    // 带时区标记（Z / ±hh:mm）的字符串表示绝对时刻：先按原时刻解析，再转换为
    // 展示时区（dayjs.tz(str, tz) 对带偏移的串按墙钟处理，不会做时刻换算）。
    // 不带时区的串（如本地拼接的 "YYYY-MM-DD"）仍按展示时区墙钟解析。
    const hasTzDesignator =
      typeof time === "string" && /(z|[+-]\d{2}:?\d{2})$/i.test(time.trim());
    const date = hasTzDesignator
      ? dateUtil(time).tz(DEFAULT_TZ)
      : dateUtil.tz(time, DEFAULT_TZ);
    if (!date.isValid()) {
      // throw new Error('Invalid date');
      return "";
    }
    return date.format(format);
  } catch (error) {
    console.error(`Error formatting date: ${error}`);
    return time;
  }
}

export function formatDateTime(time?: number | string | Date | null) {
  if (time === null || time === undefined || time === "") {
    return "";
  }
  return formatDate(time, "YYYY-MM-DD HH:mm:ss");
}

export function isDate(value: any): value is Date {
  return value instanceof Date;
}

export function isDayjsObject(value: any): value is dayjs.Dayjs {
  return dateUtil.isDayjs(value);
}
