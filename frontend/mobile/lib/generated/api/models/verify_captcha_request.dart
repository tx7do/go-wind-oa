// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'verify_captcha_request.g.dart';

/// 验证验证码 - 请求。与 cms VerifyCaptchaRequest 同构。
@JsonSerializable()
class VerifyCaptchaRequest {
  const VerifyCaptchaRequest({
    this.captchaId,
    this.userInput,
  });
  
  factory VerifyCaptchaRequest.fromJson(Map<String, Object?> json) => _$VerifyCaptchaRequestFromJson(json);
  
  /// 验证码ID，来自 GenerateCaptcha 响应
  final String? captchaId;

  /// 用户输入的验证码文本
  final String? userInput;

  Map<String, Object?> toJson() => _$VerifyCaptchaRequestToJson(this);
}
