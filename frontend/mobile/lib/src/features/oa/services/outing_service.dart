import 'package:dio/dio.dart' show DioException;
import 'package:get_it/get_it.dart' show GetIt;

import 'package:flutter_app/src/core/services/base_service.dart';

import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

/// OA 外出申请服务（app 端）。
///
/// 提交外出申请自动走 OUTING 工作流审批，终态仅同步单据状态，无额度副作用。
class OutingService extends BaseService {
  OutingService() : super(tag: 'OutingService');

  oaApi.OutingServiceClient get _api =>
      GetIt.instance<oaApi.ApiClient>().outingService;

  /// 提交外出申请（服务端创建 OUTING 工作流实例）。
  Future<dynamic> submit({
    required String reason,
    required String destination,
    required DateTime startTime,
    required DateTime endTime,
  }) async {
    try {
      return await _api.submitOutingApplication(
          oaApi.OaServiceV1SubmitOutingApplicationRequest(
        reason: reason,
        destination: destination,
        startTime: startTime.toIso8601String(),
        endTime: endTime.toIso8601String(),
      ));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 我的外出申请列表。
  Future<dynamic> myApplications() async {
    try {
      return await _api.listOutingApplications(
          oaApi.OaServiceV1ListOutingApplicationsRequest());
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 外出申请详情。
  Future<dynamic> detail({required int id}) async {
    try {
      return await _api.getOutingApplication(
          oaApi.OaServiceV1GetOutingApplicationRequest(id: id));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }
}
