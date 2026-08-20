import 'package:flutter/material.dart';

import 'package:flutter_app/src/features/oa/services/expense_service.dart';
import 'package:flutter_app/src/features/oa/services/file_upload_service.dart';
import 'package:flutter_app/src/core/transport/http/status.dart';
import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;
import 'package:image_picker/image_picker.dart';

/// 报销申请页（含多行费用明细）。
///
/// 提交自动创建 EXPENSE 工作流实例。明细的发票凭证文件 ID 可选填（先经
/// 文件上传接口上传发票图片取得 file_id）。下方为我的报销申请列表。
class OaExpensePage extends StatefulWidget {
  const OaExpensePage({super.key});

  @override
  State<OaExpensePage> createState() => _OaExpensePageState();
}

class _OaExpensePageState extends State<OaExpensePage> {
  final _service = ExpenseService();
  final _uploadService = FileUploadService();
  final _formKey = GlobalKey<FormState>();
  final _titleCtrl = TextEditingController();

  final List<_ItemDraft> _items = [];
  List<oaApi.OaServiceV1ExpenseApplication> _applications = const [];
  bool _submitting = false;

  @override
  void initState() {
    super.initState();
    _items.add(_ItemDraft());
    _loadApplications();
  }

  @override
  void dispose() {
    _titleCtrl.dispose();
    for (final item in _items) {
      item.dispose();
    }
    super.dispose();
  }

  Future<void> _loadApplications() async {
    final result = await _service.myApplications();
    if (!mounted) return;
    setState(() {
      _applications = (result is Status)
          ? const []
          : (result as oaApi.OaServiceV1ListExpenseApplicationsResponse?)
                  ?.items ??
              const [];
    });
  }

  double get _total =>
      _items.fold(0.0, (sum, item) => sum + (double.tryParse(item.amount.text) ?? 0));

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() => _submitting = true);
    final items = _items
        .map((item) => oaApi.OaServiceV1ExpenseItem(
              category: item.category.text,
              amount: double.tryParse(item.amount.text) ?? 0,
              description: item.description.text,
              invoiceFileId: item.invoiceFileId,
            ))
        .toList();
    final result = await _service.submit(title: _titleCtrl.text, items: items);
    if (!mounted) return;
    setState(() => _submitting = false);
    if (result is Status) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(result.message ?? '提交失败')));
    } else {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('已提交，等待审批')));
      _titleCtrl.clear();
      for (final item in _items) {
        item.clear();
      }
      _loadApplications();
    }
  }

  /// 拍照/相册选图 → multipart 上传 → 回填明细的 invoiceFileId。
  Future<void> _pickInvoice(int itemIndex, ImageSource source) async {
    final picker = ImagePicker();
    XFile? image;
    try {
      image = await picker.pickImage(
        source: source,
        maxWidth: 1920,
        imageQuality: 85,
      );
    } catch (_) {
      image = null;
    }
    if (image == null || !mounted || itemIndex >= _items.length) return;

    setState(() => _items[itemIndex].invoiceUploading = true);
    final result = await _uploadService.uploadInvoice(image);
    if (!mounted) return;
    setState(() => _items[itemIndex].invoiceUploading = false);
    if (result is UploadedInvoice) {
      setState(() => _items[itemIndex].invoiceFileId = result.fileId);
      if (mounted) {
        ScaffoldMessenger.of(context)
          ..hideCurrentSnackBar()
          ..showSnackBar(const SnackBar(content: Text('发票已上传')));
      }
    } else if (result is Status) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(result.message ?? '上传失败')));
    }
  }

  static String _statusLabel(
      oaApi.OaServiceV1ExpenseApplication$ExpenseStatus? s) {
    switch (s) {
      case oaApi.OaServiceV1ExpenseApplication$ExpenseStatus.approved:
        return '已通过';
      case oaApi.OaServiceV1ExpenseApplication$ExpenseStatus.rejected:
        return '已驳回';
      case oaApi.OaServiceV1ExpenseApplication$ExpenseStatus.withdrawn:
        return '已撤回';
      default:
        return '审批中';
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(title: const Text('费用报销')),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            TextFormField(
              controller: _titleCtrl,
              decoration: const InputDecoration(
                  labelText: '报销事由', border: OutlineInputBorder()),
              validator: (v) =>
                  (v == null || v.isEmpty) ? 'required' : null,
            ),
            const SizedBox(height: 16),
            Text('费用明细（合计 ${_total.toStringAsFixed(2)}）',
                style: theme.textTheme.titleSmall),
            ..._items.asMap().entries.map(
                  (entry) => Padding(
                    padding: const EdgeInsets.only(top: 8),
                    child: _ItemCard(
                      draft: entry.value,
                      index: entry.key,
                      canRemove: _items.length > 1,
                      onRemove: () => setState(() {
                        _items[entry.key].dispose();
                        _items.removeAt(entry.key);
                      }),
                      onChanged: () => setState(() {}),
                      onPickInvoice: _pickInvoice,
                    ),
                  ),
                ),
            const SizedBox(height: 8),
            OutlinedButton.icon(
              onPressed: () => setState(() => _items.add(_ItemDraft())),
              icon: const Icon(Icons.add),
              label: const Text('添加明细'),
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
            Text('我的报销', style: theme.textTheme.titleSmall),
            if (_applications.isEmpty)
              const Padding(padding: EdgeInsets.all(8), child: Text('暂无记录'))
            else
              ..._applications.map((a) => ListTile(
                    dense: true,
                    title: Text(a.title ?? ''),
                    subtitle: Text(
                        '${a.items?.length ?? 0} 项明细    ${a.createdAt ?? ''}',
                        style: const TextStyle(fontSize: 12)),
                    trailing: Text(
                        '${a.totalAmount ?? 0}\n${_statusLabel(a.expenseStatus)}',
                        textAlign: TextAlign.end),
                  )),
          ],
        ),
      ),
    );
  }
}

