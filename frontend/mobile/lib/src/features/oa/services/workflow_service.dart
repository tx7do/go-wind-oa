import 'package:dio/dio.dart' show DioException;
import 'package:get_it/get_it.dart' show GetIt;
import 'package:cached_query/cached_query.dart' show Mutation, Query;

import 'package:flutter_app/src/core/services/base_service.dart';

// 生成代码由 buf.app.dart.gen.yaml 生成于
// generated/api/app/service/v1/index.dart。该包含 ApiClient（聚合所有
// app HTTP service client）及 OA 工作流的消息类型（OaServiceV1* 前缀）。
import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

/// OA 工作流服务。
///
/// 对齐 cms TagService 模式：extends [BaseService]，经 [GetIt] 取生成的
/// [oaApi.ApiClient].workflowService；用 cached_query 的 Query / Mutation
/// 管理列表/写操作。
///
/// 列表 query key 编码：
///   'oa-pending-tasks'     —— 待我审批（listType=PENDING）
///   'oa-done-tasks'        —— 已办（listType=DONE）
///   'oa-submitted-tasks'   —— 我发起的（listType=SUBMITTED）
class WorkflowService extends BaseService {
  WorkflowService() : super(tag: 'WorkflowService');

  oaApi.WorkflowServiceClient get _api =>
      GetIt.instance<oaApi.ApiClient>().workflowService;

  // ─── Queries ──────────────────────────────────────────

  /// 待我审批列表 Query。
  Query<List<oaApi.OaServiceV1MyTaskItem>> pendingTasksQuery() {
    return Query<List<oaApi.OaServiceV1MyTaskItem>>(
      key: 'oa-pending-tasks',
      queryFn: () async {
        final resp = await _api.getMyTasks(oaApi.OaServiceV1GetMyTasksRequest(
          listType: oaApi.OaServiceV1ListType.pending,
        ));
        return resp.items ?? const <oaApi.OaServiceV1MyTaskItem>[];
      },
    );
  }

  /// 我发起的列表 Query。
  Query<List<oaApi.OaServiceV1MyTaskItem>> submittedTasksQuery() {
    return Query<List<oaApi.OaServiceV1MyTaskItem>>(
      key: 'oa-submitted-tasks',
      queryFn: () async {
        final resp = await _api.getMyTasks(oaApi.OaServiceV1GetMyTasksRequest(
          listType: oaApi.OaServiceV1ListType.submitted,
        ));
        return resp.items ?? const <oaApi.OaServiceV1MyTaskItem>[];
      },
    );
  }

  // ─── Mutations ────────────────────────────────────────

  /// 审批 Mutation（同意/驳回/转交）。完成后待办与我发起的列表自动失效刷新。
  Mutation<void,
      ({int taskId, oaApi.OaServiceV1AuditAction action, String? comment, int? forwardToUserId})>
      auditMutation() {
    return Mutation<void,
        ({int taskId, oaApi.OaServiceV1AuditAction action, String? comment, int? forwardToUserId})>(
      mutationFn: (p) => _api.auditTask(oaApi.OaServiceV1AuditTaskRequest(
        taskId: p.taskId,
        action: p.action,
        comment: p.comment,
        forwardTo: p.forwardToUserId,
      )),
      invalidateQueries: ['oa-pending-tasks', 'oa-submitted-tasks'],
    );
  }

  /// 撤回 Mutation。完成后“我发起的”与“待办”列表自动失效刷新。
  Mutation<void, int> withdrawMutation() {
    return Mutation<void, int>(
      mutationFn: (instanceId) => _api.withdrawApply(
          oaApi.OaServiceV1WithdrawApplyRequest(instanceId: instanceId)),
      invalidateQueries: ['oa-pending-tasks', 'oa-submitted-tasks'],
    );
  }

  // ─── 直接调用方法（页面用 Future 而非 Stream） ──

  /// 待我审批列表（直接调用）。
  Future<dynamic> pendingTasks() async {
    try {
      return await _api.getMyTasks(oaApi.OaServiceV1GetMyTasksRequest(
        listType: oaApi.OaServiceV1ListType.pending,
      ));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 已办列表（直接调用）。
  Future<dynamic> doneTasks() async {
    try {
      return await _api.getMyTasks(oaApi.OaServiceV1GetMyTasksRequest(
        listType: oaApi.OaServiceV1ListType.done,
      ));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 我发起的列表（直接调用）。
  Future<dynamic> submittedTasks() async {
    try {
      return await _api.getMyTasks(oaApi.OaServiceV1GetMyTasksRequest(
        listType: oaApi.OaServiceV1ListType.submitted,
      ));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 审批（同意/驳回/转交）。供详情页按钮使用。
  Future<dynamic> audit({
    required int taskId,
    required oaApi.OaServiceV1AuditAction action,
    String? comment,
    int? forwardToUserId,
  }) async {
    try {
      return await _api.auditTask(oaApi.OaServiceV1AuditTaskRequest(
        taskId: taskId,
        action: action,
        comment: comment,
        forwardTo: forwardToUserId,
      ));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 撤回申请（仅本人进行中的实例）。供“我发起的”列表撤回按钮使用。
  Future<dynamic> withdraw({required int instanceId}) async {
    try {
      return await _api.withdrawApply(
          oaApi.OaServiceV1WithdrawApplyRequest(instanceId: instanceId));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 提交通用申请（按流程定义 code/version）。请假/报销走各自的业务服务，
  /// 它们会在服务端自动创建对应流程实例。
  Future<dynamic> submitApply({
    required String code,
    required int version,
    required String formData,
  }) async {
    try {
      return await _api.submitApply(oaApi.OaServiceV1SubmitApplyRequest(
        code: code,
        version: version,
        formData: formData,
      ));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 获取申请表单定义（按流程 code+version，仅 ENABLED 定义）。
  /// 返回 form_schema JSON 字段描述数组；空串表示无表单定义（调用方回退自由 JSON）。
  Future<dynamic> fetchApplyForm({required String code, required int version}) async {
    try {
      return await _api.getApplyForm(
          oaApi.OaServiceV1GetApplyFormRequest(code: code, version: version));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 获取单个待办任务详情（含申请表单与审批历史）。
  Future<dynamic> getTaskDetail({required int taskId}) async {
    try {
      return await _api.getTask(oaApi.OaServiceV1GetTaskRequest(id: taskId));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }
}
