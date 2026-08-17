// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'submit_apply_request.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

SubmitApplyRequest _$SubmitApplyRequestFromJson(Map<String, dynamic> json) =>
    SubmitApplyRequest(
      definitionCode: json['definitionCode'] as String?,
      definitionVersion: (json['definitionVersion'] as num?)?.toInt(),
      title: json['title'] as String?,
      formData: json['formData'] as String?,
    );

Map<String, dynamic> _$SubmitApplyRequestToJson(SubmitApplyRequest instance) =>
    <String, dynamic>{
      'definitionCode': instance.definitionCode,
      'definitionVersion': instance.definitionVersion,
      'title': instance.title,
      'formData': instance.formData,
    };
