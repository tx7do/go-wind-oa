// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'my_task_item.g.dart';

@JsonSerializable()
class MyTaskItem {
  const MyTaskItem({
    this.taskId,
    this.logId,
    this.instanceId,
    this.title,
    this.statusLabel,
    this.actionLabel,
    this.comment,
    this.occurredAt,
  });
  
  factory MyTaskItem.fromJson(Map<String, Object?> json) => _$MyTaskItemFromJson(json);
  
  /// 各字段按 list_type 选择性填充。
  final int? taskId;
  final int? logId;
  final int? instanceId;
  final String? title;
  final String? statusLabel;
  final String? actionLabel;
  final String? comment;
  final DateTime? occurredAt;

  Map<String, Object?> toJson() => _$MyTaskItemToJson(this);
}
