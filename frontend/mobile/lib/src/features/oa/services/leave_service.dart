import 'package:dio/dio.dart' show DioException;
import 'package:get_it/get_it.dart' show GetIt;

import 'package:flutter_app/src/core/services/base_service.dart';

import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

/// OA 请假服务（app 端）。
///
/// 提交请假自动走 LEAVE 工作流审批，通过后由服务端扣减假期额度。
class LeaveService extends BaseService {
  LeaveService() : super(tag: 'LeaveService');

  oaApi.LeaveServiceClient get _api =>
      GetIt.instance<oaApi.ApiClient>().leaveService;

  /// 请假类型列表（年假/病假/事假等）。
  Future<dynamic> leaveTypes() async {
    try {
      return await _api.listLeaveTypes(oaApi.PaginationPagingRequest());
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 我的假期额度（year 为空=当年）。
  Future<dynamic> myBalances({int? year}) async {
    try {
      return await _api.listLeaveBalances(
          oaApi.OaServiceV1ListLeaveBalancesRequest(year: year));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 提交请假申请（服务端校验额度并创建 LEAVE 工作流实例）。
  /// start_half=PM 表示首日只请下午；end_half=AM 表示末日只请上午。
  Future<dynamic> submit({
    required int leaveTypeId,
    required DateTime startDate,
    required DateTime endDate,
    required String reason,
    oaApi.OaServiceV1HalfOfDay startHalf = oaApi.OaServiceV1HalfOfDay.am,
    oaApi.OaServiceV1HalfOfDay endHalf = oaApi.OaServiceV1HalfOfDay.pm,
  }) async {
    try {
      return await _api.submitLeaveApplication(
          oaApi.OaServiceV1SubmitLeaveApplicationRequest(
        leaveTypeId: leaveTypeId,
        startDate: startDate.toIso8601String(),
        endDate: endDate.toIso8601String(),
        reason: reason,
        startHalf: startHalf,
        endHalf: endHalf,
      ));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 我的请假申请列表。
  Future<dynamic> myApplications() async {
    try {
      return await _api.listLeaveApplications(
          oaApi.OaServiceV1ListLeaveApplicationsRequest());
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 请假申请详情。
  Future<dynamic> detail({required int id}) async {
    try {
      return await _api
          .getLeaveApplication(oaApi.OaServiceV1GetLeaveApplicationRequest(id: id));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }
}
