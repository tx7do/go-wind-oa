import 'package:dio/dio.dart' show DioException;
import 'package:get_it/get_it.dart' show GetIt;
import 'package:cached_query/cached_query.dart' show Mutation, Query;

import 'package:flutter_app/src/core/services/base_service.dart';
import 'package:flutter_app/src/core/services/pagination_query.dart';

// 生成代码由 buf.flutter.oa.dart.gen.yaml 生成于
// generated/api/app/service/v1/index.dart。该包含 ApiClient（聚合所有
// app HTTP service client）及 OA 工作流的消息类型（OaServiceV1* 前缀）。
import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

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
  Query<List<oaApi.OaServiceV1MyTaskItem>> pendingTasksQuery([PaginationQuery? query]) {
    final q = query ?? const PaginationQuery();
    return Query<List<oaApi.OaServiceV1MyTaskItem>>(
      key: 'oa-pending-tasks',
      queryFn: () async {
        final resp = await _api.getMyTasks(
            _toMyTasksRequest(q, oaApi.OaServiceV1GetMyTasksRequest$ListType.pending));
        return resp.items ?? const <oaApi.OaServiceV1MyTaskItem>[];
      },
    );
  }

  /// 我发起的列表 Query。
  Query<List<oaApi.OaServiceV1MyTaskItem>> submittedTasksQuery([PaginationQuery? query]) {
    final q = query ?? const PaginationQuery();
    return Query<List<oaApi.OaServiceV1MyTaskItem>>(
      key: 'oa-submitted-tasks',
      queryFn: () async {
        final resp = await _api.getMyTasks(
            _toMyTasksRequest(q, oaApi.OaServiceV1GetMyTasksRequest$ListType.submitted));
        return resp.items ?? const <oaApi.OaServiceV1MyTaskItem>[];
      },
    );
  }

  // ─── Mutations ────────────────────────────────────────

  /// 审批 Mutation（同意/驳回/转交）。
  ///
  /// invalidateQueries 指向待办列表，使审批完成后列表自动刷新。
  Mutation<void, ({int taskId, oaApi.OaServiceV1AuditTaskRequest$AuditAction action, String? comment, int? forwardToUserId})>
      auditMutation() {
    return Mutation<void,
        ({int taskId, oaApi.OaServiceV1AuditTaskRequest$AuditAction action, String? comment, int? forwardToUserId})>(
      mutationFn: (p) => _api.auditTask(oaApi.OaServiceV1AuditTaskRequest(
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
  Mutation<oaApi.OaServiceV1SubmitApplyResponse,
      ({String definitionCode, int definitionVersion, String title, String formData})>
      submitApplyMutation() {
    return Mutation<oaApi.OaServiceV1SubmitApplyResponse,
        ({String definitionCode, int definitionVersion, String title, String formData})>(
      mutationFn: (p) => _api.submitApply(oaApi.OaServiceV1SubmitApplyRequest(
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
      return await _api.getMyTasks(_toMyTasksRequest(q, oaApi.OaServiceV1GetMyTasksRequest$ListType.pending));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 我发起的列表（直接调用）。
  Future<dynamic> submittedTasks([PaginationQuery? query]) async {
    final q = query ?? const PaginationQuery();
    try {
      return await _api.getMyTasks(_toMyTasksRequest(q, oaApi.OaServiceV1GetMyTasksRequest$ListType.submitted));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 审批（同意/驳回/转交）。直接调用版，供详情页按钮使用。
  Future<dynamic> audit({
    required int taskId,
    required oaApi.OaServiceV1AuditTaskRequest$AuditAction action,
    String? comment,
    int? forwardToUserId,
  }) async {
    try {
      return await _api.auditTask(oaApi.OaServiceV1AuditTaskRequest(
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
      return await _api.submitApply(oaApi.OaServiceV1SubmitApplyRequest(
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
      return await _api.getTask(oaApi.OaServiceV1GetTaskRequest(taskId: taskId));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  // ─── helpers ─────────────────────────────────────────

  oaApi.OaServiceV1GetMyTasksRequest _toMyTasksRequest(
      PaginationQuery q, oaApi.OaServiceV1GetMyTasksRequest$ListType lt) {
    return oaApi.OaServiceV1GetMyTasksRequest(
      listType: lt,
      paging: q.toPagingRequest(),
    );
  }
}
