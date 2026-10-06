import 'package:flutter/material.dart';

import 'package:flutter_app/generated/l10n.dart';
import 'package:flutter_app/src/features/oa/services/workflow_service.dart';
import 'package:flutter_app/src/core/transport/http/status.dart';
import 'package:flutter_app/src/app_router/route_names.dart';
import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;
import 'package:go_router/go_router.dart';
import 'package:flutter_app/src/core/utilities/date_time.dart';

/// 工作流任务列表页（三 Tab：“待我审批” / “已办” / “我发起的”）。
///
/// 列表数据经 [WorkflowService] 直接调用获取（Future + setState），分页加载
/// （每页 [_pageSize] 条，滚动近底部自动加载下一页，下拉刷新重置到第一页）。
/// “待我审批”项带 taskId，点击进入详情执行审批；“我发起的”进行中的项可撤回
/// （引擎侧校验：仅申请人本人 + 实例进行中）。AppBar 菜单提供请假/报销/通用申请入口。
class OaTaskListPage extends StatefulWidget {
  const OaTaskListPage({super.key});

  @override
  State<OaTaskListPage> createState() => _OaTaskListPageState();
}

/// 单个 Tab 的分页列表状态。
class _TabList {
  _TabList(this.loadFirstPage);

  final Future<void> Function(bool refresh) loadFirstPage;

  List<oaApi.OaServiceV1MyTaskItem> items = const [];
  int page = 0;
  bool initialLoading = true;
  bool loadingMore = false;
  /// 末页标记：某页返回条数不足 _pageSize 即认为到底
  bool reachedEnd = false;
  final ScrollController controller = ScrollController();

  void dispose() => controller.dispose();
}

