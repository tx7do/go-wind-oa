// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:dio/dio.dart';
import 'package:retrofit/retrofit.dart';

import '../models/audit_task_request.dart';
import '../models/create_workflow_definition_request.dart';
import '../models/enum0.dart';
import '../models/enum1.dart';
import '../models/get_my_tasks_response.dart';
import '../models/get_task_response.dart';
import '../models/list_type.dart';
import '../models/list_workflow_definition_response.dart';
import '../models/submit_apply_request.dart';
import '../models/submit_apply_response.dart';
import '../models/workflow_definition.dart';

part 'workflow_service_client.g.dart';

@RestApi()
abstract class WorkflowServiceClient {
  factory WorkflowServiceClient(Dio dio, {String? baseUrl}) = _WorkflowServiceClient;

  /// 提交申请：发起一个工作流实例，按定义的首节点生成首条待办任务并通知该审批人。
  @POST('/admin/v1/oa/workflow/apply')
  Future<SubmitApplyResponse> workflowServiceSubmitApply({
    @Body() required SubmitApplyRequest body,
  });

  /// 查询工作流定义列表（管理端）。.
  ///
  /// [pagingPage] - 当前页码（从1开始，默认1）.
  ///
  /// [pagingPageSize] - 每页条数（默认10，建议设置上限如100）.
  ///
  /// [pagingOffset] - 跳过的记录数（从0开始，默认0）.
  ///
  /// [pagingLimit] - 最多返回的记录数（默认10，建议设置上限如100）.
  ///
  /// [pagingToken] - 上一页最后一条记录的游标（如ID/时间戳+ID，首次请求为空）.
  ///
  /// [pagingNoPaging] - 是否不分页，如果为true，则page和pageSize参数无效。.
  ///
  /// [pagingQuery] - JSON字符串过滤条件，基础语法：{"field1":"val1", "field2___icontains":"val2"}，具体请参见：https://github.com/tx7do/go-crud/tree/main/pagination/filter/README.md.
  ///
  /// [pagingFilter] - Google AIP规范字符串过滤条件.
  ///
  /// [pagingFilterExprType] - 过滤表达式类型.
  ///
  /// [pagingOrderBy] - 排序条件.
  ///
  /// [pagingFieldMask] - 字段掩码，其作用为SELECT中的字段，其语法为使用逗号分隔字段名，例如：id,realName,userName。如果为空则选中所有字段，即SELECT *。.
  @GET('/admin/v1/oa/workflow/definitions')
  Future<ListWorkflowDefinitionResponse> workflowServiceListWorkflowDefinition({
    @Query('paging.page') int? object0,
    @Query('paging.pageSize') int? object1,
    @Query('paging.offset') String? object2,
    @Query('paging.limit') int? object3,
    @Query('paging.token') String? object4,
    @Query('paging.noPaging') bool? object5,
    @Query('paging.query') String? object6,
    @Query('paging.filter') String? object7,
    @Query('paging.filterExpr.type') Enum0? enum0,
    @Query('paging.orderBy') String? object8,
    @Query('paging.fieldMask') String? object9,
  });

  /// 创建工作流定义（管理端）。
  @POST('/admin/v1/oa/workflow/definitions')
  Future<WorkflowDefinition> workflowServiceCreateWorkflowDefinition({
    @Body() required CreateWorkflowDefinitionRequest body,
  });

  /// 获取当前用户的待办 / 已办 / 我的申请列表。.
  ///
  /// [pagingPage] - 当前页码（从1开始，默认1）.
  ///
  /// [pagingPageSize] - 每页条数（默认10，建议设置上限如100）.
  ///
  /// [pagingOffset] - 跳过的记录数（从0开始，默认0）.
  ///
  /// [pagingLimit] - 最多返回的记录数（默认10，建议设置上限如100）.
  ///
  /// [pagingToken] - 上一页最后一条记录的游标（如ID/时间戳+ID，首次请求为空）.
  ///
  /// [pagingNoPaging] - 是否不分页，如果为true，则page和pageSize参数无效。.
  ///
  /// [pagingQuery] - JSON字符串过滤条件，基础语法：{"field1":"val1", "field2___icontains":"val2"}，具体请参见：https://github.com/tx7do/go-crud/tree/main/pagination/filter/README.md.
  ///
  /// [pagingFilter] - Google AIP规范字符串过滤条件.
  ///
  /// [pagingFilterExprType] - 过滤表达式类型.
  ///
  /// [pagingOrderBy] - 排序条件.
  ///
  /// [pagingFieldMask] - 字段掩码，其作用为SELECT中的字段，其语法为使用逗号分隔字段名，例如：id,realName,userName。如果为空则选中所有字段，即SELECT *。.
  @GET('/admin/v1/oa/workflow/my-tasks')
  Future<GetMyTasksResponse> workflowServiceGetMyTasks({
    @Query('listType') ListType? listType,
    @Query('paging.page') int? object10,
    @Query('paging.pageSize') int? object11,
    @Query('paging.offset') String? object12,
    @Query('paging.limit') int? object13,
    @Query('paging.token') String? object14,
    @Query('paging.noPaging') bool? object15,
    @Query('paging.query') String? object16,
    @Query('paging.filter') String? object17,
    @Query('paging.filterExpr.type') Enum1? enum1,
    @Query('paging.orderBy') String? object18,
    @Query('paging.fieldMask') String? object19,
  });

  /// 获取单个待办任务详情：申请标题、申请表单数据、该实例的审批日志轨迹。.
  ///
  ///  鉴权：仅当前指派审批人可查其处于 PENDING 的任务（与 AuditTask 同款授权，.
  ///  IDEQ + AssigneeUserIDEQ(caller) + TaskStatusEQ(PENDING)，见.
  ///  WorkflowTaskRepo.GetDetailByAssignee）。其余字段（definition_id /.
  ///  current_node_index / tenant_id 等）不投影，对齐 MyTaskItem 的最小披露原则。.
  @GET('/admin/v1/oa/workflow/tasks/{taskId}')
  Future<GetTaskResponse> workflowServiceGetTask({
    @Path('taskId') required int taskId,
  });

  /// 审批/驳回/转办当前待办任务。状态机在此推进实例状态、生成下一节点任务或终结实例。.
  ///
  /// [taskId] - 绑定路径参数 {task_id}。.
  @POST('/admin/v1/oa/workflow/tasks/{taskId}/audit')
  Future<void> workflowServiceAuditTask({
    @Path('taskId') required int taskId,
    @Body() required AuditTaskRequest body,
  });
}
