// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'workflow_definition.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

WorkflowDefinition _$WorkflowDefinitionFromJson(Map<String, dynamic> json) =>
    WorkflowDefinition(
      id: (json['id'] as num?)?.toInt(),
      name: json['name'] as String?,
      code: json['code'] as String?,
      version: (json['version'] as num?)?.toInt(),
      description: json['description'] as String?,
      nodeConfig: json['nodeConfig'] as String?,
      formSchema: json['formSchema'] as String?,
      definitionStatus: json['definitionStatus'] == null
          ? null
          : WorkflowDefinitionDefinitionStatus.fromJson(
              json['definitionStatus'] as String,
            ),
      tenantId: (json['tenantId'] as num?)?.toInt(),
      tenantName: json['tenantName'] as String?,
      createdBy: (json['createdBy'] as num?)?.toInt(),
      updatedBy: (json['updatedBy'] as num?)?.toInt(),
      deletedBy: (json['deletedBy'] as num?)?.toInt(),
      createdAt: json['createdAt'] == null
          ? null
          : DateTime.parse(json['createdAt'] as String),
      updatedAt: json['updatedAt'] == null
          ? null
          : DateTime.parse(json['updatedAt'] as String),
      deletedAt: json['deletedAt'] == null
          ? null
          : DateTime.parse(json['deletedAt'] as String),
    );

Map<String, dynamic> _$WorkflowDefinitionToJson(
  WorkflowDefinition instance,
) => <String, dynamic>{
  'id': instance.id,
  'name': instance.name,
  'code': instance.code,
  'version': instance.version,
  'description': instance.description,
  'nodeConfig': instance.nodeConfig,
  'formSchema': instance.formSchema,
  'definitionStatus':
      _$WorkflowDefinitionDefinitionStatusEnumMap[instance.definitionStatus],
  'tenantId': instance.tenantId,
  'tenantName': instance.tenantName,
  'createdBy': instance.createdBy,
  'updatedBy': instance.updatedBy,
  'deletedBy': instance.deletedBy,
  'createdAt': instance.createdAt?.toIso8601String(),
  'updatedAt': instance.updatedAt?.toIso8601String(),
  'deletedAt': instance.deletedAt?.toIso8601String(),
};

const _$WorkflowDefinitionDefinitionStatusEnumMap = {
  WorkflowDefinitionDefinitionStatus.draft: 'DRAFT',
  WorkflowDefinitionDefinitionStatus.enabled: 'ENABLED',
  WorkflowDefinitionDefinitionStatus.disabled: 'DISABLED',
  WorkflowDefinitionDefinitionStatus.$unknown: r'$unknown',
};
