import 'package:dio/dio.dart' show DioException;
import 'package:get_it/get_it.dart' show GetIt;

import 'package:flutter_app/src/core/services/base_service.dart';

import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

/// OA 出差申请服务（app 端）。
///
/// 提交出差申请自动走 BUSINESS_TRIP 工作流审批，终态仅同步单据状态，无额度副作用。
class BusinessTripService extends BaseService {
  BusinessTripService() : super(tag: 'BusinessTripService');

  oaApi.BusinessTripServiceClient get _api =>
      GetIt.instance<oaApi.ApiClient>().businessTripService;

  /// 提交出差申请（服务端创建 BUSINESS_TRIP 工作流实例）。
  Future<dynamic> submit({
    required String title,
    required String destination,
    required DateTime startDate,
    required DateTime endDate,
    required String itinerary,
  }) async {
    try {
      return await _api.submitBusinessTripApplication(
          oaApi.OaServiceV1SubmitBusinessTripApplicationRequest(
        title: title,
        destination: destination,
        startDate: startDate.toIso8601String(),
        endDate: endDate.toIso8601String(),
        itinerary: itinerary,
      ));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 我的出差申请列表。
  Future<dynamic> myApplications() async {
    try {
      return await _api.listBusinessTripApplications(
          oaApi.OaServiceV1ListBusinessTripApplicationsRequest());
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 出差申请详情。
  Future<dynamic> detail({required int id}) async {
    try {
      return await _api.getBusinessTripApplication(
          oaApi.OaServiceV1GetBusinessTripApplicationRequest(id: id));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }
}
