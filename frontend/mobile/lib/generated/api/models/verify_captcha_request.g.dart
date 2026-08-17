// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'verify_captcha_request.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

VerifyCaptchaRequest _$VerifyCaptchaRequestFromJson(
  Map<String, dynamic> json,
) => VerifyCaptchaRequest(
  captchaId: json['captchaId'] as String?,
  userInput: json['userInput'] as String?,
);

Map<String, dynamic> _$VerifyCaptchaRequestToJson(
  VerifyCaptchaRequest instance,
) => <String, dynamic>{
  'captchaId': instance.captchaId,
  'userInput': instance.userInput,
};