/// 明细草稿（本地 TextEditingController 集）。
///
/// 发票凭证经拍照/相册上传后自动回填 [invoiceFileId]，
/// [invoiceUploading] 驱动按钮加载态。
class _ItemDraft {
  final category = TextEditingController();
  final amount = TextEditingController();
  final description = TextEditingController();
  int? invoiceFileId;
  bool invoiceUploading = false;

  void dispose() {
    category.dispose();
    amount.dispose();
    description.dispose();
  }

  void clear() {
    category.clear();
    amount.clear();
    description.clear();
    invoiceFileId = null;
    invoiceUploading = false;
  }
}

class _ItemCard extends StatelessWidget {
  final _ItemDraft draft;
  final int index;
  final bool canRemove;
  final VoidCallback onRemove;
  final VoidCallback onChanged;
  final void Function(int itemIndex, ImageSource source) onPickInvoice;

  const _ItemCard({
    required this.draft,
    required this.index,
    required this.canRemove,
    required this.onRemove,
    required this.onChanged,
    required this.onPickInvoice,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          children: [
            Row(
              children: [
                Expanded(
                  child: Text('明细 ${index + 1}',
                      style: Theme.of(context).textTheme.titleSmall),
                ),
                if (canRemove)
                  IconButton(
                      icon: const Icon(Icons.delete_outline, size: 20),
                      onPressed: onRemove),
              ],
            ),
            TextFormField(
              controller: draft.category,
              decoration: const InputDecoration(
                  labelText: '类别（交通/餐饮/办公…）',
                  border: OutlineInputBorder(),
                  isDense: true),
              onChanged: (_) => onChanged(),
            ),
            const SizedBox(height: 8),
            TextFormField(
              controller: draft.amount,
              keyboardType:
                  const TextInputType.numberWithOptions(decimal: true),
              decoration: const InputDecoration(
                  labelText: '金额', border: OutlineInputBorder(), isDense: true),
              validator: (v) {
                final amount = double.tryParse(v ?? '');
                return (amount == null || amount <= 0) ? 'required' : null;
              },
              onChanged: (_) => onChanged(),
            ),
            const SizedBox(height: 8),
            TextFormField(
              controller: draft.description,
              decoration: const InputDecoration(
                  labelText: '说明', border: OutlineInputBorder(), isDense: true),
            ),
            const SizedBox(height: 8),
            _InvoicePickerRow(draft: draft, index: index, onPick: onPickInvoice),
          ],
        ),
      ),
    );
  }
}


/// 发票凭证选择行：拍照/相册按钮 + 上传中/已上传状态。
class _InvoicePickerRow extends StatelessWidget {
  final _ItemDraft draft;
  final int index;
  final void Function(int itemIndex, ImageSource source) onPick;

  const _InvoicePickerRow({
    required this.draft,
    required this.index,
    required this.onPick,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    if (draft.invoiceUploading) {
      return const Row(
        children: [
          SizedBox(
              width: 16,
              height: 16,
              child: CircularProgressIndicator(strokeWidth: 2)),
          SizedBox(width: 8),
          Text('发票上传中…', style: TextStyle(fontSize: 13)),
        ],
      );
    }
    if (draft.invoiceFileId != null) {
      return Row(
        children: [
          Icon(Icons.check_circle_outline,
              size: 16, color: theme.colorScheme.primary),
          const SizedBox(width: 6),
          Text('发票已上传（文件 #${draft.invoiceFileId}）',
              style: TextStyle(
                  fontSize: 13, color: theme.colorScheme.onSurface)),
        ],
      );
    }
    return Row(
      children: [
        OutlinedButton.icon(
          onPressed: () => onPick(index, ImageSource.camera),
          icon: const Icon(Icons.photo_camera, size: 16),
          label: const Text('拍照', style: TextStyle(fontSize: 13)),
        ),
        const SizedBox(width: 8),
        OutlinedButton.icon(
          onPressed: () => onPick(index, ImageSource.gallery),
          icon: const Icon(Icons.photo_library, size: 16),
          label: const Text('相册', style: TextStyle(fontSize: 13)),
        ),
        const SizedBox(width: 8),
        Text('上传发票凭证（可选）',
            style: TextStyle(
                fontSize: 12,
                color: theme.colorScheme.onSurface.withAlpha(140))),
      ],
    );
  }
}
