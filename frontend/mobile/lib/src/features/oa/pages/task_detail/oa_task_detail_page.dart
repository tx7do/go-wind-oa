import 'package:flutter/material.dart';

import 'package:flutter_app/generated/l10n.dart';
import 'package:flutter_app/src/features/oa/services/workflow_service.dart';
import 'package:flutter_app/src/core/transport/http/status.dart';
import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

/// 任务详情页。
///
/// 调 [WorkflowService.getTaskDetail] 取单任务详情（申请标题、申请表单数据、
/// 该实例的审批日志轨迹），渲染为只读摘要 + 表单数据 + 历史轨迹。鉴权由后端
/// GetTask 服务强制（task 必须指派给 caller 且 PENDING），前端无需二次校验。
///
/// 审批按钮调 [WorkflowService.audit]，传入 APPROVE/REJECT/FORWARD。
/// 转交弹 dialog 收集 forwardToUserId。审批完成后页面返回列表。
class OaTaskDetailPage extends StatefulWidget {
  final int taskId;
  const OaTaskDetailPage({super.key, required this.taskId});

  @override
  State<OaTaskDetailPage> createState() => _OaTaskDetailPageState();
}

class _OaTaskDetailPageState extends State<OaTaskDetailPage> {
  final _service = WorkflowService();
  final _commentCtrl = TextEditingController();
  final _forwardToCtrl = TextEditingController();

  @override
  void dispose() {
    _commentCtrl.dispose();
    _forwardToCtrl.dispose();
    super.dispose();
  }

  Future<void> _doAudit(oaApi.OaServiceV1AuditTaskRequest$AuditAction action,
      {int? forwardTo}) async {
    final result = await _service.audit(
      taskId: widget.taskId,
      action: action,
      comment: _commentCtrl.text.isEmpty ? null : _commentCtrl.text,
      forwardToUserId: forwardTo,
    );
    if (!mounted) return;
    if (result is Status) {
      ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(result.message ?? 'audit failed')));
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('OK')));
      Navigator.of(context).maybePop();
    }
  }

  void _showForwardDialog() {
    final loc = S.of(context);
    showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(loc.oaTaskDetailForwardDialogTitle),
        content: TextField(
          controller: _forwardToCtrl,
          keyboardType: TextInputType.number,
          decoration:
              InputDecoration(labelText: loc.oaTaskDetailForwardTo),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: Text(loc.oaTaskDetailCancel),
          ),
          TextButton(
            onPressed: () {
              final fid = int.tryParse(_forwardToCtrl.text);
              Navigator.of(ctx).pop();
              if (fid != null && fid > 0) {
                _doAudit(oaApi.OaServiceV1AuditTaskRequest$AuditAction.forward, forwardTo: fid);
              }
            },
            child: Text(loc.oaTaskDetailConfirm),
          ),
        ],
      ),
    );
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
        title: Text(loc.oaTaskDetailTitle),
      ),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text('Task ID: ${widget.taskId}',
                style: theme.textTheme.titleMedium),
            const SizedBox(height: 8),
            Expanded(
              child: FutureBuilder(
                future: _service.getTaskDetail(taskId: widget.taskId),
                builder: (context, snapshot) {
                  if (snapshot.connectionState != ConnectionState.done) {
                    return Center(
                        child: CircularProgressIndicator(
                            valueColor: AlwaysStoppedAnimation<Color>(
                                theme.colorScheme.onSurface.withAlpha(120))));
                  }
                  final result = snapshot.data;
                  if (result is Status) {
                    return Center(
                        child: Text(result.message ?? 'load failed',
                            style: TextStyle(
                                color: theme.colorScheme.error)));
                  }
                  final detail = result as oaApi.OaServiceV1GetTaskResponse?;
                  if (detail == null) {
                    return Center(
                        child: Text(loc.oaTaskDetailLoading,
                            style: TextStyle(
                                color: theme.colorScheme.onSurface.withAlpha(120))));
                  }
                  return _DetailContent(detail: detail, theme: theme, loc: loc);
                },
              ),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: _commentCtrl,
              maxLines: 3,
              decoration: InputDecoration(
                  labelText: loc.oaTaskDetailComment,
                  border: const OutlineInputBorder()),
            ),
            const SizedBox(height: 16),
            Row(
              children: [
                Expanded(
                  child: FilledButton(
                    onPressed: () =>
                        _doAudit(oaApi.OaServiceV1AuditTaskRequest$AuditAction.approve),
                    child: Text(loc.oaTaskDetailApprove),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: FilledButton.tonal(
                    onPressed: () =>
                        _doAudit(oaApi.OaServiceV1AuditTaskRequest$AuditAction.reject),
                    child: Text(loc.oaTaskDetailReject),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: OutlinedButton(
                    onPressed: _showForwardDialog,
                    child: Text(loc.oaTaskDetailForward),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

/// 详情内容渲染：申请标题、申请表单数据（只读 JSON 文本）、审批日志轨迹。
///
/// 表单数据与历史轨迹均为只读展示，引擎不解释表单字段语义。历史轨迹仅含
/// 审批类动作（APPROVE/REJECT/FORWARD），提交动作不在详情视图——与后端
/// ListByInstance 过滤口径一致。
class _DetailContent extends StatelessWidget {
  final oaApi.OaServiceV1GetTaskResponse detail;
  final ThemeData theme;
  final S loc;

  const _DetailContent(
      {required this.detail, required this.theme, required this.loc});

  @override
  Widget build(BuildContext context) {
    final history = detail.history ?? <oaApi.OaServiceV1AuditLogEntry>[];
    return ListView(
      children: [
        Text(loc.oaTaskDetailSummaryTitle,
            style: theme.textTheme.titleSmall),
        const SizedBox(height: 8),
        Text('${loc.oaTaskDetailSummaryTitleField}: ${detail.title ?? '-'}',
            style: theme.textTheme.bodyMedium),
        const SizedBox(height: 16),
        Text(loc.oaTaskDetailFormDataTitle,
            style: theme.textTheme.titleSmall),
        const SizedBox(height: 8),
        Container(
          padding: const EdgeInsets.all(8),
          decoration: BoxDecoration(
            color: theme.colorScheme.surfaceContainerHighest,
            borderRadius: BorderRadius.circular(4),
          ),
          child: SelectableText(
            detail.formData ?? '-',
            style: theme.textTheme.bodySmall,
          ),
        ),
        const SizedBox(height: 16),
        Text(loc.oaTaskDetailHistoryTitle,
            style: theme.textTheme.titleSmall),
        const SizedBox(height: 8),
        if (history.isEmpty)
          Text(loc.oaTaskDetailHistoryEmpty,
              style: TextStyle(
                  color: theme.colorScheme.onSurface.withAlpha(120)))
        else
          ...history.map((e) => _HistoryRow(entry: e, theme: theme)),
      ],
    );
  }
}

class _HistoryRow extends StatelessWidget {
  final oaApi.OaServiceV1AuditLogEntry entry;
  final ThemeData theme;

  const _HistoryRow({required this.entry, required this.theme});

  @override
  Widget build(BuildContext context) {
    final ts = entry.occurredAt ?? '-';
    return Card(
      child: ListTile(
        dense: true,
        title: Text(entry.actionLabel ?? '-',
            style: theme.textTheme.bodyMedium),
        subtitle: Text(
            '${ts}\n${entry.comment ?? '-'}',
            style: theme.textTheme.bodySmall),
      ),
    );
  }
}
