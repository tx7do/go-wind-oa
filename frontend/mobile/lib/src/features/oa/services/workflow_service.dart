import 'package:dio/dio.dart' show DioException;
import 'package:get_it/get_it.dart' show GetIt;
import 'package:cached_query/cached_query.dart' show Mutation, Query;

import 'package:flutter_app/src/core/services/base_service.dart';
import 'package:flutter_app/src/core/services/pagination_query.dart';

// 生成代码的导入路径，由 swagger_parser 消费
// backend/app/oa/service/cmd/server/assets/openapi.yaml 生成。
// 若实际生成的路径不同，调整此处 import 即可。
import 'package:flutter_app/generated/api/oa/v1/index.dart' as oaApi;

/// OA 工作流服务。
///
/// 对齐 cms TagService 模式：extends [BaseService]，经 [GetIt] 取生成的
/// [oaApi.ApiClient].workflowService；用 cached_query 的 Query / Mutation
/// 管理列表/写操作。
///
/// 关键：审批 Mutation 的 invalidateQueries 指向待办列表的 query key，
/// 使审批/驳回/转交后，待办列表下次读取自动重新拉取——即"审批完一笔，
/// 待办立刻减少一笔"的刷新语义。
///
/// 列表 query key 编码：
///   'oa-pending-tasks'     —— 待我审批（listType=PENDING）
///   'oa-submitted-tasks'   —— 我发起的（listType=SUBMITTED）
class WorkflowService extends BaseService {
  WorkflowService() : super(tag: 'WorkflowService');

  oaApi.WorkflowServiceClient get _api =>
      GetIt.instance<oaApi.ApiClient>().workflowService;

  // ─── Queries ──────────────────────────────────────────

  /// 待我审批列表 Query。
  Query<List<oaApi.MyTaskItem>> pendingTasksQuery([PaginationQuery? query]) {
    final q = query ?? const PaginationQuery();
    return Query<List<oaApi.MyTaskItem>>(
      key: 'oa-pending-tasks',
      queryFn: () => _api.getMyTasks(_toMyTasksRequest(q, oaApi.GetMyTasksRequest_ListType_PENDING)),
    );
  }

  /// 我发起的列表 Query。
  Query<List<oaApi.MyTaskItem>> submittedTasksQuery([PaginationQuery? query]) {
    final q = query ?? const PaginationQuery();
    return Query<List<oaApi.MyTaskItem>>(
      key: 'oa-submitted-tasks',
      queryFn: () => _api.getMyTasks(_toMyTasksRequest(q, oaApi.GetMyTasksRequest_ListType_SUBMITTED)),
    );
  }

  // ─── Mutations ────────────────────────────────────────

  /// 审批 Mutation（同意/驳回/转交）。
  ///
  /// invalidateQueries 指向待办列表，使审批完成后列表自动刷新。
  Mutation<void, ({int taskId, oaApi.AuditTaskRequest_AuditAction action, String? comment, int? forwardToUserId})>
      auditMutation() {
    return Mutation<void,
        ({int taskId, oaApi.AuditTaskRequest_AuditAction action, String? comment, int? forwardToUserId})>(
      mutationFn: (p) => _api.auditTask(oaApi.AuditTaskRequest(
        taskId: p.taskId,
        action: p.action,
        comment: p.comment,
        forwardToUserId: p.forwardToUserId,
      )),
      invalidateQueries: ['oa-pending-tasks', 'oa-submitted-tasks'],
    );
  }

  /// 提交申请 Mutation。
  ///
  /// invalidateQueries 指向我发起的列表，使提交后该列表刷新出新建实例。
  Mutation<oaApi.SubmitApplyResponse,
      ({String definitionCode, int definitionVersion, String title, String formData})>
      submitApplyMutation() {
    return Mutation<oaApi.SubmitApplyResponse,
        ({String definitionCode, int definitionVersion, String title, String formData})>(
      mutationFn: (p) => _api.submitApply(oaApi.SubmitApplyRequest(
        definitionCode: p.definitionCode,
        definitionVersion: p.definitionVersion,
        title: p.title,
        formData: p.formData,
      )),
      invalidateQueries: ['oa-submitted-tasks'],
    );
  }

  // ─── 直接调用方法（与 cms 各 Service 同构，页面用 Future 而非 Stream） ──

  /// 待我审批列表（直接调用）。
  Future<dynamic> pendingTasks([PaginationQuery? query]) async {
    final q = query ?? const PaginationQuery();
    try {
      return await _api.getMyTasks(_toMyTasksRequest(q, oaApi.GetMyTasksRequest_ListType_PENDING));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 我发起的列表（直接调用）。
  Future<dynamic> submittedTasks([PaginationQuery? query]) async {
    final q = query ?? const PaginationQuery();
    try {
      return await _api.getMyTasks(_toMyTasksRequest(q, oaApi.GetMyTasksRequest_ListType_SUBMITTED));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 审批（同意/驳回/转交）。直接调用版，供详情页按钮使用。
  Future<dynamic> audit({
    required int taskId,
    required oaApi.AuditTaskRequest_AuditAction action,
    String? comment,
    int? forwardToUserId,
  }) async {
    try {
      return await _api.auditTask(oaApi.AuditTaskRequest(
        taskId: taskId,
        action: action,
        comment: comment,
        forwardToUserId: forwardToUserId,
      ));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 提交申请。直接调用版，供提交申请表单使用。
  Future<dynamic> submitApply({
    required String definitionCode,
    required int definitionVersion,
    required String title,
    required String formData,
  }) async {
    try {
      return await _api.submitApply(oaApi.SubmitApplyRequest(
        definitionCode: definitionCode,
        definitionVersion: definitionVersion,
        title: title,
        formData: formData,
      ));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 获取单个待办任务详情。直接调用版，供详情页渲染申请摘要、表单数据、
  /// 审批日志轨迹使用。
  ///
  /// 鉴权由后端 GetTask 服务强制（task 必须指派给 caller 且 PENDING，否则
  /// NotFound/Forbidden）——前端无需二次校验。
  Future<dynamic> getTaskDetail({required int taskId}) async {
    try {
      return await _api.getTask(oaApi.GetTaskRequest(taskId: taskId));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  // ─── helpers ─────────────────────────────────────────

  oaApi.GetMyTasksRequest _toMyTasksRequest(
      PaginationQuery q, oaApi.GetMyTasksRequest_ListType lt) {
    return oaApi.GetMyTasksRequest(
      listType: lt,
      paging: q.toPagingRequest(),
    );
  }
}
