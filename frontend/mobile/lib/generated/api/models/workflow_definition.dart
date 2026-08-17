// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'workflow_definition_definition_status.dart';

part 'workflow_definition.g.dart';

@JsonSerializable()
class WorkflowDefinition {
  const WorkflowDefinition({
    this.id,
    this.name,
    this.code,
    this.version,
    this.description,
    this.nodeConfig,
    this.formSchema,
    this.definitionStatus,
    this.tenantId,
    this.tenantName,
    this.createdBy,
    this.updatedBy,
    this.deletedBy,
    this.createdAt,
    this.updatedAt,
    this.deletedAt,
  });
  
  factory WorkflowDefinition.fromJson(Map<String, Object?> json) => _$WorkflowDefinitionFromJson(json);
  
  final int? id;
  final String? name;
  final String? code;
  final int? version;
  final String? description;

  /// 节点配置，原始 JSON 文本。结构由引擎 internal/workflow 包解析。
  final String? nodeConfig;

  /// 动态表单 schema，原始 JSON 文本，供前端按定义渲染。
  final String? formSchema;
  final WorkflowDefinitionDefinitionStatus? definitionStatus;

  /// ---- 租户 / 审计字段（与 go-wind-cms 固定字段号对齐） ----
  final int? tenantId;
  final String? tenantName;
  final int? createdBy;
  final int? updatedBy;
  final int? deletedBy;
  final DateTime? createdAt;
  final DateTime? updatedAt;
  final DateTime? deletedAt;

  Map<String, Object?> toJson() => _$WorkflowDefinitionToJson(this);
}
