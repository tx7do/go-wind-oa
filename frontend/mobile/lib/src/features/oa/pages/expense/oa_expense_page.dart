import 'package:flutter/material.dart';

import 'package:flutter_app/src/features/oa/services/expense_service.dart';
import 'package:flutter_app/src/core/transport/http/status.dart';
import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

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
              invoiceFileId: int.tryParse(item.invoiceFileId.text),
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
class _ItemDraft {
  final category = TextEditingController();
  final amount = TextEditingController();
  final description = TextEditingController();
  final invoiceFileId = TextEditingController();

  void dispose() {
    category.dispose();
    amount.dispose();
    description.dispose();
    invoiceFileId.dispose();
  }

  void clear() {
    category.clear();
    amount.clear();
    description.clear();
    invoiceFileId.clear();
  }
}

class _ItemCard extends StatelessWidget {
  final _ItemDraft draft;
  final int index;
  final bool canRemove;
  final VoidCallback onRemove;
  final VoidCallback onChanged;

  const _ItemCard({
    required this.draft,
    required this.index,
    required this.canRemove,
    required this.onRemove,
    required this.onChanged,
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
            TextFormField(
              controller: draft.invoiceFileId,
              keyboardType: TextInputType.number,
              decoration: const InputDecoration(
                  labelText: '发票文件ID（可选，上传发票图后填入）',
                  border: OutlineInputBorder(),
                  isDense: true),
            ),
          ],
        ),
      ),
    );
  }
}
