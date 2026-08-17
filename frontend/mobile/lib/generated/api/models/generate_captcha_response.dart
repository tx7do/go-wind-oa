// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'generate_captcha_response.g.dart';

/// 生成验证码 - 回应。与 cms GenerateCaptchaResponse 同构。
@JsonSerializable()
class GenerateCaptchaResponse {
  const GenerateCaptchaResponse({
    this.captchaId,
    this.imageBase64,
  });
  
  factory GenerateCaptchaResponse.fromJson(Map<String, Object?> json) => _$GenerateCaptchaResponseFromJson(json);
  
  /// 验证码ID，客户端应在后续请求中提供该ID以验证用户输入的验证码
  final String? captchaId;

  /// 验证码图片的Base64编码字符串，客户端可以解码并显示给用户
  final String? imageBase64;

  Map<String, Object?> toJson() => _$GenerateCaptchaResponseToJson(this);
}
