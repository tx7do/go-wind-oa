/// 路由名称常量
///
/// 集中管理 GoRouter 的 route name，避免硬编码字符串。
/// 仅含 OA 工作流相关路由 + 登录 + 404。
class RouteNames {
  RouteNames._();

  static const String login = 'login';
  static const String notFound = 'not_found';

  // OA 工作流
  static const String oaTaskList = 'oa_task_list';
  static const String oaTaskDetail = 'oa_task_detail';
  static const String oaSubmitApply = 'oa_submit_apply';

  // OA 移动端骨架功能（推送 / 考勤）
  static const String oaNotifications = 'oa_notifications';
  static const String oaAttendance = 'oa_attendance';

  // OA 行政单据（请假 / 报销 / 出差 / 加班 / 用印 / 外出）
  static const String oaLeave = 'oa_leave';
  static const String oaExpense = 'oa_expense';
  static const String oaBusinessTrip = 'oa_business_trip';
  static const String oaOvertime = 'oa_overtime';
  static const String oaSealApplication = 'oa_seal_application';
  static const String oaOuting = 'oa_outing';
  static const String oaDirectory = 'oa_directory';
}
