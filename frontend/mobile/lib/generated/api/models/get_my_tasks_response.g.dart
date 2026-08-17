// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'get_my_tasks_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

GetMyTasksResponse _$GetMyTasksResponseFromJson(Map<String, dynamic> json) =>
    GetMyTasksResponse(
      items: (json['items'] as List<dynamic>?)
          ?.map((e) => MyTaskItem.fromJson(e as Map<String, dynamic>))
          .toList(),
      total: json['total'] as String?,
    );

Map<String, dynamic> _$GetMyTasksResponseToJson(GetMyTasksResponse instance) =>
    <String, dynamic>{'items': instance.items, 'total': instance.total};
