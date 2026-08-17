// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'login_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

LoginResponse _$LoginResponseFromJson(Map<String, dynamic> json) =>
    LoginResponse(
      accessToken: json['access_token'] as String?,
      expiresIn: json['expires_in'] as String?,
      refreshToken: json['refresh_token'] as String?,
      scope: json['scope'] as String?,
      refreshExpiresIn: json['refresh_expires_in'] as String?,
      idToken: json['id_token'] as String?,
      tokenType: json['token_type'] == null
          ? LoginResponseTokenType.bearer
          : LoginResponseTokenType.fromJson(json['token_type'] as String),
    );

Map<String, dynamic> _$LoginResponseToJson(LoginResponse instance) =>
    <String, dynamic>{
      'token_type': _$LoginResponseTokenTypeEnumMap[instance.tokenType]!,
      'access_token': instance.accessToken,
      'expires_in': instance.expiresIn,
      'refresh_token': instance.refreshToken,
      'scope': instance.scope,
      'refresh_expires_in': instance.refreshExpiresIn,
      'id_token': instance.idToken,
    };

const _$LoginResponseTokenTypeEnumMap = {
  LoginResponseTokenType.bearer: 'bearer',
  LoginResponseTokenType.mac: 'mac',
  LoginResponseTokenType.$unknown: r'$unknown',
};
