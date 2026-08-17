// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:dio/dio.dart';

import 'authentication_service/authentication_service_client.dart';
import 'workflow_service/workflow_service_client.dart';

///  `v0.0.1`
class RestClient {
  RestClient(
    Dio dio, {
    String? baseUrl,
  })  : _dio = dio,
        _baseUrl = baseUrl;

  final Dio _dio;
  final String? _baseUrl;

  static String get version => '0.0.1';

  AuthenticationServiceClient? _authenticationService;
  WorkflowServiceClient? _workflowService;

  AuthenticationServiceClient get authenticationService => _authenticationService ??= AuthenticationServiceClient(_dio, baseUrl: _baseUrl);

  WorkflowServiceClient get workflowService => _workflowService ??= WorkflowServiceClient(_dio, baseUrl: _baseUrl);
}
