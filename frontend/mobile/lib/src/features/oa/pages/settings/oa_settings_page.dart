import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:flutter_app/generated/l10n.dart';
import 'package:flutter_app/src/core/themes/index.dart' as theme;
import 'package:flutter_app/src/core/widgets/app_back_button.dart';

/// 设置页（全屏，[AppBackButton] 返回）。
///
/// 仅含本地外观偏好，无服务端“系统配置”端点：
/// - 主题模式（浅色 / 深色 / 跟随系统）
/// - 主题色（种子色，预置色板）
/// - 语言（简体中文 / English）
///
/// 三者均经 [theme.AppThemeCubit] 即时生效，由 app.dart 处的
/// `context.watch<AppThemeCubit>()` 驱动 MaterialApp 重建。
class OaSettingsPage extends StatelessWidget {
  const OaSettingsPage({super.key});

  @override
  Widget build(BuildContext context) {
    final loc = S.of(context);
    final themeCubit = context.watch<theme.AppThemeCubit>();
    final currentMode = themeCubit.currentValue;
    final currentSeed = themeCubit.currentSeedColor;
    final currentLocale = themeCubit.currentLocale;

    return Scaffold(
      appBar: AppBar(leading: const AppBackButton(), title: Text(loc.settings)),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Text(loc.appearance,
              style: Theme.of(context).textTheme.titleSmall),
          const SizedBox(height: 8),
          // 主题模式
          SegmentedButton<ThemeMode>(
            segments: [
              ButtonSegment(
                value: ThemeMode.light,
                label: Text(loc.light),
                icon: const Icon(Icons.light_mode_outlined),
              ),
              ButtonSegment(
                value: ThemeMode.dark,
                label: Text(loc.dark),
                icon: const Icon(Icons.dark_mode_outlined),
              ),
              ButtonSegment(
                value: ThemeMode.system,
                label: Text(loc.followSystem),
                icon: const Icon(Icons.settings_brightness_outlined),
              ),
            ],
            selected: {currentMode},
            onSelectionChanged: (set) => themeCubit.modify(set.first),
          ),
          const SizedBox(height: 24),
          // 主题色
          Text(loc.themeColor, style: Theme.of(context).textTheme.titleSmall),
          const SizedBox(height: 8),
          Wrap(
            spacing: 12,
            children: _seedColors.map((c) {
              final selected = currentSeed.toARGB32() == c.toARGB32();
              return GestureDetector(
                onTap: () => themeCubit.modifySeedColor(c),
                child: Container(
                  width: 40,
                  height: 40,
                  decoration: BoxDecoration(
                    color: c,
                    shape: BoxShape.circle,
                    border: Border.all(
                      color: selected
                          ? Theme.of(context).colorScheme.onSurface
                          : Theme.of(context).dividerColor,
                      width: selected ? 3 : 1,
                    ),
                  ),
                ),
              );
            }).toList(),
          ),
          const SizedBox(height: 24),
          // 语言
          Text(loc.language, style: Theme.of(context).textTheme.titleSmall),
          const SizedBox(height: 8),
          DropdownButtonFormField<Locale>(
            value: currentLocale,
            decoration: const InputDecoration(border: OutlineInputBorder()),
            items: themeCubit.supportedLocales
                .map((l) => DropdownMenuItem(
                      value: l,
                      child: Text(_localeLabel(l)),
                    ))
                .toList(),
            onChanged: (l) {
              if (l != null) themeCubit.modifyLocale(l);
            },
          ),
        ],
      ),
    );
  }

  /// 预置主题色板。
  static final List<Color> _seedColors = [
    const Color(0xFF6750A4), // 默认紫
    const Color(0xFF0061A4), // 蓝
    const Color(0xFF006E1C), // 绿
    const Color(0xFF9C4146), // 红
    const Color(0xFF935F00), // 棕
    const Color(0xFF605D62), // 灰
  ];

  String _localeLabel(Locale l) {
    if (l.languageCode == 'zh') return '简体中文';
    return 'English';
  }
}
