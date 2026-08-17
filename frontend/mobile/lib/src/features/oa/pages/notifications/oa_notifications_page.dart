import 'package:flutter/material.dart';

import 'package:flutter_app/generated/l10n.dart';
import 'package:flutter_app/src/features/oa/services/notification_service.dart';

/// 通知收件箱页（骨架）。
///
/// 监听 [NotificationService.notificationStream]；当前 stream 为空，故显示
/// "推送未配置"占位。后端 SSE 通道对接完成后，stream 产出的事件列表将在此渲染。
class OaNotificationsPage extends StatelessWidget {
  const OaNotificationsPage({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final loc = S.of(context);
    final service = NotificationService._();

    return Scaffold(
      appBar: AppBar(
        backgroundColor: theme.colorScheme.surface,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        title: Text(loc.oaNotificationsTitle),
      ),
      body: StreamBuilder<String>(
        stream: service.notificationStream,
        builder: (context, snapshot) {
          if (!snapshot.hasData) {
            return Center(
              child: Padding(
                padding: const EdgeInsets.all(32),
                child: Text(
                  loc.oaNotificationsNotConfigured,
                  textAlign: TextAlign.center,
                  style: TextStyle(
                      color: theme.colorScheme.onSurface.withAlpha(120)),
                ),
              ),
            );
          }
          // 后端对接后此处渲染事件列表
          return const SizedBox.shrink();
        },
      ),
    );
  }
}
