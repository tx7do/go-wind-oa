import 'package:flutter/material.dart';

import 'package:flutter_app/src/features/oa/services/seal_application_service.dart';
import 'package:flutter_app/src/core/transport/http/status.dart';
import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

/// 用印申请页。
///
/// 上半部：提交表单（事由 / 印章类型 / 文件份数 / 收件方），提交自动创建
/// SEAL_APPLICATION 工作流实例，终态仅同步单据状态，无额度副作用。下半部：我的
/// 用印申请列表。
class OaSealApplicationPage extends StatefulWidget {
  const OaSealApplicationPage({super.key});

  @override
  State<OaSealApplicationPage> createState() => _OaSealApplicationPageState();
}

class _OaSealApplicationPageState extends State<OaSealApplicationPage> {
  final _service = SealApplicationService();
  final _formKey = GlobalKey<FormState>();
  final _purposeCtrl = TextEditingController();
  final _recipientCtrl = TextEditingController();

  List<oaApi.OaServiceV1SealApplication> _applications = const [];
  oaApi.OaServiceV1SealApplication$SealType _sealType =
      oaApi.OaServiceV1SealApplication$SealType.officialSeal;
  int _fileCount = 1;
  bool _submitting = false;

  @override
  void initState() {
    super.initState();
    _loadApplications();
  }

  @override
  void dispose() {
    _purposeCtrl.dispose();
    _recipientCtrl.dispose();
    super.dispose();
  }

  Future<void> _loadApplications() async {
    final result = await _service.myApplications();
    if (!mounted) return;
    setState(() {
      _applications = (result is Status)
          ? const []
          : (result as oaApi.OaServiceV1ListSealApplicationsResponse?)
                  ?.items ??
              const [];
    });
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() => _submitting = true);
    final result = await _service.submit(
      purpose: _purposeCtrl.text,
      sealType: _sealType,
      fileCount: _fileCount,
      recipient: _recipientCtrl.text,
    );
    if (!mounted) return;
    setState(() => _submitting = false);
    if (result is Status) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(result.message ?? '提交失败')));
    } else {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('已提交，等待审批')));
      _purposeCtrl.clear();
      _recipientCtrl.clear();
      _loadApplications();
    }
  }

  static String _statusLabel(
      oaApi.OaServiceV1SealApplication$SealStatus? s) {
    switch (s) {
      case oaApi.OaServiceV1SealApplication$SealStatus.approved:
        return '已通过';
      case oaApi.OaServiceV1SealApplication$SealStatus.rejected:
        return '已驳回';
      case oaApi.OaServiceV1SealApplication$SealStatus.withdrawn:
        return '已撤回';
      default:
        return '审批中';
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(title: const Text('用印申请')),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            TextFormField(
              controller: _purposeCtrl,
              maxLines: 3,
              decoration: const InputDecoration(
                  labelText: '用印事由', border: OutlineInputBorder()),
              validator: (v) =>
                  (v == null || v.isEmpty) ? 'required' : null,
            ),
            const SizedBox(height: 12),
            DropdownButtonFormField<oaApi.OaServiceV1SealApplication$SealType>(
              initialValue: _sealType,
              decoration: const InputDecoration(
                  labelText: '印章类型', border: OutlineInputBorder()),
              items: const [
                DropdownMenuItem(
                    value: oaApi.OaServiceV1SealApplication$SealType.officialSeal,
                    child: Text('公章')),
                DropdownMenuItem(
                    value: oaApi.OaServiceV1SealApplication$SealType.contractSeal,
                    child: Text('合同章')),
                DropdownMenuItem(
                    value: oaApi.OaServiceV1SealApplication$SealType.financeSeal,
                    child: Text('财务章')),
                DropdownMenuItem(
                    value: oaApi.OaServiceV1SealApplication$SealType.legalSeal,
                    child: Text('法人章')),
              ],
              onChanged: (v) => setState(() => _sealType = v ?? _sealType),
              validator: (v) => v == null ? 'required' : null,
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _recipientCtrl,
              decoration: const InputDecoration(
                  labelText: '收件方', border: OutlineInputBorder()),
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
            Text('我的用印申请', style: theme.textTheme.titleSmall),
            if (_applications.isEmpty)
              const Padding(padding: EdgeInsets.all(8), child: Text('暂无记录'))
            else
              ..._applications.map((a) => ListTile(
                    dense: true,
                    title: Text(a.purpose ?? ''),
                    subtitle: Text(a.recipient ?? '',
                        maxLines: 1, overflow: TextOverflow.ellipsis),
                    trailing: Text(_statusLabel(a.sealStatus)),
                  )),
          ],
        ),
      ),
    );
  }
}
