/// 应用路由的路径定义
class AppRoutePath {
  AppRoutePath._(); // Private constructor to prevent instantiation

  static const initial = '/';

  static const login = '/login';
  static const signIn = '/sign_in';
  static const signUp = '/sign_up';

  static const notFound = '/not_found';

  // OA 工作流（移动端）
  static const oaTaskList = '/oa/tasks';
  static const oaSubmitApply = '/oa/apply';
  static const oaNotifications = '/oa/notifications';
  static const oaAttendance = '/oa/attendance';
  static const oaLeave = '/oa/leave';
  static const oaExpense = '/oa/expense';
  static const oaBusinessTrip = '/oa/business-trip';
  static const oaOvertime = '/oa/overtime';
  static const oaSealApplication = '/oa/seal-application';
  static const oaOuting = '/oa/outing';
}
