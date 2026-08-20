import 'package:dio/dio.dart' show DioException;
import 'package:get_it/get_it.dart' show GetIt;

import 'package:flutter_app/src/core/services/base_service.dart';

import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

/// OA 费用报销服务（app 端）。
///
/// 提交报销自动走 EXPENSE 工作流审批；明细的发票凭证图先经文件上传接口
/// 取得 file_id 后填入 invoiceFileId。
class ExpenseService extends BaseService {
  ExpenseService() : super(tag: 'ExpenseService');

  oaApi.ExpenseServiceClient get _api =>
      GetIt.instance<oaApi.ApiClient>().expenseService;

  /// 提交报销申请（含多行明细）。
  Future<dynamic> submit({
    required String title,
    required List<oaApi.OaServiceV1ExpenseItem> items,
  }) async {
    try {
      return await _api.submitExpenseApplication(
          oaApi.OaServiceV1SubmitExpenseApplicationRequest(
        title: title,
        items: items,
      ));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 我的报销申请列表。
  Future<dynamic> myApplications() async {
    try {
      return await _api.listExpenseApplications(
          oaApi.OaServiceV1ListExpenseApplicationsRequest());
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 报销申请详情（含明细）。
  Future<dynamic> detail({required int id}) async {
    try {
      return await _api.getExpenseApplication(
          oaApi.OaServiceV1GetExpenseApplicationRequest(id: id));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }
}
