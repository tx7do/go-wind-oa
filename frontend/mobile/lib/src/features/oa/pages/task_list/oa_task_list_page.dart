import 'package:flutter/material.dart';

import 'package:flutter_app/generated/l10n.dart';
import 'package:flutter_app/src/features/oa/services/workflow_service.dart';
import 'package:flutter_app/src/core/transport/http/status.dart';
import 'package:flutter_app/src/app_router/route_names.dart';
import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;
import 'package:go_router/go_router.dart';

/// 工作流任务列表页（三 Tab：“待我审批” / “已办” / “我发起的”）。
///
/// 列表数据经 [WorkflowService] 直接调用获取（Future + setState）。
/// “待我审批”项带 taskId，点击进入详情执行审批；“我发起的”进行中的项可撤回
/// （引擎侧校验：仅申请人本人 + 实例进行中）。AppBar 菜单提供请假/报销/通用申请入口。
class OaTaskListPage extends StatefulWidget {
  const OaTaskListPage({super.key});

  @override
  State<OaTaskListPage> createState() => _OaTaskListPageState();
}

class _OaTaskListPageState extends State<OaTaskListPage>
    with SingleTickerProviderStateMixin {
  late final TabController _tabController;
  final _service = WorkflowService();

  List<oaApi.OaServiceV1MyTaskItem> _pending = const [];
  List<oaApi.OaServiceV1MyTaskItem> _done = const [];
  List<oaApi.OaServiceV1MyTaskItem> _submitted = const [];
  bool _loadingPending = true;
  bool _loadingDone = true;
  bool _loadingSubmitted = true;
  bool _withdrawing = false;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 3, vsync: this);
    _tabController.addListener(() {
      if (!_tabController.indexIsChanging) {
        final idx = _tabController.index;
        if (idx == 1 && _loadingDone && _done.isEmpty) _loadDone();
        if (idx == 2 && _loadingSubmitted && _submitted.isEmpty) _loadSubmitted();
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
      _pending = (result is Status)
          ? const []
          : (result as oaApi.OaServiceV1GetMyTasksResponse?)?.items ?? const [];
      _loadingPending = false;
    });
  }

  Future<void> _loadDone() async {
    final result = await _service.doneTasks();
    if (!mounted) return;
    setState(() {
      _done = (result is Status)
          ? const []
          : (result as oaApi.OaServiceV1GetMyTasksResponse?)?.items ?? const [];
      _loadingDone = false;
    });
  }

  Future<void> _loadSubmitted() async {
    final result = await _service.submittedTasks();
    if (!mounted) return;
    setState(() {
      _submitted = (result is Status)
          ? const []
          : (result as oaApi.OaServiceV1GetMyTasksResponse?)?.items ?? const [];
      _loadingSubmitted = false;
    });
  }

  Future<void> _withdraw(int instanceId) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('撤回申请'),
        content: const Text('确认撤回该申请？撤回后审批流程终止。'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('撤回')),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;

    setState(() => _withdrawing = true);
    final result = await _service.withdraw(instanceId: instanceId);
    if (!mounted) return;
    setState(() => _withdrawing = false);
    if (result is Status) {
      ScaffoldMessenger.of(context)
        ..hideCurrentSnackBar()
        ..showSnackBar(SnackBar(content: Text(result.message ?? '撤回失败')));
    } else {
      ScaffoldMessenger.of(context)
        ..hideCurrentSnackBar()
        ..showSnackBar(const SnackBar(content: Text('已撤回')));
      _loadSubmitted();
    }
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
        actions: [
          PopupMenuButton<String>(
            onSelected: (value) =>
                GoRouter.of(context).goNamed(value),
            itemBuilder: (ctx) => const [
              PopupMenuItem(value: RouteNames.oaLeave, child: Text('请假申请')),
              PopupMenuItem(value: RouteNames.oaExpense, child: Text('费用报销')),
              PopupMenuItem(value: RouteNames.oaBusinessTrip, child: Text('出差申请')),
              PopupMenuItem(value: RouteNames.oaOvertime, child: Text('加班申请')),
              PopupMenuItem(value: RouteNames.oaSealApplication, child: Text('用印申请')),
              PopupMenuItem(value: RouteNames.oaOuting, child: Text('外出申请')),
              PopupMenuItem(value: RouteNames.oaDirectory, child: Text('通讯录')),
              PopupMenuItem(value: RouteNames.oaSubmitApply, child: Text('通用申请')),
            ],
          ),
        ],
        bottom: TabBar(
          controller: _tabController,
          tabs: [
            Tab(text: loc.oaTaskListTitlePending),
            const Tab(text: '已办'),
            Tab(text: loc.oaTaskListTitleSubmitted),
          ],
        ),
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => GoRouter.of(context).goNamed(RouteNames.oaSubmitApply),
        icon: const Icon(Icons.add),
        label: Text(loc.oaTaskListFabApply),
      ),
      body: _withdrawing
          ? const Center(child: CircularProgressIndicator())
          : TabBarView(
              controller: _tabController,
              children: [
                _buildPendingList(),
                _buildSimpleList(_done, _loadingDone, _loadDone),
                _buildSubmittedList(),
              ],
            ),
    );
  }

  Widget _buildPendingList() {
    return _buildSimpleList(_pending, _loadingPending, _loadPending, tappable: true);
  }

  Widget _buildSubmittedList() {
    final theme = Theme.of(context);
    final loc = S.of(context);

    if (_loadingSubmitted) {
      return const Center(child: CircularProgressIndicator());
    }
    if (_submitted.isEmpty) {
      return Center(
        child: Text(loc.oaTaskListEmpty,
            style: TextStyle(color: theme.colorScheme.onSurface.withAlpha(120))),
      );
    }
    return RefreshIndicator(
      onRefresh: _loadSubmitted,
      child: ListView.separated(
        physics: const AlwaysScrollableScrollPhysics(),
        itemCount: _submitted.length,
        separatorBuilder: (_, __) => const Divider(height: 1),
        itemBuilder: (context, i) {
          final row = _submitted[i];
          final active = row.statusLabel == '进行中';
          return ListTile(
            title: Text('申请 #${row.instanceId ?? ''}'),
            subtitle: Text(
              '${loc.oaTaskListStatus}: ${row.statusLabel ?? ''}',
              style: const TextStyle(fontSize: 12),
            ),
            trailing: active
                ? TextButton.icon(
                    onPressed: () => _withdraw(row.instanceId ?? 0),
                    icon: const Icon(Icons.undo, size: 16),
                    label: const Text('撤回'),
                  )
                : Text(
                    '${row.createdAt ?? ''}',
                    style: const TextStyle(fontSize: 11),
                  ),
          );
        },
      ),
    );
  }

  Widget _buildSimpleList(
    List<oaApi.OaServiceV1MyTaskItem> items,
    bool loading,
    Future<void> Function() onRefresh, {
    bool tappable = false,
  }) {
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
          final row = items[i];
          final int taskId = row.taskId ?? 0;
          return ListTile(
            title: Text('申请 #${row.instanceId ?? ''}'),
            subtitle: Text(
              '${loc.oaTaskListStatus}: ${row.statusLabel ?? ''}',
              style: const TextStyle(fontSize: 12),
            ),
            trailing: Text(
              '${row.createdAt ?? ''}',
              style: const TextStyle(fontSize: 11),
            ),
            onTap: tappable && taskId > 0
                ? () {
                    GoRouter.of(context).goNamed(RouteNames.oaTaskDetail,
                        pathParameters: {'id': taskId.toString()});
                  }
                : null,
          );
        },
      ),
    );
  }
}
