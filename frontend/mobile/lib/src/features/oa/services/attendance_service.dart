/// 考勤服务（骨架）。
///
/// 后端待办（详见 docs/oa-mobile-design.md §"后端待办"）：
///  - 考勤服务接口（打卡记录、班次规则）；
///  - 地理围栏库（公司方圆 N 米的多边形/半径定义）；
///  - 公司 Wi-Fi 指纹库（SSID/BSSID 白名单）。
///
/// 当前实现：[checkIn] 永远返回 [CheckInResult.notConfigured]；
/// [isInFence] 永远返回 false。UI 据此展示"考勤服务未配置"。
class AttendanceService {
  AttendanceService._();
}

enum CheckInResult { ok, notInFence, notConfigured }

/// 占位打卡实现。地理围栏判定硬编码 false。
Future<CheckInResult> checkIn() async => CheckInResult.notConfigured;

/// 占位围栏判定。硬编码 false。
Future<bool> isInFence() async => false;
