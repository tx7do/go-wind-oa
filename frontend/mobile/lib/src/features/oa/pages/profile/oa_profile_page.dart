import 'package:flutter/material.dart';

import 'package:flutter_app/generated/l10n.dart';
import 'package:flutter_app/src/core/transport/http/status.dart';
import 'package:flutter_app/src/core/widgets/app_back_button.dart';
import 'package:flutter_app/src/features/oa/services/user_profile_service.dart';
import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;
import 'package:flutter_app/src/core/utilities/date_time.dart';

/// 个人信息页（全屏，[AppBackButton] 返回）。
///
/// 上半部“账号信息”只读展示：用户名 / 租户 / 部门 / 职位 / 角色 / 注册时间 /
/// 最近登录。下半部“可编辑信息”表单：昵称 / 真实姓名 / 手机号 / 邮箱 / 性别 /
/// 备注。保存时仅提交相对服务端原值发生变化的字段（字段掩码）。“修改密码”
/// 按钮弹出对话框，凭旧密码改新密码。
class OaProfilePage extends StatefulWidget {
  const OaProfilePage({super.key});

  @override
  State<OaProfilePage> createState() => _OaProfilePageState();
}

class _OaProfilePageState extends State<OaProfilePage> {
  final _service = UserProfileService();
  oaApi.IdentityServiceV1User? _server;
  bool _loading = true;

  // 可编辑字段控制器
  late final TextEditingController _nicknameCtrl;
  late final TextEditingController _realnameCtrl;
  late final TextEditingController _mobileCtrl;
  late final TextEditingController _emailCtrl;
  late final TextEditingController _remarkCtrl;
  oaApi.IdentityServiceV1User$Gender? _gender;

  @override
  void initState() {
    super.initState();
    _nicknameCtrl = TextEditingController();
    _realnameCtrl = TextEditingController();
    _mobileCtrl = TextEditingController();
    _emailCtrl = TextEditingController();
    _remarkCtrl = TextEditingController();
    _load();
  }

  @override
  void dispose() {
    _nicknameCtrl.dispose();
    _realnameCtrl.dispose();
    _mobileCtrl.dispose();
    _emailCtrl.dispose();
    _remarkCtrl.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    final result = await _service.getUser();
    if (!mounted) return;
    if (result is Status) {
      setState(() => _loading = false);
      return;
    }
    final u = result as oaApi.IdentityServiceV1User?;
    _server = u;
    _nicknameCtrl.text = u?.nickname ?? '';
    _realnameCtrl.text = u?.realname ?? '';
    _mobileCtrl.text = u?.mobile ?? '';
    _emailCtrl.text = u?.email ?? '';
    _remarkCtrl.text = u?.remark ?? '';
    _gender = u?.gender;
    setState(() => _loading = false);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final loc = S.of(context);

    if (_loading) {
      return Scaffold(
        appBar: AppBar(leading: const AppBackButton(), title: Text(loc.profileTitle)),
        body: const Center(child: CircularProgressIndicator()),
      );
    }
    if (_server == null) {
      return Scaffold(
        appBar: AppBar(leading: const AppBackButton(), title: Text(loc.profileTitle)),
        body: Center(child: Text(loc.profileLoadFailed)),
      );
    }

    return Scaffold(
      appBar: AppBar(leading: const AppBackButton(), title: Text(loc.profileTitle)),
      body: ListView(
        padding: const EdgeInsets.all(12),
        children: [
          _sectionTitle(theme, loc.profileAccountInfo),
          _readOnlyField(theme, loc.profileUsername, _server!.username),
          _readOnlyField(theme, loc.profileTenant, _server!.tenantName),
          _readOnlyField(theme, loc.profileOrgUnit,
              (_server!.orgUnitNames ?? const []).join(' / ')),
          _readOnlyField(theme, loc.profilePosition,
              (_server!.positionNames ?? const []).join(' / ')),
          _readOnlyField(theme, loc.profileRoles,
              (_server!.roleNames ?? const []).join(' / ')),
          _readOnlyField(theme, loc.profileCreatedAt,
              DateTimeUtils.formatDateTime(_server!.createdAt)),
          _readOnlyField(theme, loc.profileLastLogin,
              DateTimeUtils.formatDateTime(_server!.lastLoginAt)),
          const SizedBox(height: 16),
          _sectionTitle(theme, loc.profileEditableInfo),
          TextField(
            controller: _nicknameCtrl,
            decoration: InputDecoration(labelText: loc.profileNickname),
          ),
          TextField(
            controller: _realnameCtrl,
            decoration: InputDecoration(labelText: loc.profileRealname),
          ),
          TextField(
            controller: _mobileCtrl,
            decoration: InputDecoration(labelText: loc.profileMobile),
            keyboardType: TextInputType.phone,
          ),
          TextField(
            controller: _emailCtrl,
            decoration: InputDecoration(labelText: loc.profileEmail),
            keyboardType: TextInputType.emailAddress,
          ),
          DropdownButtonFormField<oaApi.IdentityServiceV1User$Gender?>(
            value: _gender,
            decoration: InputDecoration(labelText: loc.profileGender),
            items: [
              DropdownMenuItem(
                value: oaApi.IdentityServiceV1User$Gender.male,
                child: Text(loc.genderMale),
              ),
              DropdownMenuItem(
                value: oaApi.IdentityServiceV1User$Gender.female,
                child: Text(loc.genderFemale),
              ),
              DropdownMenuItem(
                value: oaApi.IdentityServiceV1User$Gender.secret,
                child: Text(loc.genderSecret),
              ),
            ],
            onChanged: (v) => setState(() => _gender = v),
          ),
          TextField(
            controller: _remarkCtrl,
            decoration: InputDecoration(labelText: loc.profileRemark),
            maxLines: 3,
          ),
          const SizedBox(height: 16),
          FilledButton.icon(
            onPressed: _save,
            icon: const Icon(Icons.save_outlined),
            label: Text(loc.profileSave),
          ),
          const SizedBox(height: 8),
          OutlinedButton.icon(
            onPressed: _showChangePasswordDialog,
            icon: const Icon(Icons.lock_outline),
            label: Text(loc.profileChangePassword),
          ),
        ],
      ),
    );
  }

