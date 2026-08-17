// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'audit_log_entry.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

AuditLogEntry _$AuditLogEntryFromJson(Map<String, dynamic> json) =>
    AuditLogEntry(
      actionLabel: json['actionLabel'] as String?,
      comment: json['comment'] as String?,
      occurredAt: json['occurredAt'] == null
          ? null
          : DateTime.parse(json['occurredAt'] as String),
    );

Map<String, dynamic> _$AuditLogEntryToJson(AuditLogEntry instance) =>
    <String, dynamic>{
      'actionLabel': instance.actionLabel,
      'comment': instance.comment,
      'occurredAt': instance.occurredAt?.toIso8601String(),
    };
