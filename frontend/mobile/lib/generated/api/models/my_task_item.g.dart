// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'my_task_item.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

MyTaskItem _$MyTaskItemFromJson(Map<String, dynamic> json) => MyTaskItem(
  taskId: (json['taskId'] as num?)?.toInt(),
  logId: (json['logId'] as num?)?.toInt(),
  instanceId: (json['instanceId'] as num?)?.toInt(),
  title: json['title'] as String?,
  statusLabel: json['statusLabel'] as String?,
  actionLabel: json['actionLabel'] as String?,
  comment: json['comment'] as String?,
  occurredAt: json['occurredAt'] == null
      ? null
      : DateTime.parse(json['occurredAt'] as String),
);

Map<String, dynamic> _$MyTaskItemToJson(MyTaskItem instance) =>
    <String, dynamic>{
      'taskId': instance.taskId,
      'logId': instance.logId,
      'instanceId': instance.instanceId,
      'title': instance.title,
      'statusLabel': instance.statusLabel,
      'actionLabel': instance.actionLabel,
      'comment': instance.comment,
      'occurredAt': instance.occurredAt?.toIso8601String(),
    };
