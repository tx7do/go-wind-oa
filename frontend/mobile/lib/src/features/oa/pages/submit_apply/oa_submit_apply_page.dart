import 'dart:convert';

import 'package:flutter/material.dart';

import 'package:flutter_app/generated/l10n.dart';
import 'package:flutter_app/src/features/oa/services/workflow_service.dart';
import 'package:flutter_app/src/core/transport/http/status.dart';

/// 提交工作流申请页。
///
/// 输入流程定义 code/version 后点「加载表单」：若定义配置了 form_schema
/// （字段描述数组），按描述动态渲染表单（文本/多行/数字/日期/下拉），
/// 提交时组装成 JSON；若无表单定义，回退为自由 JSON 文本输入。
class OaSubmitApplyPage extends StatefulWidget {
  const OaSubmitApplyPage({super.key});

  @override
  State<OaSubmitApplyPage> createState() => _OaSubmitApplyPageState();
}

/// 表单字段描述（form_schema 数组元素）。
class FormFieldDesc {
  final String key;
  final String label;
  final String type; // text | textarea | number | date | select
  final bool required;
  final List<String> options;

  const FormFieldDesc({
    required this.key,
    required this.label,
    this.type = 'text',
    this.required = false,
    this.options = const [],
  });

  static FormFieldDesc? fromJson(Map<String, dynamic> json) {
    final key = json['key'];
    if (key == null || key.toString().isEmpty) return null;
    return FormFieldDesc(
      key: key.toString(),
      label: (json['label'] ?? key).toString(),
      type: (json['type'] ?? 'text').toString(),
      required: json['required'] == true,
      options: (json['options'] as List<dynamic>? ?? const [])
          .map((e) => e.toString())
          .toList(),
    );
  }
}

class _OaSubmitApplyPageState extends State<OaSubmitApplyPage> {
  final _service = WorkflowService();
  final _codeCtrl = TextEditingController();
  final _versionCtrl = TextEditingController(text: '1');
  final _formDataCtrl = TextEditingController();

  final Map<String, TextEditingController> _textCtrls = {};
  final Map<String, DateTime?> _dateValues = {};
  final Map<String, String?> _selectValues = {};

  List<FormFieldDesc> _fields = const [];
  bool _formLoaded = false;
  bool _loadingForm = false;
  bool _submitting = false;

  @override
  void dispose() {
    _codeCtrl.dispose();
    _versionCtrl.dispose();
    _formDataCtrl.dispose();
    for (final c in _textCtrls.values) {
      c.dispose();
    }
    super.dispose();
  }

