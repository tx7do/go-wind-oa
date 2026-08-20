import 'package:flutter/material.dart';

import 'package:flutter_app/src/features/oa/services/leave_service.dart';
import 'package:flutter_app/src/core/transport/http/status.dart';
import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

/// 请假申请页。
///
/// 上半部：提交表单（类型下拉 / 起止日期 / 事由），提交自动创建 LEAVE 工作流
/// 实例，审批通过后服务端扣减额度。下半部：我的额度 + 我的请假申请列表。
class OaLeavePage extends StatefulWidget {
  const OaLeavePage({super.key});

  @override
  State<OaLeavePage> createState() => _OaLeavePageState();
}

class _OaLeavePageState extends State<OaLeavePage> {
  final _service = LeaveService();
  final _formKey = GlobalKey<FormState>();
  final _reasonCtrl = TextEditingController();

  List<oaApi.OaServiceV1LeaveType> _types = const [];
  List<oaApi.OaServiceV1LeaveBalance> _balances = const [];
  List<oaApi.OaServiceV1LeaveApplication> _applications = const [];
  oaApi.OaServiceV1LeaveType? _selectedType;
  DateTime? _startDate;
  DateTime? _endDate;
  oaApi.OaServiceV1HalfOfDay _startHalf = oaApi.OaServiceV1HalfOfDay.am;
  oaApi.OaServiceV1HalfOfDay _endHalf = oaApi.OaServiceV1HalfOfDay.pm;
  bool _submitting = false;

  @override
  void initState() {
    super.initState();
    _loadAll();
  }

  @override
  void dispose() {
    _reasonCtrl.dispose();
    super.dispose();
  }

  Future<void> _loadAll() async {
    final typesResult = await _service.leaveTypes();
    final balancesResult = await _service.myBalances();
    final appsResult = await _service.myApplications();
    if (!mounted) return;
    setState(() {
      _types = (typesResult is Status)
          ? const []
          : (typesResult as oaApi.OaServiceV1ListLeaveTypesResponse?)?.items ??
              const [];
      _balances = (balancesResult is Status)
          ? const []
          : (balancesResult as oaApi.OaServiceV1ListLeaveBalancesResponse?)
                  ?.items ??
              const [];
      _applications = (appsResult is Status)
          ? const []
          : (appsResult as oaApi.OaServiceV1ListLeaveApplicationsResponse?)
                  ?.items ??
              const [];
    });
  }

  Future<void> _pickDate({required bool isStart}) async {
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
        _startDate = picked;
      } else {
        _endDate = picked;
      }
    });
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    if (_selectedType == null || _startDate == null || _endDate == null) {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('请选择类型与起止日期')));
      return;
    }
    setState(() => _submitting = true);
    final result = await _service.submit(
      leaveTypeId: _selectedType!.id ?? 0,
      startDate: _startDate!,
      endDate: _endDate!,
      reason: _reasonCtrl.text,
      startHalf: _startHalf,
      endHalf: _endHalf,
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
      _loadAll();
    }
  }

  static String _statusLabel(oaApi.OaServiceV1LeaveApplication$LeaveStatus? s) {
    switch (s) {
      case oaApi.OaServiceV1LeaveApplication$LeaveStatus.approved:
        return '已通过';
      case oaApi.OaServiceV1LeaveApplication$LeaveStatus.rejected:
        return '已驳回';
      case oaApi.OaServiceV1LeaveApplication$LeaveStatus.withdrawn:
        return '已撤回';
      default:
        return '审批中';
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(title: const Text('请假申请')),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            DropdownButtonFormField<oaApi.OaServiceV1LeaveType>(
              initialValue: _selectedType,
              decoration: const InputDecoration(
                  labelText: '请假类型', border: OutlineInputBorder()),
              items: _types
                  .map((t) => DropdownMenuItem(value: t, child: Text(t.name ?? '')))
                  .toList(),
              onChanged: (v) => setState(() => _selectedType = v),
              validator: (v) => v == null ? 'required' : null,
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      OutlinedButton(
                        onPressed: () => _pickDate(isStart: true),
                        child: Text(_startDate == null
                            ? '开始日期'
                            : _startDate!.toString().split(' ').first),
                      ),
                      const SizedBox(height: 6),
                      SegmentedButton<oaApi.OaServiceV1HalfOfDay>(
                        segments: const [
                          ButtonSegment(value: oaApi.OaServiceV1HalfOfDay.am, label: Text('上午起')),
                          ButtonSegment(value: oaApi.OaServiceV1HalfOfDay.pm, label: Text('下午起')),
                        ],
                        selected: {_startHalf},
                        onSelectionChanged: (v) =>
                            setState(() => _startHalf = v.first),
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      OutlinedButton(
                        onPressed: () => _pickDate(isStart: false),
                        child: Text(_endDate == null
                            ? '结束日期'
                            : _endDate!.toString().split(' ').first),
                      ),
                      const SizedBox(height: 6),
                      SegmentedButton<oaApi.OaServiceV1HalfOfDay>(
                        segments: const [
                          ButtonSegment(value: oaApi.OaServiceV1HalfOfDay.am, label: Text('上午止')),
                          ButtonSegment(value: oaApi.OaServiceV1HalfOfDay.pm, label: Text('下午止')),
                        ],
                        selected: {_endHalf},
                        onSelectionChanged: (v) =>
                            setState(() => _endHalf = v.first),
                      ),
                    ],
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _reasonCtrl,
              maxLines: 3,
              decoration: const InputDecoration(
                  labelText: '请假事由', border: OutlineInputBorder()),
              validator: (v) =>
                  (v == null || v.isEmpty) ? 'required' : null,
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
            Text('我的额度', style: theme.textTheme.titleSmall),
            if (_balances.isEmpty)
              const Padding(
                  padding: EdgeInsets.all(8), child: Text('暂无额度，请联系管理员授予'))
            else
              ..._balances.map((b) {
                final typeName = _types
                    .where((t) => t.id == b.leaveTypeId)
                    .map((t) => t.name ?? '')
                    .followedBy(['']).first;
                final left = (b.totalDays ?? 0) - (b.usedDays ?? 0);
                return ListTile(
                  dense: true,
                  title: Text('${typeName} ${b.year ?? ''}'),
                  subtitle: Text(
                      '总额 ${b.totalDays ?? 0} 天 / 已用 ${b.usedDays ?? 0} 天',
                      style: const TextStyle(fontSize: 12)),
                  trailing: Text('余 $left 天'),
                );
              }),
            const SizedBox(height: 16),
            Text('我的申请', style: theme.textTheme.titleSmall),
            if (_applications.isEmpty)
              const Padding(padding: EdgeInsets.all(8), child: Text('暂无记录'))
            else
              ..._applications.map((a) => ListTile(
                    dense: true,
                    title: Text(
                        '${a.leaveTypeName ?? ''} ${(a.startDate ?? '').split('T').first} ~ ${(a.endDate ?? '').split('T').first}（${_fmtDays(a.days)} 天）'),
                    subtitle: Text(a.reason ?? '',
                        maxLines: 1, overflow: TextOverflow.ellipsis),
                    trailing: Text(_statusLabel(a.leaveStatus)),
                  )),
          ],
        ),
      ),
    );
  }

  static String _fmtDays(double? d) {
    if (d == null) return '-';
    return d == d.roundToDouble() ? d.toInt().toString() : d.toString();
  }
}
