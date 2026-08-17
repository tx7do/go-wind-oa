// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'audit_log_entry.dart';

part 'get_task_response.g.dart';

/// GetTaskResponse 投影待办任务详情。字段对齐详情页渲染所需，最小披露：.
///  申请标题、申请表单数据（原始 JSON 文本，引擎不解释）、该实例的审批日志轨迹。.
@JsonSerializable()
class GetTaskResponse {
  const GetTaskResponse({
    this.taskId,
    this.instanceId,
    this.title,
    this.formData,
    this.history,
  });
  
  factory GetTaskResponse.fromJson(Map<String, Object?> json) => _$GetTaskResponseFromJson(json);
  
  final int? taskId;
  final int? instanceId;
  final String? title;
  final String? formData;
  final List<AuditLogEntry>? history;

  Map<String, Object?> toJson() => _$GetTaskResponseToJson(this);
}
