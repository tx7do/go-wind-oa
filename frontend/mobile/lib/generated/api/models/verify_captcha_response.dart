// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'verify_captcha_response.g.dart';

/// 验证验证码 - 回应。与 cms VerifyCaptchaResponse 同构。
@JsonSerializable()
class VerifyCaptchaResponse {
  const VerifyCaptchaResponse({
    this.valid,
  });
  
  factory VerifyCaptchaResponse.fromJson(Map<String, Object?> json) => _$VerifyCaptchaResponseFromJson(json);
  
  /// 验证码验证结果，true表示验证成功，false表示验证失败
  final bool? valid;

  Map<String, Object?> toJson() => _$VerifyCaptchaResponseToJson(this);
}
