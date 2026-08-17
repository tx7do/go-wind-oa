// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'my_task_item.dart';

part 'get_my_tasks_response.g.dart';

@JsonSerializable()
class GetMyTasksResponse {
  const GetMyTasksResponse({
    this.items,
    this.total,
  });
  
  factory GetMyTasksResponse.fromJson(Map<String, Object?> json) => _$GetMyTasksResponseFromJson(json);
  
  final List<MyTaskItem>? items;
  final String? total;

  Map<String, Object?> toJson() => _$GetMyTasksResponseToJson(this);
}
