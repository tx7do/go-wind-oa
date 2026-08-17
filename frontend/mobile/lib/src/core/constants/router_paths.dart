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
}
