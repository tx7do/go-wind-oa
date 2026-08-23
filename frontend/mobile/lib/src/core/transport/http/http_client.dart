import 'package:dio/dio.dart';
import 'package:get_it/get_it.dart' show GetIt;

import 'package:flutter_app/src/core/config/environments.dart';
import 'package:flutter_app/src/core/transport/http/interceptors/authentication_interceptor.dart';
import 'package:flutter_app/src/features/auth/services/authentication_service.dart';

/// 配置选项
void _configureOptions(Dio dio) {
  // debug('_configureOptions');

  dio.options.baseUrl = Environments.apiBaseUrl;
  dio.options.connectTimeout = Environments.connectionTimeout;
  dio.options.receiveTimeout = Environments.receiveTimeout;
  dio.options.responseType = ResponseType.json;
  dio.options.contentType = Headers.jsonContentType;
}

/// 注册默认拦截器
void _configureInterceptors(Dio dio) {
  // 认证拦截器：用懒工厂推迟 AuthenticationService 的实例化到首次请求时，
  // 以规避 transport 初始化（早）与 UserAuthCache 注册（晚）的顺序依赖。
  // 修复 AUD9-M4 相关的拦截器未注册 bug：此前该拦截器被整段注释，
  // 导致 Flutter 端所有需鉴权的 API 调用都不带 Authorization 头。
  //
  // autoRefreshToken=true 且注入 dio 实例：访问令牌过期（401）时，拦截器
  // 用 refresh token 换新 access token 并经【同一 dio 实例】重试——确保重试
  // 请求继承 baseUrl 与完整拦截器链。此前用 new Dio() 重试，新建实例无 baseUrl、
  // 无拦截器，重试必失败；且 autoRefreshToken=false 直接关闭了刷新链。
  dio.interceptors.add(AuthenticationInterceptor(
    authServiceFactory: () => AuthenticationService(),
    dio: dio,
    autoRefreshToken: true,
  ));
}

/// 注册拦截器
void registerInterceptor(Interceptor interceptor) {
  final dio = GetIt.instance<Dio>();
  dio.interceptors.add(interceptor);
}

Dio createDio() {
  final Dio dio = Dio();

  _configureOptions(dio);
  _configureInterceptors(dio);

  return dio;
}
