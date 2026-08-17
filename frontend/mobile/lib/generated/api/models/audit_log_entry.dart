// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'audit_log_entry.g.dart';

/// AuditLogEntry 审批日志轨迹条目。投影自 WorkflowLog，仅含审批类动作.
///  （APPROVE / REJECT / FORWARD），与 ListByActor 的过滤口径一致。.
@JsonSerializable()
class AuditLogEntry {
  const AuditLogEntry({
    this.actionLabel,
    this.comment,
    this.occurredAt,
  });
  
  factory AuditLogEntry.fromJson(Map<String, Object?> json) => _$AuditLogEntryFromJson(json);
  
  final String? actionLabel;
  final String? comment;
  final DateTime? occurredAt;

  Map<String, Object?> toJson() => _$AuditLogEntryToJson(this);
}
