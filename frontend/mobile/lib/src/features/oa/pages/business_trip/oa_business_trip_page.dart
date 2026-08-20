import 'package:flutter/material.dart';

import 'package:flutter_app/src/features/oa/services/business_trip_service.dart';
import 'package:flutter_app/src/core/transport/http/status.dart';
import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

/// 出差申请页。
///
/// 上半部：提交表单（标题 / 目的地 / 起止日期 / 行程说明），提交自动创建
/// BUSINESS_TRIP 工作流实例，终态仅同步单据状态，无额度副作用。下半部：我的
/// 出差申请列表。
class OaBusinessTripPage extends StatefulWidget {
  const OaBusinessTripPage({super.key});

  @override
  State<OaBusinessTripPage> createState() => _OaBusinessTripPageState();
}

class _OaBusinessTripPageState extends State<OaBusinessTripPage> {
  final _service = BusinessTripService();
  final _formKey = GlobalKey<FormState>();
  final _titleCtrl = TextEditingController();
  final _destinationCtrl = TextEditingController();
  final _itineraryCtrl = TextEditingController();

  List<oaApi.OaServiceV1BusinessTripApplication> _applications = const [];
  DateTime? _startDate;
  DateTime? _endDate;
  bool _submitting = false;

  @override
  void initState() {
    super.initState();
    _loadApplications();
  }

  @override
  void dispose() {
    _titleCtrl.dispose();
    _destinationCtrl.dispose();
    _itineraryCtrl.dispose();
    super.dispose();
  }

  Future<void> _loadApplications() async {
    final result = await _service.myApplications();
    if (!mounted) return;
    setState(() {
      _applications = (result is Status)
          ? const []
          : (result as oaApi.OaServiceV1ListBusinessTripApplicationsResponse?)
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
    if (_startDate == null || _endDate == null) {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('请选择起止日期')));
      return;
    }
    setState(() => _submitting = true);
    final result = await _service.submit(
      title: _titleCtrl.text,
      destination: _destinationCtrl.text,
      startDate: _startDate!,
      endDate: _endDate!,
      itinerary: _itineraryCtrl.text,
    );
    if (!mounted) return;
    setState(() => _submitting = false);
    if (result is Status) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(result.message ?? '提交失败')));
    } else {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('已提交，等待审批')));
      _titleCtrl.clear();
      _destinationCtrl.clear();
      _itineraryCtrl.clear();
      _loadApplications();
    }
  }

  static String _statusLabel(
      oaApi.OaServiceV1BusinessTripApplication$BusinessTripStatus? s) {
    switch (s) {
      case oaApi.OaServiceV1BusinessTripApplication$BusinessTripStatus.approved:
        return '已通过';
      case oaApi.OaServiceV1BusinessTripApplication$BusinessTripStatus.rejected:
        return '已驳回';
      case oaApi.OaServiceV1BusinessTripApplication$BusinessTripStatus.withdrawn:
        return '已撤回';
      default:
        return '审批中';
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(title: const Text('出差申请')),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            TextFormField(
              controller: _titleCtrl,
              decoration: const InputDecoration(
                  labelText: '出差事由标题', border: OutlineInputBorder()),
              validator: (v) =>
                  (v == null || v.isEmpty) ? 'required' : null,
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _destinationCtrl,
              decoration: const InputDecoration(
                  labelText: '出差目的地', border: OutlineInputBorder()),
              validator: (v) =>
                  (v == null || v.isEmpty) ? 'required' : null,
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: () => _pickDate(isStart: true),
                    child: Text(_startDate == null
                        ? '开始日期'
                        : _startDate!.toString().split(' ').first),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: OutlinedButton(
                    onPressed: () => _pickDate(isStart: false),
                    child: Text(_endDate == null
                        ? '结束日期'
                        : _endDate!.toString().split(' ').first),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _itineraryCtrl,
              maxLines: 4,
              decoration: const InputDecoration(
                  labelText: '行程安排说明', border: OutlineInputBorder()),
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
            Text('我的出差申请', style: theme.textTheme.titleSmall),
            if (_applications.isEmpty)
              const Padding(padding: EdgeInsets.all(8), child: Text('暂无记录'))
            else
              ..._applications.map((a) => ListTile(
                    dense: true,
                    title: Text(a.title ?? ''),
                    subtitle: Text(
                        '${a.destination ?? ''} ${(a.startDate ?? '').split('T').first} ~ ${(a.endDate ?? '').split('T').first}',
                        maxLines: 1, overflow: TextOverflow.ellipsis),
                    trailing: Text(_statusLabel(a.tripStatus)),
                  )),
          ],
        ),
      ),
    );
  }
}
