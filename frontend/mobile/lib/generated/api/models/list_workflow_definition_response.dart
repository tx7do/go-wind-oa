// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'workflow_definition.dart';

part 'list_workflow_definition_response.g.dart';

@JsonSerializable()
class ListWorkflowDefinitionResponse {
  const ListWorkflowDefinitionResponse({
    this.items,
    this.total,
  });
  
  factory ListWorkflowDefinitionResponse.fromJson(Map<String, Object?> json) => _$ListWorkflowDefinitionResponseFromJson(json);
  
  final List<WorkflowDefinition>? items;
  final String? total;

  Map<String, Object?> toJson() => _$ListWorkflowDefinitionResponseToJson(this);
}
