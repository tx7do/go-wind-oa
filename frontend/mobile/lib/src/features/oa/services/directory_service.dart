import 'package:dio/dio.dart' show DioException;
import 'package:get_it/get_it.dart' show GetIt;

import 'package:flutter_app/src/core/services/base_service.dart';

import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

/// OA 通讯录服务（app 端，只读）。
///
/// 提供组织架构树与成员列表查询，供移动端通讯录浏览。敏感字段经后端脱敏。
class DirectoryService extends BaseService {
  DirectoryService() : super(tag: 'DirectoryService');

  /// 组织架构树（全租户）。
  Future<dynamic> orgTree() async {
    try {
      return await GetIt.instance<oaApi.ApiClient>().orgUnitService.list(oaApi.PaginationPagingRequest());
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 成员列表（全租户，带部门/职位标注）。
  Future<dynamic> members() async {
    try {
      return await GetIt.instance<oaApi.ApiClient>().userService.list(oaApi.PaginationPagingRequest());
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }
}
