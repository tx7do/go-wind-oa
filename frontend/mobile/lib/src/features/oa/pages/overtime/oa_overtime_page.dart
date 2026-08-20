import 'package:flutter/material.dart';

import 'package:flutter_app/src/features/oa/services/overtime_service.dart';
import 'package:flutter_app/src/core/transport/http/status.dart';
import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

/// 加班申请页。
///
/// 上半部：提交表单（事由 / 起止时间 / 补偿方式），提交自动创建 OVERTIME 工作流
/// 实例，终态仅同步单据状态，无额度副作用。下半部：我的加班申请列表。
class OaOvertimePage extends StatefulWidget {
  const OaOvertimePage({super.key});

  @override
  State<OaOvertimePage> createState() => _OaOvertimePageState();
}

class _OaOvertimePageState extends State<OaOvertimePage> {
  final _service = OvertimeService();
  final _formKey = GlobalKey<FormState>();
  final _reasonCtrl = TextEditingController();

  List<oaApi.OaServiceV1OvertimeApplication> _applications = const [];
  DateTime? _startTime;
  DateTime? _endTime;
  oaApi.OaServiceV1OvertimeApplication$CompensationType _compensation =
      oaApi.OaServiceV1OvertimeApplication$CompensationType.compLeave;
  bool _submitting = false;

  @override
  void initState() {
    super.initState();
    _loadApplications();
  }

  @override
  void dispose() {
    _reasonCtrl.dispose();
    super.dispose();
  }

  Future<void> _loadApplications() async {
    final result = await _service.myApplications();
    if (!mounted) return;
    setState(() {
      _applications = (result is Status)
          ? const []
          : (result as oaApi.OaServiceV1ListOvertimeApplicationsResponse?)
                  ?.items ??
              const [];
    });
  }

  Future<void> _pickTime({required bool isStart}) async {
    final now = DateTime.now();
    final picked = await showDatePicker(
      context: context,
      initialDate: now,
      firstDate: DateTime(now.year - 1),
      lastDate: DateTime(now.year + 2),
    );
    if (picked == null || !mounted) return;
    setState(() {
      if (isStart) {
        _startTime = picked;
      } else {
        _endTime = picked;
      }
    });
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    if (_startTime == null || _endTime == null) {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('请选择起止时间')));
      return;
    }
    setState(() => _submitting = true);
    final result = await _service.submit(
      reason: _reasonCtrl.text,
      startTime: _startTime!,
      endTime: _endTime!,
      compensationType: _compensation,
    );
    if (!mounted) return;
    setState(() => _submitting = false);
    if (result is Status) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(result.message ?? '提交失败')));
    } else {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('已提交，等待审批')));
      _reasonCtrl.clear();
      _loadApplications();
    }
  }

  static String _statusLabel(
      oaApi.OaServiceV1OvertimeApplication$OvertimeStatus? s) {
    switch (s) {
      case oaApi.OaServiceV1OvertimeApplication$OvertimeStatus.approved:
        return '已通过';
      case oaApi.OaServiceV1OvertimeApplication$OvertimeStatus.rejected:
        return '已驳回';
      case oaApi.OaServiceV1OvertimeApplication$OvertimeStatus.withdrawn:
        return '已撤回';
      default:
        return '审批中';
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(title: const Text('加班申请')),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            TextFormField(
              controller: _reasonCtrl,
              maxLines: 3,
              decoration: const InputDecoration(
                  labelText: '加班事由', border: OutlineInputBorder()),
              validator: (v) =>
                  (v == null || v.isEmpty) ? 'required' : null,
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: () => _pickTime(isStart: true),
                    child: Text(_startTime == null
                        ? '开始时间'
                        : _startTime!.toString().split(' ').first),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: OutlinedButton(
                    onPressed: () => _pickTime(isStart: false),
                    child: Text(_endTime == null
                        ? '结束时间'
                        : _endTime!.toString().split(' ').first),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            DropdownButtonFormField<oaApi.OaServiceV1OvertimeApplication$CompensationType>(
              initialValue: _compensation,
              decoration: const InputDecoration(
                  labelText: '补偿方式', border: OutlineInputBorder()),
              items: const [
                DropdownMenuItem(
                    value: oaApi.OaServiceV1OvertimeApplication$CompensationType.compLeave,
                    child: Text('调休')),
                DropdownMenuItem(
                    value: oaApi.OaServiceV1OvertimeApplication$CompensationType.overtimePay,
                    child: Text('加班费')),
              ],
              onChanged: (v) => setState(() => _compensation = v ?? _compensation),
              validator: (v) => v == null ? 'required' : null,
            ),
            const SizedBox(height: 16),
            FilledButton(
              onPressed: _submitting ? null : _submit,
              child: _submitting
                  ? const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(strokeWidth: 2))
                  : const Text('提交申请'),
            ),
            const SizedBox(height: 24),
            Text('我的加班申请', style: theme.textTheme.titleSmall),
            if (_applications.isEmpty)
              const Padding(padding: EdgeInsets.all(8), child: Text('暂无记录'))
            else
              ..._applications.map((a) => ListTile(
                    dense: true,
                    title: Text(a.reason ?? ''),
                    subtitle: Text(
                        '${(a.startTime ?? '').split('T').first} ~ ${(a.endTime ?? '').split('T').first}',
                        maxLines: 1, overflow: TextOverflow.ellipsis),
                    trailing: Text(_statusLabel(a.overtimeStatus)),
                  )),
          ],
        ),
      ),
    );
  }
}
