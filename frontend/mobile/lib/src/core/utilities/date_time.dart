import 'package:intl/intl.dart';

/// 日期时间工具集
class DateTimeUtils {
  DateTimeUtils._(); // Private constructor to prevent instantiation

  static int ntpOffset = 0;

  static init() async {
    // NTP对时
    // DateTimeUtil.refreshNTPOffset();
  }

  /// 刷新NTF时间偏移量
  static refreshNTPOffset() async {
    // final int offset = await NTP.getNtpOffset(localTime: DateTime.now(), lookUpAddress: Environments.ntpHost);
    // ntpOffset = offset;
  }

  /// 校准过的当前时间
  static DateTime now({bool forceSync = false}) {
    if (forceSync) {
      refreshNTPOffset();
    }
    DateTime internetTime = DateTime.now().add(Duration(milliseconds: ntpOffset));
    return internetTime;
  }

  /// 当前时间的毫秒时间戳
  static int currentTimeMillis() {
    return now().millisecondsSinceEpoch;
  }

  /// 当前utc时间戳
  static int utc() {
    return now().toUtc().millisecondsSinceEpoch;
  }

  /// UTC时间的秒时间戳
  static int utcSecond() {
    return utc() ~/ 1000;
  }

  /// 将服务端返回的日期时间字符串格式化为当前 locale 下的展示文案。
  ///
  /// 后端日期字段统一以 RFC3339（如 `2026-08-23T15:30:00Z`）下发，此处由
  /// [DateTime.parse] 解析后用 [DateFormat] 按当前 locale 渲染，避免直接
  /// `.split('T')` / `.toString()` 漏出原始格式或英文月份。locale 由
  /// `flutter_intl` 在 `S.load` 中通过 `Intl.defaultLocale` 设定，`DateFormat`
  /// 默认构造即沿用该 locale，无需显式传参。
  ///
  /// 解析失败或入参为空时返回 `-`，绝不向上抛。
  static String formatDateTime(String? raw) {
    if (raw == null || raw.isEmpty) return '-';
    try {
      final dt = DateTime.parse(raw);
      return DateFormat.yMd().add_Hm().format(dt.toLocal());
    } catch (_) {
      return '-';
    }
  }

  /// [DateTime] 入参的重载，用于本地产出的日期（如 [showDatePicker] 回填值）。
  static String formatDateTimeDt(DateTime? dt) {
    if (dt == null) return '-';
    try {
      return DateFormat.yMd().add_Hm().format(dt);
    } catch (_) {
      return '-';
    }
  }

  /// 仅日期（无时间）的 locale 化展示，用于日期范围、请假起止等不含时分秒的字段。
  static String formatDate(String? raw) {
    if (raw == null || raw.isEmpty) return '-';
    try {
      final dt = DateTime.parse(raw);
      return DateFormat.yMd().format(dt.toLocal());
    } catch (_) {
      return '-';
    }
  }

  /// [DateTime] 入参的重载，用于本地产出的日期（如 [showDatePicker] 回填值）。
  static String formatDateDt(DateTime? dt) {
    if (dt == null) return '-';
    try {
      return DateFormat.yMd().format(dt);
    } catch (_) {
      return '-';
    }
  }
}
