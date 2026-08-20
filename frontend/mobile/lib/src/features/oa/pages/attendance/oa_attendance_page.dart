import 'package:flutter/material.dart';

import 'package:flutter_app/generated/l10n.dart';
import 'package:flutter_app/src/features/oa/services/attendance_service.dart';
import 'package:flutter_app/src/core/transport/http/status.dart';
import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

/// 考勤打卡页。
///
/// 调 [AttendanceService.checkIn]：采集 GPS/WiFi BSSID 后提交，服务端当日
/// 首次打卡=签到、第二次=签退并结算（正常/迟到/早退/请假）。下方展示近 30 天
/// 打卡记录与结算结果。
class OaAttendancePage extends StatefulWidget {
  const OaAttendancePage({super.key});

  @override
  State<OaAttendancePage> createState() => _OaAttendancePageState();
}

class _OaAttendancePageState extends State<OaAttendancePage> {
  final AttendanceService _service = AttendanceService();
  bool _busy = false;
  List<oaApi.OaServiceV1AttendanceRecord> _records = const [];

  @override
  void initState() {
    super.initState();
    _loadRecords();
  }

  Future<void> _loadRecords() async {
    final result = await _service.myRecords();
    if (!mounted) return;
    setState(() {
      _records = (result is Status)
          ? const []
          : (result as oaApi.OaServiceV1ListAttendanceRecordsResponse?)?.items ??
              const [];
    });
  }

  Future<void> _doCheckIn() async {
    setState(() => _busy = true);
    final result = await _service.checkIn();
    if (!mounted) return;
    setState(() => _busy = false);

    final theme = Theme.of(context);
    final (msg, color) = _messageFor(result, theme);
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(msg),
        backgroundColor: color ?? theme.colorScheme.surfaceContainerHighest,
      ),
    );
    _loadRecords();
  }

  /// 打卡结果 → 展示文案。成功时按返回记录区分签到/签退与当日结算结果。
  (String, Color?) _messageFor(dynamic result, ThemeData theme) {
    final scheme = theme.colorScheme;
    final loc = S.of(context);
    if (result is AttendancePreflight) {
      switch (result) {
        case AttendancePreflight.permissionDenied:
          return (loc.oaAttendancePermissionDenied, scheme.error);
        case AttendancePreflight.locationUnavailable:
          return (loc.oaAttendanceLocationUnavailable, scheme.error);
      }
    }
    if (result is Status) {
      return (result.message ?? loc.errorOccurred, scheme.error);
    }
    final record = result as oaApi.OaServiceV1AttendanceRecord?;
    if (record == null) {
      return (loc.errorOccurred, scheme.error);
    }
    if (record.checkOutAt != null) {
      return ('下班签退成功，当日结算：${dayResultLabel(record.dayResult)}', scheme.primary);
    }
    return ('上班签到成功 ${record.checkInAt ?? ''}', scheme.primary);
  }

  static String dayResultLabel(oaApi.OaServiceV1AttendanceRecord$DayResult? r) {
    switch (r) {
      case oaApi.OaServiceV1AttendanceRecord$DayResult.normal:
        return '正常';
      case oaApi.OaServiceV1AttendanceRecord$DayResult.late_:
        return '迟到';
      case oaApi.OaServiceV1AttendanceRecord$DayResult.earlyLeave:
        return '早退';
      case oaApi.OaServiceV1AttendanceRecord$DayResult.absent:
        return '旷工';
      case oaApi.OaServiceV1AttendanceRecord$DayResult.onLeave:
        return '请假';
      default:
        return '待结算';
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
      body: Column(
        children: [
          Padding(
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
                  label: Text(
                      _busy ? loc.oaAttendanceSubmitting : loc.oaAttendanceCheckIn),
                ),
              ],
            ),
          ),
          const Divider(height: 1),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
            child: Align(
              alignment: Alignment.centerLeft,
              child: Text('近 30 天记录',
                  style: theme.textTheme.titleSmall),
            ),
          ),
          Expanded(
            child: RefreshIndicator(
              onRefresh: _loadRecords,
              child: _records.isEmpty
                  ? ListView(children: const [
                      Padding(
                        padding: EdgeInsets.all(32),
                        child: Center(child: Text('暂无打卡记录')),
                      ),
                    ])
                  : ListView.separated(
                      physics: const AlwaysScrollableScrollPhysics(),
                      itemCount: _records.length,
                      separatorBuilder: (_, __) => const Divider(height: 1),
                      itemBuilder: (context, i) {
                        final r = _records[i];
                        final date = (r.workDate ?? '').split('T').first;
                        return ListTile(
                          dense: true,
                          title: Text(date),
                          subtitle: Text(
                            '签到 ${(r.checkInAt ?? '-').split('T').last.split('.').first}    '
                            '签退 ${(r.checkOutAt ?? '-').split('T').last.split('.').first}',
                            style: const TextStyle(fontSize: 12),
                          ),
                          trailing: Text(dayResultLabel(r.dayResult)),
                        );
                      },
                    ),
            ),
          ),
        ],
      ),
    );
  }
}
