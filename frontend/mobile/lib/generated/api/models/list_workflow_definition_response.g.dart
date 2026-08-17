// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'list_workflow_definition_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ListWorkflowDefinitionResponse _$ListWorkflowDefinitionResponseFromJson(
  Map<String, dynamic> json,
) => ListWorkflowDefinitionResponse(
  items: (json['items'] as List<dynamic>?)
      ?.map((e) => WorkflowDefinition.fromJson(e as Map<String, dynamic>))
      .toList(),
  total: json['total'] as String?,
);

Map<String, dynamic> _$ListWorkflowDefinitionResponseToJson(
  ListWorkflowDefinitionResponse instance,
) => <String, dynamic>{'items': instance.items, 'total': instance.total};
