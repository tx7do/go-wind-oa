import 'package:flutter/material.dart';

import 'package:flutter_app/generated/l10n.dart';
import 'package:flutter_app/src/features/oa/services/attendance_service.dart';

/// 考勤打卡页（骨架）。
///
/// 调 [checkIn]，当前永远返回 [CheckInResult.notConfigured]，故按钮点击后
/// 展示"考勤服务未配置"。后端考勤服务落地后替换为真实判定与记录逻辑。
class OaAttendancePage extends StatefulWidget {
  const OaAttendancePage({super.key});

  @override
  State<OaAttendancePage> createState() => _OaAttendancePageState();
}

class _OaAttendancePageState extends State<OaAttendancePage> {
  bool _busy = false;

  Future<void> _doCheckIn() async {
    setState(() => _busy = true);
    final result = await checkIn();
    if (!mounted) return;
    setState(() => _busy = false);
    final loc = S.of(context);
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(loc.oaAttendanceNotConfigured)),
    );
    // 后端落地后按 result 分支：ok / notInFence / notConfigured
    if (result == CheckInResult.ok) {
      // 记录成功
    }
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
        title: Text(loc.oaAttendanceTitle),
      ),
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Padding(
              padding: const EdgeInsets.all(32),
              child: Text(
                loc.oaAttendanceNotConfigured,
                textAlign: TextAlign.center,
                style: TextStyle(
                    color: theme.colorScheme.onSurface.withAlpha(120)),
              ),
            ),
            const SizedBox(height: 24),
            FilledButton(
              onPressed: _busy ? null : _doCheckIn,
              child: _busy
                  ? const CircularProgressIndicator(strokeWidth: 2)
                  : Text(loc.oaAttendanceCheckIn),
            ),
          ],
        ),
      ),
    );
  }
}
