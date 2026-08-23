import 'package:dio/dio.dart' show DioException;
import 'package:get_it/get_it.dart' show GetIt;

import 'package:flutter_app/src/core/services/base_service.dart';

import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

/// 当前用户个人资料服务（app 端 /app/v1/me）。
///
/// 封装 GetUser（查看自身资料）、UpdateUser（修改自身资料，带字段掩码）、
/// ChangePassword（凭旧密码改新密码）三个端点。头像上传/联系人绑定在服务端
/// 尚未实现，此处不暴露。
class UserProfileService extends BaseService {
  UserProfileService() : super(tag: 'UserProfileService');

  /// 获取当前登录用户的资料。
  Future<dynamic> getUser() async {
    try {
      return await GetIt.instance<oaApi.ApiClient>()
          .userProfileService
          .getUser({});
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 修改当前登录用户的资料（仅传 [paths] 指定的字段）。
  ///
  /// [paths] 为字段掩码路径列表，后端据此更新对应字段。可更新字段以服务端
  /// allowlist 为准（昵称/头像/真实姓名/手机/邮箱/备注/性别）。
  Future<dynamic> updateUser(
    oaApi.IdentityServiceV1User data,
    List<String> paths,
  ) async {
    try {
      final request = oaApi.IdentityServiceV1UpdateUserRequest(
        data: data,
        updateMask: paths.join(','),
      );
      return await GetIt.instance<oaApi.ApiClient>()
          .userProfileService
          .updateUser(request);
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 修改当前登录用户的密码（需校验旧密码）。
  Future<dynamic> changePassword(String oldPassword, String newPassword) async {
    try {
      final request = oaApi.IdentityServiceV1ChangePasswordRequest(
        oldPassword: oldPassword,
        newPassword: newPassword,
      );
      return await GetIt.instance<oaApi.ApiClient>()
          .userProfileService
          .changePassword(request);
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }
}
