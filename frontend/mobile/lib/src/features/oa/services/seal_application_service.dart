import 'package:dio/dio.dart' show DioException;
import 'package:get_it/get_it.dart' show GetIt;

import 'package:flutter_app/src/core/services/base_service.dart';

import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

/// OA 用印申请服务（app 端）。
///
/// 提交用印申请自动走 SEAL_APPLICATION 工作流审批，终态仅同步单据状态，无额度副作用。
class SealApplicationService extends BaseService {
  SealApplicationService() : super(tag: 'SealApplicationService');

  oaApi.SealApplicationServiceClient get _api =>
      GetIt.instance<oaApi.ApiClient>().sealApplicationService;

  /// 提交用印申请（服务端创建 SEAL_APPLICATION 工作流实例）。
  Future<dynamic> submit({
    required String purpose,
    required oaApi.OaServiceV1SealApplication$SealType sealType,
    required int fileCount,
    required String recipient,
  }) async {
    try {
      return await _api.submitSealApplication(
          oaApi.OaServiceV1SubmitSealApplicationRequest(
        purpose: purpose,
        sealType: sealType,
        fileCount: fileCount,
        recipient: recipient,
      ));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 我的用印申请列表。
  Future<dynamic> myApplications() async {
    try {
      return await _api.listSealApplications(
          oaApi.OaServiceV1ListSealApplicationsRequest());
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 用印申请详情。
  Future<dynamic> detail({required int id}) async {
    try {
      return await _api.getSealApplication(
          oaApi.OaServiceV1GetSealApplicationRequest(id: id));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }
}
