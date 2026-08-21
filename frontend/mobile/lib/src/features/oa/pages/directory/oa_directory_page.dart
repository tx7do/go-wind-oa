import 'package:flutter/material.dart';

import 'package:flutter_app/src/features/oa/services/directory_service.dart';
import 'package:flutter_app/src/core/transport/http/status.dart';
import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

/// 通讯录页。
///
/// 上半部：组织架构树（ExpansionTile 递归展开）。下半部：成员列表（全租户，
/// 标注部门/职位，敏感字段经后端脱敏）。
class OaDirectoryPage extends StatefulWidget {
  const OaDirectoryPage({super.key});

  @override
  State<OaDirectoryPage> createState() => _OaDirectoryPageState();
}

class _OaDirectoryPageState extends State<OaDirectoryPage> {
  final _service = DirectoryService();
  List<oaApi.IdentityServiceV1OrgUnit> _orgTree = const [];
  List<oaApi.IdentityServiceV1User> _members = const [];

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final treeResult = await _service.orgTree();
    final memberResult = await _service.members();
    if (!mounted) return;
    setState(() {
      _orgTree = (treeResult is Status)
          ? const []
          : (treeResult as oaApi.IdentityServiceV1ListOrgUnitResponse?)?.items ?? const [];
      _members = (memberResult is Status)
          ? const []
          : (memberResult as oaApi.IdentityServiceV1ListUserResponse?)?.items ?? const [];
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('通讯录')),
      body: ListView(
        padding: const EdgeInsets.all(12),
        children: [
          Text('组织架构', style: theme.textTheme.titleSmall),
          const SizedBox(height: 4),
          ..._orgTree.map((o) => _orgTile(o, 0)),
          const SizedBox(height: 16),
          Text('成员', style: theme.textTheme.titleSmall),
          const SizedBox(height: 4),
          if (_members.isEmpty)
            const Padding(padding: EdgeInsets.all(8), child: Text('暂无成员'))
          else
            ..._members.map((u) => ListTile(
                  dense: true,
                  leading: CircleAvatar(child: Text((u.nickname ?? '?').characters.first)),
                  title: Text(u.realname ?? u.nickname ?? ''),
                  subtitle: Text(_deptLabel(u), maxLines: 1, overflow: TextOverflow.ellipsis),
                )),
        ],
      ),
    );
  }

  Widget _orgTile(oaApi.IdentityServiceV1OrgUnit node, int depth) {
    final children = node.children ?? const [];
    if (children.isEmpty) {
      return Padding(
        padding: EdgeInsets.only(left: depth * 12.0 + 8, top: 4, bottom: 4),
        child: Text(node.name ?? '', style: const TextStyle(fontSize: 13)),
      );
    }
    return ExpansionTile(
      tilePadding: EdgeInsets.only(left: depth * 12.0),
      title: Text(node.name ?? '', style: const TextStyle(fontSize: 13)),
      children: children.map((c) => _orgTile(c, depth + 1)).toList(),
    );
  }

  String _deptLabel(oaApi.IdentityServiceV1User u) {
    final depts = u.orgUnitNames;
    final pos = u.positionNames;
    final parts = <String>[];
    if (depts != null && depts.isNotEmpty) parts.add(depts.join('/'));
    if (pos != null && pos.isNotEmpty) parts.add(pos.join('/'));
    return parts.join(' · ');
  }
}
