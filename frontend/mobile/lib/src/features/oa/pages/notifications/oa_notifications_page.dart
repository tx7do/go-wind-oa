import 'package:flutter/material.dart';

import 'package:flutter_app/generated/l10n.dart';
import 'package:flutter_app/src/features/oa/services/notification_service.dart';

/// 站内信收件箱页。
///
/// 经 [NotificationService.listMessages] 拉取当前用户收件箱的站内信列表，
/// 与 cms tag_list_page 同构（Future + setState）。审批流转产生的通知经
/// core-service notifyAsync 落 internal_message_recipient 表，此页读取展示。
class OaNotificationsPage extends StatefulWidget {
  const OaNotificationsPage({super.key});

  @override
  State<OaNotificationsPage> createState() => _OaNotificationsPageState();
}

class _OaNotificationsPageState extends State<OaNotificationsPage> {
  final _service = NotificationService();
  List<dynamic> _items = [];
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final result = await _service.listMessages();
    if (!mounted) return;
    setState(() {
      _items = result;
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
        title: Text(loc.oaNotificationsTitle,
            style: const TextStyle(fontWeight: FontWeight.bold)),
      ),
      body: _buildList(),
    );
  }

  Widget _buildList() {
    final theme = Theme.of(context);
    final loc = S.of(context);

    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }
    if (_items.isEmpty) {
      return Center(
        child: Text(loc.oaNotificationsEmpty,
            style: TextStyle(color: theme.colorScheme.onSurface.withAlpha(120))),
      );
    }
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView.separated(
        physics: const AlwaysScrollableScrollPhysics(),
        itemCount: _items.length,
        separatorBuilder: (_, __) => const Divider(height: 1),
        itemBuilder: (context, i) {
          final row = _items[i] as Map<String, dynamic>? ?? const {};
          return ListTile(
            title: Text(row['title'] as String? ?? '',
                maxLines: 1, overflow: TextOverflow.ellipsis),
            subtitle: Text(
              '${row['sender_name'] ?? ''} · ${row['created_at'] ?? ''}',
              style: const TextStyle(fontSize: 12),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          );
        },
      ),
    );
  }
}
