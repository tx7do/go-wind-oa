// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'submit_apply_response.g.dart';

@JsonSerializable()
class SubmitApplyResponse {
  const SubmitApplyResponse({
    this.instanceId,
  });
  
  factory SubmitApplyResponse.fromJson(Map<String, Object?> json) => _$SubmitApplyResponseFromJson(json);
  
  final int? instanceId;

  Map<String, Object?> toJson() => _$SubmitApplyResponseToJson(this);
}
