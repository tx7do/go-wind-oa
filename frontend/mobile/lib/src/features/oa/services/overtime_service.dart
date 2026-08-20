import 'package:dio/dio.dart' show DioException;
import 'package:get_it/get_it.dart' show GetIt;

import 'package:flutter_app/src/core/services/base_service.dart';

import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

/// OA 加班申请服务（app 端）。
///
/// 提交加班申请自动走 OVERTIME 工作流审批，终态仅同步单据状态，无额度副作用。
class OvertimeService extends BaseService {
  OvertimeService() : super(tag: 'OvertimeService');

  oaApi.OvertimeServiceClient get _api =>
      GetIt.instance<oaApi.ApiClient>().overtimeService;

  /// 提交加班申请（服务端创建 OVERTIME 工作流实例）。
  Future<dynamic> submit({
    required String reason,
    required DateTime startTime,
    required DateTime endTime,
    required oaApi.OaServiceV1OvertimeApplication$CompensationType compensationType,
  }) async {
    try {
      return await _api.submitOvertimeApplication(
          oaApi.OaServiceV1SubmitOvertimeApplicationRequest(
        reason: reason,
        startTime: startTime.toIso8601String(),
        endTime: endTime.toIso8601String(),
        compensationType: compensationType,
      ));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 我的加班申请列表。
  Future<dynamic> myApplications() async {
    try {
      return await _api.listOvertimeApplications(
          oaApi.OaServiceV1ListOvertimeApplicationsRequest());
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 加班申请详情。
  Future<dynamic> detail({required int id}) async {
    try {
      return await _api.getOvertimeApplication(
          oaApi.OaServiceV1GetOvertimeApplicationRequest(id: id));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }
}
