// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'submit_apply_request.g.dart';

@JsonSerializable()
class SubmitApplyRequest {
  const SubmitApplyRequest({
    this.definitionCode,
    this.definitionVersion,
    this.title,
    this.formData,
  });
  
  factory SubmitApplyRequest.fromJson(Map<String, Object?> json) => _$SubmitApplyRequestFromJson(json);
  
  final String? definitionCode;
  final int? definitionVersion;
  final String? title;

  /// 申请表单数据，原始 JSON 文本。
  final String? formData;

  Map<String, Object?> toJson() => _$SubmitApplyRequestToJson(this);
}
