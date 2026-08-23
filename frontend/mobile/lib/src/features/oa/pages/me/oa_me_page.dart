import 'package:flutter/material.dart';

import 'package:flutter_app/generated/l10n.dart';
import 'package:flutter_app/src/app_router/route_names.dart';
import 'package:flutter_app/src/app_router/app_router.dart' show AppRouter;
import 'package:flutter_app/src/core/constants/index.dart' as constants;
import 'package:flutter_app/src/core/transport/http/status.dart';
import 'package:flutter_app/src/features/auth/services/authentication_service.dart';
import 'package:flutter_app/src/features/oa/services/user_profile_service.dart';
import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;
import 'package:go_router/go_router.dart';

/// “我的”页（底部导航第四 Tab）。
///
/// 顶部展示当前用户头像与昵称（经 [UserProfileService.getUser] 拉取），
/// 下方为入口列表：个人信息（查看/编辑自身资料）、设置（外观与语言偏好）、
/// 退出登录。本页不持业务状态，仅作入口聚合。
class OaMePage extends StatefulWidget {
  const OaMePage({super.key});

  @override
  State<OaMePage> createState() => _OaMePageState();
}

class _OaMePageState extends State<OaMePage> {
  final _service = UserProfileService();
  oaApi.IdentityServiceV1User? _user;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final result = await _service.getUser();
    if (!mounted) return;
    setState(() {
      _user = (result is Status) ? null : (result as oaApi.IdentityServiceV1User?);
      _loading = false;
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final loc = S.of(context);

    return Scaffold(
      appBar: AppBar(
        backgroundColor: theme.colorScheme.surface,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        title: Text(loc.me, style: const TextStyle(fontWeight: FontWeight.bold)),
      ),
      body: ListView(
        children: [
          _buildHeader(theme),
          const Divider(height: 1),
          ListTile(
            leading: const Icon(Icons.person_outline),
            title: Text(loc.profileTitle),
            trailing: const Icon(Icons.chevron_right, color: Colors.grey),
            onTap: () => GoRouter.of(context).goNamed(RouteNames.oaProfile),
          ),
          ListTile(
            leading: const Icon(Icons.settings_outlined),
            title: Text(loc.settings),
            subtitle: Text(loc.themeLanguagePrefs),
            trailing: const Icon(Icons.chevron_right, color: Colors.grey),
            onTap: () => GoRouter.of(context).goNamed(RouteNames.oaSettings),
          ),
          const Divider(height: 1),
          ListTile(
            leading: const Icon(Icons.logout, color: Colors.red),
            title: Text(loc.logout, style: const TextStyle(color: Colors.red)),
            onTap: _confirmLogout,
          ),
        ],
      ),
    );
  }

  /// 顶部用户信息卡片：头像 + 昵称。
  Widget _buildHeader(ThemeData theme) {
    if (_loading) {
      return const Padding(
        padding: EdgeInsets.all(32),
        child: Center(child: CircularProgressIndicator()),
      );
    }
    final nickname = _user?.nickname ?? _user?.username ?? '';
    return Container(
      padding: const EdgeInsets.fromLTRB(20, 28, 20, 28),
      color: theme.colorScheme.surface,
      child: Row(
        children: [
          CircleAvatar(
            radius: 32,
            backgroundColor: theme.colorScheme.primaryContainer,
            backgroundImage:
                _user?.avatar != null && _user!.avatar!.isNotEmpty
                    ? NetworkImage(_user!.avatar!)
                    : null,
            child: _user?.avatar == null || _user!.avatar!.isEmpty
                ? Text(
                    nickname.isNotEmpty ? nickname.characters.first : '?',
                    style: TextStyle(fontSize: 24, color: theme.colorScheme.onPrimaryContainer),
                  )
                : null,
          ),
          const SizedBox(width: 16),
          Expanded(
            child: Text(
              nickname,
              style: theme.textTheme.titleMedium,
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ],
      ),
    );
  }

  /// 退出登录确认弹窗。
  Future<void> _confirmLogout() async {
    final loc = S.of(context);
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(loc.logout),
        content: Text(loc.logoutConfirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(loc.cancel),
          ),
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(loc.confirm),
          ),
        ],
      ),
    );
    if (confirmed != true) return;
    final authService = AuthenticationService();
    await authService.clearTokens();
    AppRouter.router.go(constants.AppRoutePath.login);
  }
}