  Future<void> _loadForm() async {
    final code = _codeCtrl.text.trim();
    final version = int.tryParse(_versionCtrl.text) ?? 0;
    if (code.isEmpty || version <= 0) {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('请填写流程代码与版本')));
      return;
    }
    setState(() => _loadingForm = true);
    final result = await _service.fetchApplyForm(code: code, version: version);
    if (!mounted) return;
    setState(() => _loadingForm = false);

    if (result is Status) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(result.message ?? '加载失败')));
      return;
    }
    final schema = (result as dynamic).formSchema;
    _disposeFieldCtrls();
    if (schema == null || schema.toString().trim().isEmpty) {
      // 无表单定义：回退自由 JSON 输入。
      setState(() {
        _fields = const [];
        _formLoaded = true;
      });
      ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('该流程无表单定义，请直接输入 JSON')));
      return;
    }
    try {
      final list = jsonDecode(schema.toString()) as List<dynamic>;
      final descs = <FormFieldDesc>[];
      for (final e in list) {
        if (e is Map<String, dynamic>) {
          final d = FormFieldDesc.fromJson(e);
          if (d != null) descs.add(d);
        }
      }
      setState(() {
        _fields = descs;
        _formLoaded = true;
      });
    } catch (_) {
      setState(() {
        _fields = const [];
        _formLoaded = true;
      });
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('表单定义解析失败，回退 JSON 输入')));
    }
  }

  void _disposeFieldCtrls() {
    for (final c in _textCtrls.values) {
      c.dispose();
    }
    _textCtrls.clear();
    _dateValues.clear();
    _selectValues.clear();
  }

  String? _validate() {
    for (final f in _fields) {
      if (!f.required) continue;
      switch (f.type) {
        case 'date':
          if (_dateValues[f.key] == null) return '请填写「${f.label}」';
          break;
        case 'select':
          if (_selectValues[f.key] == null) return '请选择「${f.label}」';
          break;
        default:
          if (!(_textCtrls[f.key]?.text ?? '').trim().isNotEmpty) {
            return '请填写「${f.label}」';
          }
      }
    }
    return null;
  }

  Map<String, dynamic> _buildFormData() {
    final data = <String, dynamic>{};
    for (final f in _fields) {
      switch (f.type) {
        case 'number':
          final raw = (_textCtrls[f.key]?.text ?? '').trim();
          if (raw.isNotEmpty) data[f.key] = num.tryParse(raw) ?? raw;
          break;
        case 'date':
          final d = _dateValues[f.key];
          if (d != null) {
            data[f.key] =
                '${d.year.toString().padLeft(4, '0')}-${d.month.toString().padLeft(2, '0')}-${d.day.toString().padLeft(2, '0')}';
          }
          break;
        case 'select':
          final v = _selectValues[f.key];
          if (v != null) data[f.key] = v;
          break;
        default:
          final raw = (_textCtrls[f.key]?.text ?? '').trim();
          if (raw.isNotEmpty) data[f.key] = raw;
      }
    }
    return data;
  }

  Future<void> _submit() async {
    String formData;
    if (_fields.isNotEmpty) {
      final err = _validate();
      if (err != null) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(err)));
        return;
      }
      formData = jsonEncode(_buildFormData());
    } else {
      formData = _formDataCtrl.text;
    }

    setState(() => _submitting = true);
    final result = await _service.submitApply(
      code: _codeCtrl.text.trim(),
      version: int.tryParse(_versionCtrl.text) ?? 0,
      formData: formData,
    );
    if (!mounted) return;
    setState(() => _submitting = false);
    if (result is Status) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(result.message ?? 'submit failed')));
    } else {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('OK')));
      Navigator.of(context).maybePop();
    }
  }

  Future<void> _pickDate(FormFieldDesc f) async {
    final now = DateTime.now();
    final picked = await showDatePicker(
      context: context,
      initialDate: _dateValues[f.key] ?? now,
      firstDate: DateTime(now.year - 5),
      lastDate: DateTime(now.year + 5),
    );
    if (picked != null && mounted) {
      setState(() => _dateValues[f.key] = picked);
    }
  }

  @override
  Widget build(BuildContext context) {
    final loc = S.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(loc.oaSubmitApplyTitle)),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Row(
            children: [
              Expanded(
                flex: 2,
                child: TextFormField(
                  controller: _codeCtrl,
                  decoration: InputDecoration(
                      labelText: loc.oaSubmitApplyDefinitionCode,
                      border: const OutlineInputBorder()),
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: TextFormField(
                  controller: _versionCtrl,
                  keyboardType: TextInputType.number,
                  decoration: InputDecoration(
                      labelText: loc.oaSubmitApplyDefinitionVersion,
                      border: const OutlineInputBorder()),
                ),
              ),
              const SizedBox(width: 8),
              OutlinedButton(
                onPressed: _loadingForm ? null : _loadForm,
                child: _loadingForm
                    ? const SizedBox(
                        width: 16,
                        height: 16,
                        child: CircularProgressIndicator(strokeWidth: 2))
                    : const Text('加载表单'),
              ),
            ],
          ),
          const SizedBox(height: 16),
          if (!_formLoaded)
            Padding(
              padding: const EdgeInsets.all(16),
              child: Center(
                child: Text('填写流程代码与版本后点「加载表单」',
                    style: TextStyle(
                        color: Theme.of(context)
                            .colorScheme
                            .onSurface
                            .withAlpha(140))),
              ),
            )
          else if (_fields.isEmpty)
            TextFormField(
              controller: _formDataCtrl,
              maxLines: 8,
              decoration: InputDecoration(
                  labelText: loc.oaSubmitApplyFormData,
                  border: const OutlineInputBorder()),
            )
          else ...[
            for (final f in _fields) _buildField(f),
          ],
          const SizedBox(height: 24),
          FilledButton(
            onPressed: _submitting || !_formLoaded ? null : _submit,
            child: _submitting
                ? const SizedBox(
                    width: 18,
                    height: 18,
                    child: CircularProgressIndicator(strokeWidth: 2))
                : Text(loc.oaSubmitApplySubmit),
          ),
        ],
      ),
    );
  }

  Widget _buildField(FormFieldDesc f) {
    switch (f.type) {
      case 'textarea':
        return Padding(
          padding: const EdgeInsets.only(bottom: 12),
          child: TextFormField(
            controller: _textCtrls.putIfAbsent(f.key, TextEditingController.new),
            maxLines: 3,
            decoration: InputDecoration(
              labelText: f.required ? '${f.label} *' : f.label,
              border: const OutlineInputBorder(),
            ),
          ),
        );
      case 'number':
        return Padding(
          padding: const EdgeInsets.only(bottom: 12),
          child: TextFormField(
            controller: _textCtrls.putIfAbsent(f.key, TextEditingController.new),
            keyboardType:
                const TextInputType.numberWithOptions(decimal: true),
            decoration: InputDecoration(
              labelText: f.required ? '${f.label} *' : f.label,
              border: const OutlineInputBorder(),
            ),
          ),
        );
      case 'date':
        final d = _dateValues[f.key];
        return Padding(
          padding: const EdgeInsets.only(bottom: 12),
          child: InkWell(
            onTap: () => _pickDate(f),
            borderRadius: BorderRadius.circular(4),
            child: InputDecorator(
              decoration: InputDecoration(
                labelText: f.required ? '${f.label} *' : f.label,
                border: const OutlineInputBorder(),
                suffixIcon: const Icon(Icons.calendar_today, size: 18),
              ),
              child: Text(d == null
                  ? '请选择日期'
                  : '${d.year}-${d.month.toString().padLeft(2, '0')}-${d.day.toString().padLeft(2, '0')}'),
            ),
          ),
        );
      case 'select':
        return Padding(
          padding: const EdgeInsets.only(bottom: 12),
          child: DropdownButtonFormField<String>(
            value: _selectValues[f.key],
            items: f.options
                .map((o) => DropdownMenuItem(value: o, child: Text(o)))
                .toList(),
            onChanged: (v) => setState(() => _selectValues[f.key] = v),
            decoration: InputDecoration(
              labelText: f.required ? '${f.label} *' : f.label,
              border: const OutlineInputBorder(),
            ),
          ),
        );
      default:
        return Padding(
          padding: const EdgeInsets.only(bottom: 12),
          child: TextFormField(
            controller: _textCtrls.putIfAbsent(f.key, TextEditingController.new),
            decoration: InputDecoration(
              labelText: f.required ? '${f.label} *' : f.label,
              border: const OutlineInputBorder(),
            ),
          ),
        );
    }
  }
}
