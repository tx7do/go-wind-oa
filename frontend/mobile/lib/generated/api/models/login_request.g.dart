// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'login_request.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

LoginRequest _$LoginRequestFromJson(Map<String, dynamic> json) => LoginRequest(
  clientId: json['client_id'] as String?,
  clientSecret: json['client_secret'] as String?,
  scope: json['scope'] as String?,
  redirectUri: json['redirect_uri'] as String?,
  userId: (json['user_id'] as num?)?.toInt(),
  username: json['username'] as String?,
  email: json['email'] as String?,
  mobile: json['mobile'] as String?,
  password: json['password'] as String?,
  refreshToken: json['refresh_token'] as String?,
  code: json['code'] as String?,
  clientType: json['client_type'] == null
      ? null
      : LoginRequestClientType.fromJson(json['client_type'] as String),
  deviceId: json['device_id'] as String?,
  jti: json['jti'] as String?,
  tenantCode: json['tenant_code'] as String?,
  grantType: json['grant_type'] == null
      ? LoginRequestGrantType.password
      : LoginRequestGrantType.fromJson(json['grant_type'] as String),
);

Map<String, dynamic> _$LoginRequestToJson(LoginRequest instance) =>
    <String, dynamic>{
      'grant_type': _$LoginRequestGrantTypeEnumMap[instance.grantType]!,
      'client_id': instance.clientId,
      'client_secret': instance.clientSecret,
      'scope': instance.scope,
      'redirect_uri': instance.redirectUri,
      'user_id': instance.userId,
      'username': instance.username,
      'email': instance.email,
      'mobile': instance.mobile,
      'password': instance.password,
      'refresh_token': instance.refreshToken,
      'code': instance.code,
      'client_type': _$LoginRequestClientTypeEnumMap[instance.clientType],
      'device_id': instance.deviceId,
      'jti': instance.jti,
      'tenant_code': instance.tenantCode,
    };

const _$LoginRequestGrantTypeEnumMap = {
  LoginRequestGrantType.password: 'password',
  LoginRequestGrantType.clientCredentials: 'client_credentials',
  LoginRequestGrantType.authorizationCode: 'authorization_code',
  LoginRequestGrantType.refreshToken: 'refresh_token',
  LoginRequestGrantType.implicit: 'implicit',
  LoginRequestGrantType.$unknown: r'$unknown',
};

const _$LoginRequestClientTypeEnumMap = {
  LoginRequestClientType.admin: 'admin',
  LoginRequestClientType.app: 'app',
  LoginRequestClientType.$unknown: r'$unknown',
};
