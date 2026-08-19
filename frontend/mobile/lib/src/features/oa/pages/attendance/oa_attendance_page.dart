import 'package:flutter/material.dart';

import 'package:flutter_app/generated/l10n.dart';
import 'package:flutter_app/src/features/oa/services/attendance_service.dart';

/// 考勤打卡页。
///
/// 调 [AttendanceService.checkIn] 完成定位采集与服务端围栏 / Wi-Fi 白名单
/// 判定，按返回的 [CheckInResult] 展示结果。
class OaAttendancePage extends StatefulWidget {
  const OaAttendancePage({super.key});

  @override
  State<OaAttendancePage> createState() => _OaAttendancePageState();
}

class _OaAttendancePageState extends State<OaAttendancePage> {
  final AttendanceService _service = AttendanceService();
  bool _busy = false;

  Future<void> _doCheckIn() async {
    setState(() => _busy = true);
    final result = await _service.checkIn();
    if (!mounted) return;
    setState(() => _busy = false);

    final (msg, color) = _messageFor(result);
    final theme = Theme.of(context);
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(msg),
        backgroundColor: color ?? theme.colorScheme.surfaceContainerHighest,
      ),
    );
  }

  /// 将 [CheckInResult] 映射到展示文案与横幅底色。
  /// 成功（inFence/inWifi）用品牌色，拒绝（denied）与前置失败用警示色。
  (String, Color?) _messageFor(CheckInResult r) {
    final loc = S.of(context);
    final scheme = Theme.of(context).colorScheme;
    switch (r) {
      case CheckInResult.inFence:
        return (loc.oaAttendanceResultInFence, scheme.primary);
      case CheckInResult.inWifi:
        return (loc.oaAttendanceResultInWifi, scheme.primary);
      case CheckInResult.denied:
        return (loc.oaAttendanceResultDenied, scheme.error);
      case CheckInResult.permissionDenied:
        return (loc.oaAttendancePermissionDenied, scheme.error);
      case CheckInResult.locationUnavailable:
        return (loc.oaAttendanceLocationUnavailable, scheme.error);
      case CheckInResult.error:
        return (loc.errorOccurred, scheme.error);
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
        child: Padding(
          padding: const EdgeInsets.all(32),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(Icons.place_outlined,
                  size: 64,
                  color: theme.colorScheme.onSurface.withAlpha(120)),
              const SizedBox(height: 16),
              Text(
                loc.oaAttendanceHint,
                textAlign: TextAlign.center,
                style: TextStyle(
                    color: theme.colorScheme.onSurface.withAlpha(160)),
              ),
              const SizedBox(height: 32),
              FilledButton.icon(
                onPressed: _busy ? null : _doCheckIn,
                icon: _busy
                    ? const SizedBox(
                        width: 18,
                        height: 18,
                        child: CircularProgressIndicator(strokeWidth: 2))
                    : const Icon(Icons.login),
                label: Text(_busy
                    ? loc.oaAttendanceSubmitting
                    : loc.oaAttendanceCheckIn),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
