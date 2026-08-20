import 'package:dio/dio.dart' show DioException;
import 'package:get_it/get_it.dart' show GetIt;

import 'package:flutter_app/src/core/services/base_service.dart';

// 生成代码由 buf.app.dart.gen.yaml 生成于
// generated/api/app/service/v1/index.dart。含 ApiClient.internalMessageService
// 及站内信消息类型（Internal_messageServiceV1* 前缀）。
import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

/// 站内信通知服务。
///
/// 经 [oaApi.ApiClient].internalMessageService.listMyMessages 拉取当前用户
/// 收件箱（收件人过滤在 core 侧按 viewer 强制，不含已删除/已撤销）。
/// 工作流审批/驳回/撤回等通知由 core 的 notifyManyAsync 落库。
class NotificationService extends BaseService {
  NotificationService() : super(tag: 'NotificationService');

  oaApi.InternalMessageServiceClient get _api =>
      GetIt.instance<oaApi.ApiClient>().internalMessageService;

  /// 拉取收件箱（直接调用，供页面消费）。失败返回空列表。
  Future<List<oaApi.Internal_messageServiceV1InternalMessage>> listMessages(
      {int limit = 50}) async {
    try {
      final resp = await _api.listMyMessages(
          oaApi.Internal_messageServiceV1ListMyMessagesRequest(limit: limit));
      return resp.items ??
          const <oaApi.Internal_messageServiceV1InternalMessage>[];
    } on DioException catch (e) {
      handleDioError(e);
      return const <oaApi.Internal_messageServiceV1InternalMessage>[];
    }
  }
}
