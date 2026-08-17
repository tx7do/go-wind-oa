// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'workflow_definition.dart';

part 'create_workflow_definition_request.g.dart';

@JsonSerializable()
class CreateWorkflowDefinitionRequest {
  const CreateWorkflowDefinitionRequest({
    this.data,
  });
  
  factory CreateWorkflowDefinitionRequest.fromJson(Map<String, Object?> json) => _$CreateWorkflowDefinitionRequestFromJson(json);
  
  final WorkflowDefinition? data;

  Map<String, Object?> toJson() => _$CreateWorkflowDefinitionRequestToJson(this);
}
