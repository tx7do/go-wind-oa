// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'create_workflow_definition_request.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

CreateWorkflowDefinitionRequest _$CreateWorkflowDefinitionRequestFromJson(
  Map<String, dynamic> json,
) => CreateWorkflowDefinitionRequest(
  data: json['data'] == null
      ? null
      : WorkflowDefinition.fromJson(json['data'] as Map<String, dynamic>),
);

Map<String, dynamic> _$CreateWorkflowDefinitionRequestToJson(
  CreateWorkflowDefinitionRequest instance,
) => <String, dynamic>{'data': instance.data};
