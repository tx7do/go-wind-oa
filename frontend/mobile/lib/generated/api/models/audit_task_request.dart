// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'audit_task_request_action.dart';

part 'audit_task_request.g.dart';

@JsonSerializable()
class AuditTaskRequest {
  const AuditTaskRequest({
    this.taskId,
    this.action,
    this.comment,
    this.forwardToUserId,
  });
  
  factory AuditTaskRequest.fromJson(Map<String, Object?> json) => _$AuditTaskRequestFromJson(json);
  
  /// 绑定路径参数 {task_id}。
  final int? taskId;
  final AuditTaskRequestAction? action;
  final String? comment;

  /// 仅 action == FORWARD 时有效：被转办人 ID。
  final int? forwardToUserId;

  Map<String, Object?> toJson() => _$AuditTaskRequestToJson(this);
}
