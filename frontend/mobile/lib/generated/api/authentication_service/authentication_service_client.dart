// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:dio/dio.dart';
import 'package:retrofit/retrofit.dart';

import '../models/generate_captcha_response.dart';
import '../models/login_request.dart';
import '../models/login_response.dart';
import '../models/verify_captcha_request.dart';
import '../models/verify_captcha_response.dart';

part 'authentication_service_client.g.dart';

@RestApi()
abstract class AuthenticationServiceClient {
  factory AuthenticationServiceClient(Dio dio, {String? baseUrl}) = _AuthenticationServiceClient;

  /// 生成验证码。转发至 cms /admin/v1/captcha。
  @GET('/admin/v1/captcha')
  Future<GenerateCaptchaResponse> authenticationServiceGenerateCaptcha();

  /// 验证验证码。转发至 cms /admin/v1/captcha/verify。
  @POST('/admin/v1/captcha/verify')
  Future<VerifyCaptchaResponse> authenticationServiceVerifyCaptcha({
    @Body() required VerifyCaptchaRequest body,
  });

  /// 登录。转发至 cms /admin/v1/login。
  @POST('/admin/v1/login')
  Future<LoginResponse> authenticationServiceLogin({
    @Body() required LoginRequest body,
  });

  /// 登出。转发至 cms /admin/v1/logout。
  @POST('/admin/v1/logout')
  Future<void> authenticationServiceLogout();

  /// 刷新认证令牌。转发至 cms /admin/v1/refresh-token。
  @POST('/admin/v1/refresh-token')
  Future<LoginResponse> authenticationServiceRefreshToken({
    @Body() required LoginRequest body,
  });
}
