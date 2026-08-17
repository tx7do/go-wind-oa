import 'package:flutter/material.dart';

import 'package:flutter_app/generated/l10n.dart';
import 'package:flutter_app/src/features/oa/services/workflow_service.dart';
import 'package:flutter_app/src/core/transport/http/status.dart';
import 'package:flutter_app/src/app_router/route_names.dart';
import 'package:go_router/go_router.dart';

/// 工作流任务列表页（两 Tab：“待我审批” / “我发起的”）。
///
/// 列表数据经 [WorkflowService.pendingTasks] / [submittedTasks] 直接调用获取，
/// 与 cms tag_list_page 同构（Future + setState）。审批/驳回/转交在详情页完成；
/// 该页每次进入重新拉取，故审批完成后返回本页自动体现“少一笔”。
class OaTaskListPage extends StatefulWidget {
  const OaTaskListPage({super.key});

  @override
  State<OaTaskListPage> createState() => _OaTaskListPageState();
}

class _OaTaskListPageState extends State<OaTaskListPage>
    with SingleTickerProviderStateMixin {
  late final TabController _tabController;
  final _service = WorkflowService();

  // 两 Tab 各自的列表数据
  List<dynamic> _pending = [];
  List<dynamic> _submitted = [];
  bool _loadingPending = true;
  bool _loadingSubmitted = true;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 2, vsync: this);
    _tabController.addListener(() {
      if (!_tabController.indexIsChanging) {
        // Tab 切换时按需加载（首次进入对应 Tab 才发起请求）
        final idx = _tabController.index;
        if (idx == 0 && _loadingPending && _pending.isEmpty) _loadPending();
        if (idx == 1 && _loadingSubmitted && _submitted.isEmpty) _loadSubmitted();
      }
    });
    _loadPending();
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  Future<void> _loadPending() async {
    final result = await _service.pendingTasks();
    if (!mounted) return;
    setState(() {
      _pending = (result is Status) ? [] : (result as dynamic)?.items ?? [];
      _loadingPending = false;
    });
  }

  Future<void> _loadSubmitted() async {
    final result = await _service.submittedTasks();
    if (!mounted) return;
    setState(() {
      _submitted = (result is Status) ? [] : (result as dynamic)?.items ?? [];
      _loadingSubmitted = false;
    });
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
        title: Text(loc.oaTaskListTitlePending,
            style: const TextStyle(fontWeight: FontWeight.bold)),
        bottom: TabBar(
          controller: _tabController,
          tabs: [
            Tab(text: loc.oaTaskListTitlePending),
            Tab(text: loc.oaTaskListTitleSubmitted),
          ],
        ),
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => GoRouter.of(context).goNamed(RouteNames.oaSubmitApply),
        icon: const Icon(Icons.add),
        label: Text(loc.oaTaskListFabApply),
      ),
      body: TabBarView(
        controller: _tabController,
        children: [
          _buildList(_pending, _loadingPending, _loadPending),
          _buildList(_submitted, _loadingSubmitted, _loadSubmitted),
        ],
      ),
    );
  }

  Widget _buildList(
      List<dynamic> items, bool loading, Future<void> Function() onRefresh) {
    final theme = Theme.of(context);
    final loc = S.of(context);

    if (loading) {
      return const Center(child: CircularProgressIndicator());
    }
    if (items.isEmpty) {
      return Center(
        child: Text(loc.oaTaskListEmpty,
            style: TextStyle(color: theme.colorScheme.onSurface.withAlpha(120))),
      );
    }
    return RefreshIndicator(
      onRefresh: onRefresh,
      child: ListView.separated(
        physics: const AlwaysScrollableScrollPhysics(),
        itemCount: items.length,
        separatorBuilder: (_, __) => const Divider(height: 1),
        itemBuilder: (context, i) {
          final row = items[i] as Map<String, dynamic>? ?? const {};
          final int taskId = (row['task_id'] as int?) ?? 0;
          return ListTile(
            title: Text(row['title'] as String? ?? ''),
            subtitle: Text(
              '${loc.oaTaskListStatus}: ${row['status_label'] ?? ''}',
              style: const TextStyle(fontSize: 12),
            ),
            trailing: Text(
              '${row['occurred_at'] ?? ''}',
              style: const TextStyle(fontSize: 11),
            ),
            onTap: () {
              if (taskId > 0) {
                GoRouter.of(context).goNamed(RouteNames.oaTaskDetail, pathParameters: {'id': taskId.toString()});
              }
            },
          );
        },
      ),
    );
  }
}