  Widget _sectionTitle(ThemeData theme, String text) {
    return Padding(
      padding: const EdgeInsets.only(top: 8, bottom: 8),
      child: Text(text, style: theme.textTheme.titleSmall),
    );
  }

  Widget _readOnlyField(ThemeData theme, String label, String? value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: InputDecorator(
        decoration: InputDecoration(
          labelText: label,
          isDense: true,
          enabledBorder: OutlineInputBorder(
            borderSide: BorderSide(color: theme.dividerColor),
          ),
          border: OutlineInputBorder(
            borderSide: BorderSide(color: theme.dividerColor),
          ),
        ),
        child: SelectableText(
          (value == null || value.isEmpty) ? '—' : value,
          style: theme.textTheme.bodyMedium,
        ),
      ),
    );
  }

  /// 保存：仅提交相对服务端原值变化的字段，构造字段掩码调用 updateUser。
  Future<void> _save() async {
    final loc = S.of(context);
    final paths = <String>[];
    final draft = oaApi.IdentityServiceV1User();
    if (_nicknameCtrl.text != (_server!.nickname ?? '')) {
      draft.nickname = _nicknameCtrl.text;
      paths.add('nickname');
    }
    if (_realnameCtrl.text != (_server!.realname ?? '')) {
      draft.realname = _realnameCtrl.text;
      paths.add('realname');
    }
    if (_mobileCtrl.text != (_server!.mobile ?? '')) {
      draft.mobile = _mobileCtrl.text;
      paths.add('mobile');
    }
    if (_emailCtrl.text != (_server!.email ?? '')) {
      draft.email = _emailCtrl.text;
      paths.add('email');
    }
    if (_remarkCtrl.text != (_server!.remark ?? '')) {
      draft.remark = _remarkCtrl.text;
      paths.add('remark');
    }
    if (_gender != _server!.gender) {
      draft.gender = _gender;
      paths.add('gender');
    }
    if (paths.isEmpty) {
      _toast(loc.profileSaveNothing);
      return;
    }
    final result = await _service.updateUser(draft, paths);
    final ok = result is! Status;
    if (!mounted) return;
    _toast(ok ? loc.profileSaveSuccess : loc.profileSaveFailed);
    if (ok) {
      // 以服务端返回为准刷新本地副本，避免再次拉取。
      _server = _server!.copyWith(
        nickname: draft.nickname,
        realname: draft.realname,
        mobile: draft.mobile,
        email: draft.email,
        remark: draft.remark,
        gender: draft.gender,
      );
    }
  }

  void _toast(String msg) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
  }

  /// 修改密码对话框：旧密码 + 新密码 + 确认新密码，前端基础校验后调用
  /// [UserProfileService.changePassword]。
  Future<void> _showChangePasswordDialog() async {
    final loc = S.of(context);
    final oldCtrl = TextEditingController();
    final newCtrl = TextEditingController();
    final confirmCtrl = TextEditingController();

    await showDialog<void>(
      context: context,
      builder: (ctx) {
        return StatefulBuilder(
          builder: (ctx, setState) {
            return AlertDialog(
              title: Text(loc.changePasswordTitle),
              content: SingleChildScrollView(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    TextField(
                      controller: oldCtrl,
                      obscureText: true,
                      decoration: InputDecoration(labelText: loc.profileOldPassword),
                    ),
                    TextField(
                      controller: newCtrl,
                      obscureText: true,
                      decoration: InputDecoration(labelText: loc.profileNewPassword),
                    ),
                    TextField(
                      controller: confirmCtrl,
                      obscureText: true,
                      decoration: InputDecoration(labelText: loc.profileConfirmPassword),
                    ),
                  ],
                ),
              ),
              actions: [
                TextButton(
                  onPressed: () => Navigator.of(ctx).pop(),
                  child: Text(loc.cancel),
                ),
                TextButton(
                  onPressed: () async {
                    if (newCtrl.text != confirmCtrl.text) {
                      ScaffoldMessenger.of(context).showSnackBar(
                        SnackBar(content: Text(loc.profilePasswordMismatch)),
                      );
                      return;
                    }
                    if (newCtrl.text.length < 6) {
                      ScaffoldMessenger.of(context).showSnackBar(
                        SnackBar(content: Text(loc.profilePasswordTooShort)),
                      );
                      return;
                    }
                    Navigator.of(ctx).pop();
                    final res =
                        await _service.changePassword(oldCtrl.text, newCtrl.text);
                    if (!mounted) return;
                    ScaffoldMessenger.of(context).showSnackBar(SnackBar(
                      content: Text(res is Status
                          ? loc.profilePasswordChangeFailed
                          : loc.profilePasswordChanged),
                    ));
                  },
                  child: Text(loc.confirm),
                ),
              ],
            );
          },
        );
      },
    );
    oldCtrl.dispose();
    newCtrl.dispose();
    confirmCtrl.dispose();
  }
}
