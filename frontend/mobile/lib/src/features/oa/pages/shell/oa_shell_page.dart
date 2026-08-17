import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'package:flutter_app/src/core/constants/index.dart' as constants;
import 'package:flutter_app/src/app_router/route_names.dart';
import 'package:flutter_app/generated/l10n.dart';

/// OA 移动端 Shell：底部导航三 Tab（工作流 / 通知 / 考勤）。
///
/// 当前路由由 [currentRoute] 提供；底部导航根据它高亮对应 Tab，点击切路由。
/// child 由 ShellRoute 注入，是当前 Tab 的页面。Shell 不持有任何业务状态。
class OaShellPage extends StatelessWidget {
  final String currentRoute;
  final Widget child;

  const OaShellPage({
    super.key,
    required this.currentRoute,
    required this.child,
  });

  int _currentIndex() {
    switch (currentRoute) {
      case constants.AppRoutePath.oaTaskList:
        return 0;
      case constants.AppRoutePath.oaNotifications:
        return 1;
      case constants.AppRoutePath.oaAttendance:
        return 2;
      default:
        // 工作流详情子路由（/oa/tasks/detail/:id）属工作流 Tab 的下钻，
        // 高亮工作流 Tab。
        if (currentRoute.startsWith('${constants.AppRoutePath.oaTaskList}/')) {
          return 0;
        }
        return 0;
    }
  }

  void _onTap(int index) {
    switch (index) {
      case 0:
        GoRouter.of(context).goNamed(RouteNames.oaTaskList);
        break;
      case 1:
        GoRouter.of(context).goNamed(RouteNames.oaNotifications);
        break;
      case 2:
        GoRouter.of(context).goNamed(RouteNames.oaAttendance);
        break;
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final loc = S.of(context);

    return Scaffold(
      body: child,
      bottomNavigationBar: NavigationBar(
        selectedIndex: _currentIndex(),
        onDestinationSelected: _onTap,
        backgroundColor: theme.colorScheme.surface,
        destinations: [
          NavigationDestination(
            icon: const Icon(Icons.assignment_outlined),
            selectedIcon: const Icon(Icons.assignment),
            label: loc.oaTabWorkflow,
          ),
          NavigationDestination(
            icon: const Icon(Icons.notifications_outlined),
            selectedIcon: const Icon(Icons.notifications),
            label: loc.oaTabNotifications,
          ),
          NavigationDestination(
            icon: const Icon(Icons.location_on_outlined),
            selectedIcon: const Icon(Icons.location_on),
            label: loc.oaTabAttendance,
          ),
        ],
      ),
    );
  }
}