class _OaTaskListPageState extends State<OaTaskListPage>
    with SingleTickerProviderStateMixin {
  static const int _pageSize = 20;

  late final TabController _tabController;
  final _service = WorkflowService();

  late final _TabList _pending = _TabList(_loadPending)
    ..controller.addListener(() => _onScroll(_pending));
  late final _TabList _done = _TabList(_loadDone)
    ..controller.addListener(() => _onScroll(_done));
  late final _TabList _submitted = _TabList(_loadSubmitted)
    ..controller.addListener(() => _onScroll(_submitted));

  bool _withdrawing = false;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 3, vsync: this);
    _tabController.addListener(() {
      if (!_tabController.indexIsChanging) {
        final idx = _tabController.index;
        if (idx == 1 && _done.initialLoading) _load(_done, refresh: true);
        if (idx == 2 && _submitted.initialLoading) {
          _load(_submitted, refresh: true);
        }
      }
    });
    _load(_pending, refresh: true);
  }

  @override
  void dispose() {
    _tabController.dispose();
    _pending.dispose();
    _done.dispose();
    _submitted.dispose();
    super.dispose();
  }

  // ============ 数据加载 ============

  /// 按需加载：refresh 重置到第一页，否则翻下一页（带加载中/到底守卫）
  Future<void> _load(_TabList tab, {bool refresh = false}) async {
    if (tab.loadingMore) return;
    if (!refresh && tab.reachedEnd) return;
    tab.loadingMore = true;
    await tab.loadFirstPage(refresh);
  }

  Future<void> _loadPending(bool refresh) async {
    final result = await _service.pendingTasks(
        page: refresh ? 1 : _pending.page + 1, pageSize: _pageSize);
    _applyPage(_pending, result, refresh);
  }

  Future<void> _loadDone(bool refresh) async {
    final result = await _service.doneTasks(
        page: refresh ? 1 : _done.page + 1, pageSize: _pageSize);
    _applyPage(_done, result, refresh);
  }

  Future<void> _loadSubmitted(bool refresh) async {
    final result = await _service.submittedTasks(
        page: refresh ? 1 : _submitted.page + 1, pageSize: _pageSize);
    _applyPage(_submitted, result, refresh);
  }

  void _applyPage(_TabList tab, dynamic result, bool refresh) {
    if (!mounted) return;
    final resp = (result is Status)
        ? null
        : result as oaApi.OaServiceV1GetMyTasksResponse?;
    final newItems = resp?.items ?? const <oaApi.OaServiceV1MyTaskItem>[];
    setState(() {
      tab.items = refresh
          ? newItems
          : [...tab.items, ...newItems];
      tab.page = refresh ? 1 : tab.page + 1;
      tab.initialLoading = false;
      tab.loadingMore = false;
      tab.reachedEnd = newItems.length < _pageSize;
    });
  }

  /// 滚动近底部且未到底时加载下一页
  void _onScroll(_TabList tab) {
    if (!tab.controller.hasClients) return;
    if (tab.loadingMore || tab.reachedEnd || tab.initialLoading) return;
    if (tab.controller.position.extentAfter < 400) {
      _load(tab);
    }
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
      _load(_submitted, refresh: true);
    }
  }

  // ============ UI ============

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
                GoRouter.of(context).pushNamed(value),
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
        onPressed: () => GoRouter.of(context).pushNamed(RouteNames.oaSubmitApply),
        icon: const Icon(Icons.add),
        label: Text(loc.oaTaskListFabApply),
      ),
      body: _withdrawing
          ? const Center(child: CircularProgressIndicator())
          : TabBarView(
              controller: _tabController,
              children: [
                _buildPendingList(),
                _buildDoneList(),
                _buildSubmittedList(),
              ],
            ),
    );
  }

  Widget _buildPendingList() {
    return _buildTaskList(_pending, tappable: true);
  }

  Widget _buildDoneList() {
    return _buildTaskList(_done);
  }

  Widget _buildSubmittedList() {
    final theme = Theme.of(context);
    final loc = S.of(context);

    if (_submitted.initialLoading) {
      return const Center(child: CircularProgressIndicator());
    }
    if (_submitted.items.isEmpty) {
      return Center(
        child: Text(loc.oaTaskListEmpty,
            style: TextStyle(color: theme.colorScheme.onSurface.withAlpha(120))),
      );
    }
    return RefreshIndicator(
      onRefresh: () => _load(_submitted, refresh: true),
      child: ListView.separated(
        controller: _submitted.controller,
        physics: const AlwaysScrollableScrollPhysics(),
        itemCount: _submitted.items.length + 1,
        separatorBuilder: (_, _) => const Divider(height: 1),
        itemBuilder: (context, i) {
          if (i == _submitted.items.length) {
            return _buildListFooter(_submitted);
          }
          final row = _submitted.items[i];
          // 实例状态走枚举本地化；SUSPENDED 等无词条态回退服务端文案
          final statusText = switch (row.instanceStatus) {
            oaApi.OaServiceV1WorkflowInstance$InstanceStatus.pending =>
              loc.oaInstanceStatusPending,
            oaApi.OaServiceV1WorkflowInstance$InstanceStatus.approved =>
              loc.oaInstanceStatusApproved,
            oaApi.OaServiceV1WorkflowInstance$InstanceStatus.rejected =>
              loc.oaInstanceStatusRejected,
            oaApi.OaServiceV1WorkflowInstance$InstanceStatus.withdrawn =>
              loc.oaInstanceStatusWithdrawn,
            _ => row.statusLabel ?? '-',
          };
          final active =
              row.instanceStatus == oaApi.OaServiceV1WorkflowInstance$InstanceStatus.pending;
          return ListTile(
            title: Text('申请 #${row.instanceId ?? ''}'),
            subtitle: Text(
              '${loc.oaTaskListStatus}: $statusText',
              style: const TextStyle(fontSize: 12),
            ),
            trailing: active
                ? TextButton.icon(
                    onPressed: () => _withdraw(row.instanceId ?? 0),
                    icon: const Icon(Icons.undo, size: 16),
                    label: const Text('撤回'),
                  )
                : Text(
                    DateTimeUtils.formatDateTime(row.createdAt),
                    style: const TextStyle(fontSize: 11),
                  ),
          );
        },
      ),
    );
  }

  Widget _buildTaskList(
    _TabList tab, {
    bool tappable = false,
  }) {
    final theme = Theme.of(context);
    final loc = S.of(context);

    if (tab.initialLoading) {
      return const Center(child: CircularProgressIndicator());
    }
    if (tab.items.isEmpty) {
      return Center(
        child: Text(loc.oaTaskListEmpty,
            style: TextStyle(color: theme.colorScheme.onSurface.withAlpha(120))),
      );
    }
    return RefreshIndicator(
      onRefresh: () => _load(tab, refresh: true),
      child: ListView.separated(
        controller: tab.controller,
        physics: const AlwaysScrollableScrollPhysics(),
        itemCount: tab.items.length + 1,
        separatorBuilder: (_, _) => const Divider(height: 1),
        itemBuilder: (context, i) {
          if (i == tab.items.length) {
            return _buildListFooter(tab);
          }
          final row = tab.items[i];
          final int taskId = row.taskId ?? 0;
          // 已办 Tab 的动作走枚举本地化，待办保持服务端“待办”文案
          final statusText = switch (row.auditAction) {
            oaApi.OaServiceV1WorkflowLog$LogAction.submit => loc.oaLogActionSubmit,
            oaApi.OaServiceV1WorkflowLog$LogAction.approve => loc.oaLogActionApprove,
            oaApi.OaServiceV1WorkflowLog$LogAction.reject => loc.oaLogActionReject,
            oaApi.OaServiceV1WorkflowLog$LogAction.forward => loc.oaLogActionForward,
            oaApi.OaServiceV1WorkflowLog$LogAction.withdraw => loc.oaLogActionWithdraw,
            _ => row.statusLabel ?? '-',
          };
          return ListTile(
            title: Text('申请 #${row.instanceId ?? ''}'),
            subtitle: Text(
              '${loc.oaTaskListStatus}: $statusText',
              style: const TextStyle(fontSize: 12),
            ),
            trailing: Text(
              DateTimeUtils.formatDateTime(row.createdAt),
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

  /// 列表尾部：加载中指示器 / 到底提示
  Widget _buildListFooter(_TabList tab) {
    if (tab.reachedEnd) {
      return const Padding(
        padding: EdgeInsets.symmetric(vertical: 16),
        child: Center(
          child: Text('没有更多了',
              style: TextStyle(fontSize: 12, color: Colors.grey)),
        ),
      );
    }
    return const Padding(
      padding: EdgeInsets.symmetric(vertical: 16),
      child: Center(child: SizedBox(
        width: 20,
        height: 20,
        child: CircularProgressIndicator(strokeWidth: 2),
      )),
    );
  }
}
