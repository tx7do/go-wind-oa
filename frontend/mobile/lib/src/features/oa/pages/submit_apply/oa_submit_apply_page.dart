import 'package:flutter/material.dart';

import 'package:flutter_app/generated/l10n.dart';
import 'package:flutter_app/src/features/oa/services/workflow_service.dart';
import 'package:flutter_app/src/core/transport/http/status.dart';

/// 提交工作流申请页。
///
/// 收集流程定义编码 / 版本 / 标题 / 表单数据(JSON)，调
/// [WorkflowService.submitApply]。成功后返回我发起的列表（路由切换由调用方负责）。
class OaSubmitApplyPage extends StatefulWidget {
  const OaSubmitApplyPage({super.key});

  @override
  State<OaSubmitApplyPage> createState() => _OaSubmitApplyPageState();
}

class _OaSubmitApplyPageState extends State<OaSubmitApplyPage> {
  final _service = WorkflowService();
  final _formKey = GlobalKey<FormState>();
  final _codeCtrl = TextEditingController();
  final _versionCtrl = TextEditingController();
  final _formDataCtrl = TextEditingController();
  bool _submitting = false;

  @override
  void dispose() {
    _codeCtrl.dispose();
    _versionCtrl.dispose();
    _formDataCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() => _submitting = true);
    final version = int.tryParse(_versionCtrl.text) ?? 0;
    final result = await _service.submitApply(
      code: _codeCtrl.text,
      version: version,
      formData: _formDataCtrl.text,
    );
    if (!mounted) return;
    setState(() => _submitting = false);
    if (result is Status) {
      ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(result.message ?? 'submit failed')));
    } else {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('OK')));
      Navigator.of(context).maybePop();
    }
  }

  @override
  Widget build(BuildContext context) {
    final loc = S.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(loc.oaSubmitApplyTitle),
      ),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            TextFormField(
              controller: _codeCtrl,
              decoration: InputDecoration(
                  labelText: loc.oaSubmitApplyDefinitionCode,
                  border: const OutlineInputBorder()),
              validator: (v) =>
                  (v == null || v.isEmpty) ? 'required' : null,
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _versionCtrl,
              keyboardType: TextInputType.number,
              decoration: InputDecoration(
                  labelText: loc.oaSubmitApplyDefinitionVersion,
                  border: const OutlineInputBorder()),
              validator: (v) =>
                  (v == null || v.isEmpty) ? 'required' : null,
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _formDataCtrl,
              maxLines: 8,
              decoration: InputDecoration(
                  labelText: loc.oaSubmitApplyFormData,
                  border: const OutlineInputBorder()),
            ),
            const SizedBox(height: 24),
            FilledButton(
              onPressed: _submitting ? null : _submit,
              child: _submitting
                  ? const CircularProgressIndicator(strokeWidth: 2)
                  : Text(loc.oaSubmitApplySubmit),
            ),
          ],
        ),
      ),
    );
  }
}
