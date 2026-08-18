import 'package:dio/dio.dart' show DioException;
import 'package:get_it/get_it.dart' show GetIt;

import 'package:flutter_app/src/core/services/base_service.dart';
import 'package:flutter_app/src/core/services/pagination_query.dart';

// 生成代码由 buf.flutter.oa.dart.gen.yaml 生成于
// generated/api/app/service/v1/index.dart。含 ApiClient.internalMessageService
// 及站内信消息类型（Internal_messageServiceV1* 前缀）。
import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

/// 站内信通知服务。
///
/// 经 [oaApi.ApiClient].internalMessageService 拉取当前用户收件箱的站内信列表。
/// 站内信由 core-service 的工作流审批通知（notifyAsync）落库，此服务仅读取展示。
class NotificationService extends BaseService {
  NotificationService() : super(tag: 'NotificationService');

  oaApi.InternalMessageServiceClient get _api =>
      GetIt.instance<oaApi.ApiClient>().internalMessageService;

  /// 拉取站内信列表（直接调用，供页面 FutureBuilder 消费）。
  Future<List<oaApi.Internal_messageServiceV1InternalMessage>> listMessages(
      [PaginationQuery? query]) async {
    final q = query ?? const PaginationQuery();
    try {
      final resp = await _api.listMessage(q.toPagingRequest());
      return resp.items ?? const <oaApi.Internal_messageServiceV1InternalMessage>[];
    } on DioException catch (e) {
      handleDioError(e);
      return const <oaApi.Internal_messageServiceV1InternalMessage>[];
    }
  }
}
