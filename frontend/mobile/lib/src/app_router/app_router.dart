import 'dart:async';

import 'package:flutter/material.dart';
import 'package:get_it/get_it.dart';
import 'package:go_router/go_router.dart';

import 'package:flutter_app/src/core/constants/index.dart' as constants;
import 'package:flutter_app/src/core/widgets/not_found_page.dart';
import 'package:flutter_app/src/core/repositories/user_auth_cache.dart';
import 'package:flutter_app/src/app_router/route_names.dart';
import 'package:flutter_app/src/features/auth/pages/login_page.dart';
import 'package:flutter_app/src/features/oa/pages/shell/oa_shell_page.dart';
import 'package:flutter_app/src/features/oa/pages/task_list/oa_task_list_page.dart';
import 'package:flutter_app/src/features/oa/pages/task_detail/oa_task_detail_page.dart';
import 'package:flutter_app/src/features/oa/pages/submit_apply/oa_submit_apply_page.dart';
import 'package/flutter_app/src/features/oa/pages/notifications/oa_notifications_page.dart';
import 'package:flutter_app/src/features/oa/pages/attendance/oa_attendance_page.dart';

/// OA 移动端路由。
///
/// 结构：login（非 Shell）+ ShellRoute（底部导航三个 Tab：工作流 / 通知 / 考勤）
/// + 工作流详情子路由（Shell 内）+ 提交申请（非 Shell）。
///
/// 鉴权：[OaGuard] 依 [UserAuthCache.hasLogin] 判定，未登录重定向到 /login。
/// 已登录用户访问 /login 重定向到 /（工作流 Tab）。
class AppRouter {
  static const initial = constants.AppRoutePath.initial;

  static final router = GoRouter(
    initialLocation: initial,
    redirect: _guard,
    errorBuilder: (context, state) => const NotFoundPage(),
    routes: [
      // ── Shell：底部导航 ─────────────────────────────
      ShellRoute(
        builder: (context, state, child) {
          return OaShellPage(
            currentRoute: state.uri.path,
            child: child,
          );
        },
        routes: [
          GoRoute(
            name: RouteNames.oaTaskList,
            path: constants.AppRoutePath.oaTaskList,
            builder: (context, state) {
              return const OaTaskListPage();
            },
            routes: [
              GoRoute(
                name: RouteNames.oaTaskDetail,
                path: 'detail/:id',
                builder: (context, state) {
                  final taskId =
                      int.tryParse(state.pathParameters['id'] ?? '0') ?? 0;
                  return OaTaskDetailPage(taskId: taskId);
                },
              ),
            ],
          ),
          GoRoute(
            name: RouteNames.oaNotifications,
            path: constants.AppRoutePath.oaNotifications,
            builder: (context, state) {
              return const OaNotificationsPage();
            },
          ),
          GoRoute(
            name: RouteNames.oaAttendance,
            path: constants.AppRoutePath.oaAttendance,
            builder: (context, state) {
              return const OaAttendancePage();
            },
          ),
        ],
      ),
      // ── 提交申请（非 Shell，全屏表单） ───────────────
      GoRoute(
        name: RouteNames.oaSubmitApply,
        path: constants.AppRoutePath.oaSubmitApply,
        builder: (context, state) {
          return const OaSubmitApplyPage();
        },
      ),
      // ── 登录（非 Shell） ────────────────────────────
      GoRoute(
        name: RouteNames.login,
        path: constants.AppRoutePath.login,
        builder: (context, state) {
          return const LoginPage();
        },
      ),
    ],
  );

  /// 路由守卫：未登录 → /login；已登录访问 /login → /。
  static FutureOr<String?> _guard(BuildContext context, GoRouterState state) {
    final authCache = GetIt.instance<UserAuthCache>();
    final loggedIn = authCache.hasLogin;
    final isLoginRoute = state.matchedLocation == constants.AppRoutePath.login;

    if (!loggedIn && !isLoginRoute) {
      return constants.AppRoutePath.login;
    }
    if (loggedIn && isLoginRoute) {
      return constants.AppRoutePath.initial;
    }
    return null;
  }
}
