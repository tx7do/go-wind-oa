// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'audit_task_request.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

AuditTaskRequest _$AuditTaskRequestFromJson(Map<String, dynamic> json) =>
    AuditTaskRequest(
      taskId: (json['taskId'] as num?)?.toInt(),
      action: json['action'] == null
          ? null
          : AuditTaskRequestAction.fromJson(json['action'] as String),
      comment: json['comment'] as String?,
      forwardToUserId: (json['forwardToUserId'] as num?)?.toInt(),
    );

Map<String, dynamic> _$AuditTaskRequestToJson(AuditTaskRequest instance) =>
    <String, dynamic>{
      'taskId': instance.taskId,
      'action': _$AuditTaskRequestActionEnumMap[instance.action],
      'comment': instance.comment,
      'forwardToUserId': instance.forwardToUserId,
    };

const _$AuditTaskRequestActionEnumMap = {
  AuditTaskRequestAction.approve: 'APPROVE',
  AuditTaskRequestAction.reject: 'REJECT',
  AuditTaskRequestAction.forward: 'FORWARD',
  AuditTaskRequestAction.$unknown: r'$unknown',
};
