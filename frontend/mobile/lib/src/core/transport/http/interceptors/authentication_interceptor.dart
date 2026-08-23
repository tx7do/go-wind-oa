import 'dart:async';

import 'package:dio/dio.dart';

import 'package:flutter_app/src/core/constants/index.dart' show AppRoutePath;
import 'package:flutter_app/src/core/services/base_service.dart'
    show BaseService;
import 'package:flutter_app/src/core/utilities/logger.dart' show fatal;

/// 认证服务接口，定义了获取和刷新令牌的方法
abstract class AuthService extends BaseService {
  /// 获取当前访问令牌
  String? getAccessToken();

  /// 获取刷新令牌
  String? getRefreshToken();

  /// 刷新访问令牌
  /// 返回新的访问令牌，失败时返回null
  Future<String?> refreshToken();

  /// 认证失败处理
  authenticationFailed();
}

/// 认证拦截器
class AuthenticationInterceptor extends Interceptor {
  final AuthService Function() _authServiceFactory;
  final Dio _dio;
  final bool _autoRefreshToken;

  /// 单飞锁：非 null 表示一次刷新正在进行，后续 401 等其完成复用结果；
  /// null 表示无刷新在进行。初始 null。
  Completer<void>? _refreshLock;

  /// 创建认证拦截器实例
  /// [authServiceFactory] - 认证服务的懒工厂（推迟到首次请求时解析，
  ///   以规避 transport 初始化早于 repository 注册的顺序依赖）
  /// [dio] - 持有 baseUrl 与完整拦截器链的原始实例，刷新后重试经此发出
  /// [autoRefreshToken] - 是否自动刷新令牌，默认为true
  AuthenticationInterceptor({
    required AuthService Function() authServiceFactory,
    required Dio dio,
    bool autoRefreshToken = true,
  }) : _authServiceFactory = authServiceFactory,
       _dio = dio,
       _autoRefreshToken = autoRefreshToken;

  @override
  void onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    if (options.path != AppRoutePath.login) {
      final token = _authServiceFactory().getAccessToken();

      if (token != null && token.isNotEmpty) {
        options.headers['Authorization'] = _makeBearerToken(accessToken: token);
      }
    }

    return handler.next(options);
  }

  @override
  void onResponse(Response response, ResponseInterceptorHandler handler) async {
    return handler.next(response);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) async {
    if (err.response == null) {
      return;
    }

    // 如果不需要自动刷新令牌或不是401错误，则直接传递错误
    if (!_autoRefreshToken || err.response?.statusCode != 401) {
      return handler.next(err);
    }

    // 单次重试守卫：同一请求只允许刷新重试一次，避免 refresh 失败/401 循环。
    if (err.requestOptions.extra['__authRetried'] == true) {
      await _authServiceFactory().authenticationFailed();
      return handler.next(err);
    }

    try {
      // 尝试刷新令牌并重新发送请求
      final newToken = await _refreshToken();
      if (newToken == null) {
        // 刷新令牌失败，清除令牌并返回错误
        await _authServiceFactory().authenticationFailed();
        return handler.next(err);
      }

      // 使用新令牌经原始 dio 实例重试（继承 baseUrl 与完整拦截器链）；
      // onRequest 会再次以新 access token 填充 Authorization 头。
      final options = err.requestOptions;
      options.headers['Authorization'] = _makeBearerToken(
        accessToken: newToken,
      );
      options.extra['__authRetried'] = true;

      final response = await _dio.fetch(options);
      return handler.resolve(response);
    } catch (e) {
      fatal('Error refreshing token: $e');
      // 发生异常时清除令牌并传递错误
      await _authServiceFactory().authenticationFailed();
      return handler.next(err);
    }
  }

  /// 刷新令牌的方法，使用单飞锁机制防止并发刷新。
  ///
  /// _refreshLock 为 null 表示无刷新在进行；非 null 表示一次刷新正在进行，
  /// 后续 401 等待其完成后直接复用已更新的 access token。原实现用
  /// `late Completer _refreshLock = Completer()`（初始未完成），首个调用者
  /// 即落入等待分支，等待一个永不 complete 的 future——死锁，刷新链从未生效。
  Future<String?> _refreshToken() async {
    final auth = _authServiceFactory();

    // 已有刷新在进行，等待并复用结果
    final pending = _refreshLock;
    if (pending != null) {
      await pending.future;
      return auth.getAccessToken();
    }

    // 自己作为首个调用者，持锁执行刷新
    final lock = Completer<void>();
    _refreshLock = lock;
    try {
      final newToken = await auth.refreshToken();
      return newToken;
    } finally {
      lock.complete();
      _refreshLock = null;
    }
  }

  /// 创建Bearer令牌
  _makeBearerToken({String? accessToken}) {
    return "Bearer ${accessToken ?? ""}";
  }
}
