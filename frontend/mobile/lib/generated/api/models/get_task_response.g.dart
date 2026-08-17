// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'get_task_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

GetTaskResponse _$GetTaskResponseFromJson(Map<String, dynamic> json) =>
    GetTaskResponse(
      taskId: (json['taskId'] as num?)?.toInt(),
      instanceId: (json['instanceId'] as num?)?.toInt(),
      title: json['title'] as String?,
      formData: json['formData'] as String?,
      history: (json['history'] as List<dynamic>?)
          ?.map((e) => AuditLogEntry.fromJson(e as Map<String, dynamic>))
          .toList(),
    );

Map<String, dynamic> _$GetTaskResponseToJson(GetTaskResponse instance) =>
    <String, dynamic>{
      'taskId': instance.taskId,
      'instanceId': instance.instanceId,
      'title': instance.title,
      'formData': instance.formData,
      'history': instance.history,
    };
