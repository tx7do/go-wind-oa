// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'generate_captcha_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

GenerateCaptchaResponse _$GenerateCaptchaResponseFromJson(
  Map<String, dynamic> json,
) => GenerateCaptchaResponse(
  captchaId: json['captchaId'] as String?,
  imageBase64: json['imageBase64'] as String?,
);

Map<String, dynamic> _$GenerateCaptchaResponseToJson(
  GenerateCaptchaResponse instance,
) => <String, dynamic>{
  'captchaId': instance.captchaId,
  'imageBase64': instance.imageBase64,
};
