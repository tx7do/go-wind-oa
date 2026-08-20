import 'package:flutter/services.dart' show PlatformException, MissingPluginException;
import 'package:get_it/get_it.dart' show GetIt;
import 'package:dio/dio.dart' show DioException;

import 'package:flutter_app/src/core/services/base_service.dart';

// 生成代码由 buf.app.dart.gen.yaml 生成于
// generated/api/app/service/v1/index.dart。含 ApiClient.attendanceService
// 及打卡请求/响应类型（OaServiceV1CheckInRequest / OaServiceV1AttendanceRecord）。
import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

import 'package:geolocator/geolocator.dart';
import 'package:network_info_plus/network_info_plus.dart';

/// 考勤打卡服务。
///
/// 流程：请求精确定位权限 → 取当前 GPS 坐标 → 尽力采集当前 Wi-Fi BSSID
/// （不可用则为 null）→ 调 [oaApi.ApiClient].attendanceService.checkIn 提交到
/// app-service 的 HTTP 边端，由其转发到 core-service 落打卡记录：
/// 当日首次打卡=上班签到，第二次=下班签退并结算当日结果（迟到/早退/正常/请假）。
///
/// 服务端返回完整 [oaApi.OaServiceV1AttendanceRecord]；前置失败（权限/定位）
/// 返回 [AttendancePreflight] 枚举，页面据此提示。
enum AttendancePreflight {
  /// 定位权限被拒。
  permissionDenied,

  /// 无法获取当前位置（GPS 关闭 / 超时等）。
  locationUnavailable,
}

class AttendanceService extends BaseService {
  AttendanceService() : super(tag: 'AttendanceService');

  oaApi.AttendanceServiceClient get _api =>
      GetIt.instance<oaApi.ApiClient>().attendanceService;

  /// 执行打卡。成功返回 [oaApi.OaServiceV1AttendanceRecord]（含签到/签退时间
  /// 与结算结果），失败返回 [AttendancePreflight] 或 [Status]（网络/服务端错误）。
  Future<dynamic> checkIn() async {
    // 1. 定位权限与可用性校验。
    final perm = await Geolocator.checkPermission();
    if (perm == LocationPermission.denied) {
      final req = await Geolocator.requestPermission();
      if (req == LocationPermission.denied ||
          req == LocationPermission.deniedForever) {
        return AttendancePreflight.permissionDenied;
      }
    }
    if (perm == LocationPermission.deniedForever) {
      return AttendancePreflight.permissionDenied;
    }

    final serviceEnabled = await Geolocator.isLocationServiceEnabled();
    if (!serviceEnabled) {
      return AttendancePreflight.locationUnavailable;
    }

    // 2. 取当前 GPS 坐标。任一异常均归为 locationUnavailable。
    final Position pos;
    try {
      pos = await Geolocator.getCurrentPosition(
        locationSettings: const LocationSettings(
          accuracy: LocationAccuracy.high,
          timeLimit: Duration(seconds: 10),
        ),
      );
    } on PlatformException {
      return AttendancePreflight.locationUnavailable;
    } on MissingPluginException {
      return AttendancePreflight.locationUnavailable;
    } catch (_) {
      return AttendancePreflight.locationUnavailable;
    }

    // 3. 尽力采集当前 Wi-Fi BSSID（与 GPS 一同落打卡记录，供审计追溯）。
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

    // 4. 提交打卡。
    try {
      return await _api.checkIn(oaApi.OaServiceV1CheckInRequest(
        latitude: pos.latitude,
        longitude: pos.longitude,
        wifiBssid: bssid,
      ));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }

  /// 我的打卡记录（默认近 30 天）。
  Future<dynamic> myRecords({DateTime? startDate, DateTime? endDate}) async {
    try {
      return await _api.getMyAttendanceRecords(
          oaApi.OaServiceV1GetMyAttendanceRecordsRequest(
        startDate: startDate?.toIso8601String(),
        endDate: endDate?.toIso8601String(),
      ));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }
}
