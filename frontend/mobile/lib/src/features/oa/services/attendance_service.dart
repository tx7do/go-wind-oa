import 'package:flutter/services.dart' show PlatformException, MissingPluginException;
import 'package:get_it/get_it.dart' show GetIt;
import 'package:dio/dio.dart' show DioException;

import 'package:flutter_app/src/core/services/base_service.dart';

// 生成代码由 buf.flutter.oa.dart.gen.yaml 生成于
// generated/api/app/service/v1/index.dart。含 ApiClient.attendanceService
// 及打卡请求/响应类型（OaServiceV1CheckInRequest / Response，枚举
// OaServiceV1CheckInResponse$CheckResult）。
import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

import 'package:geolocator/geolocator.dart';
import 'package:network_info_plus/network_info_plus.dart';

/// 打卡结果。
enum CheckInResult {
  /// 服务端判定位于公司地理围栏内。
  inFence,

  /// 服务端判定匹配公司 Wi-Fi 白名单。
  inWifi,

  /// 服务端拒绝：不在围栏也不在 Wi-Fi 白名单。
  denied,

  /// 定位权限被拒。
  permissionDenied,

  /// 无法获取当前位置（GPS 关闭 / 超时等）。
  locationUnavailable,

  /// 网络或服务端错误（状态已由 handleDioError 处理）。
  error,
}

/// 考勤打卡服务。
///
/// 流程：请求精确定位权限 → 取当前 GPS 坐标 → 尽力采集当前 Wi-Fi BSSID
/// （不可用则为 null）→ 调 [oaApi.ApiClient].attendanceService.checkIn 提交到
/// app-service 的 HTTP 边端，由其转发到 core-service 做围栏 / Wi-Fi 白名单
/// 判定并落打卡记录。返回服务端判定结果。
///
/// 权限拒绝、定位不可用等前置失败原因与服务端判定同以 [CheckInResult]
/// 返回，便于页面按结果展示。
class AttendanceService extends BaseService {
  AttendanceService() : super(tag: 'AttendanceService');

  oaApi.AttendanceServiceClient get _api =>
      GetIt.instance<oaApi.ApiClient>().attendanceService;

  /// 执行打卡。返回服务端判定或前置失败原因。
  Future<CheckInResult> checkIn() async {
    // 1. 定位权限与可用性校验。
    final perm = await Geolocator.checkPermission();
    if (perm == LocationPermission.denied) {
      final req = await Geolocator.requestPermission();
      if (req == LocationPermission.denied ||
          req == LocationPermission.deniedForever) {
        return CheckInResult.permissionDenied;
      }
    }
    if (perm == LocationPermission.deniedForever) {
      return CheckInResult.permissionDenied;
    }

    final serviceEnabled = await Geolocator.isLocationServiceEnabled();
    if (!serviceEnabled) {
      return CheckInResult.locationUnavailable;
    }

    // 2. 取当前 GPS 坐标。任一异常（超时、平台拒绝、定位不可用）均归为
    //    locationUnavailable——前端无需区分具体原因。
    final Position pos;
    try {
      pos = await Geolocator.getCurrentPosition(
        locationSettings: const LocationSettings(
          accuracy: LocationAccuracy.high,
          timeLimit: Duration(seconds: 10),
        ),
      );
    } on PlatformException {
      return CheckInResult.locationUnavailable;
    } on MissingPluginException {
      return CheckInResult.locationUnavailable;
    } catch (_) {
      return CheckInResult.locationUnavailable;
    }

    // 3. 尽力采集当前 Wi-Fi BSSID。iOS/桌面平台或无连接时为 null，
    //    不影响提交——服务端会仅做围栏判定。
    String? bssid;
    try {
      bssid = await NetworkInfo().getWifiBSSID();
    } on PlatformException {
      bssid = null;
    } on MissingPluginException {
      bssid = null;
    } catch (_) {
      bssid = null;
    }

    // 4. 构造请求并提交。
    final req = oaApi.OaServiceV1CheckInRequest(
      longitude: pos.longitude,
      latitude: pos.latitude,
      bssid: bssid,
    );

    final oaApi.OaServiceV1CheckInResponse resp;
    try {
      resp = await _api.checkIn(req);
    } on DioException catch (e) {
      handleDioError(e);
      return CheckInResult.error;
    }

    // 5. 映射服务端判定。checkResult 可空——若服务端未填，按 denied 处理。
    final result = resp.checkResult;
    if (result == null) {
      return CheckInResult.denied;
    }
    switch (result) {
      case oaApi.OaServiceV1CheckInResponse$CheckResult.inFence:
        return CheckInResult.inFence;
      case oaApi.OaServiceV1CheckInResponse$CheckResult.inWifi:
        return CheckInResult.inWifi;
      case oaApi.OaServiceV1CheckInResponse$CheckResult.denied:
        return CheckInResult.denied;
    }
  }
}
