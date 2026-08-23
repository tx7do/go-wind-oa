# go-wind-oa 移动端（Flutter）架构技术解析与二次开发导引

> 本文面向希望基于此项目进行二次开发的 Flutter 工程师，从技术栈选型、核心架构设计、关键模块实现到二开实践路径，提供一份完整的技术地图。
>
> 本项目为协同办公系统（OA）的移动端。后端架构见仓库 `docs/oa-workflow-design.md`，移动端架构导览见 `docs/oa-mobile-design.md`。

---

## 一、技术栈总览

本项目是一个 **Flutter** 多平台应用，一套 Dart 代码编译为 iOS、Android（兼可编 Web/macOS/Windows/Linux，实际部署目标是移动端）：

| 层面       | 技术                       | 用途                      |
|----------|--------------------------|-------------------------|
| 框架       | Flutter                  | 多平台 UI 框架               |
| 语言       | Dart                     | 类型安全 + 空安全              |
| 状态管理     | flutter_bloc + Cubit     | 响应式状态（Cubit 模式）         |
| 服务定位     | GetIt                    | 轻量 IoC 容器（单例管理）         |
| 路由       | GoRouter                 | 声明式路由 + ShellRoute      |
| 国际化      | flutter_intl (intl)      | ARB 翻译文件 + 代码生成         |
| HTTP 客户端 | Dio                      | REST 通信                 |
| API 代码生成 | protoc-gen-dart-http (buf) | proto wrapper → Dart 客户端 |
| 数据缓存     | cached_query             | Query/Mutation 缓存管理     |
| 响应式适配    | flutter_screenutil       | 手机端设计稿适配                |
| 主题       | Material 3 + ColorScheme.fromSeed | 动态主题色          |
| 加密       | encrypt + crypto         | AES 加密（Token 持久化）       |
| 图片缓存     | cached_network_image     | 网络图片缓存                  |
| 环境变量     | flutter_dotenv           | .env 文件管理               |
| 日志       | logger                   | 结构化日志                   |

**代码生成工具链：** `protoc-gen-dart-http`（API 客户端，经 `buf.app.dart.gen.yaml`） + `intl_utils`（国际化）。

> `pubspec.yaml` 中 `swagger_parser` / `retrofit` / `retrofit_generator` / `freezed` / `json_serializable` / `build_runner` 等依赖无对应引用，可清理。

---

## 二、核心架构设计

### 2.1 多平台编译模式

项目基于 Flutter，通过 `flutter` CLI 将同一份 Dart 代码编译为多端产物（实际部署目标为移动端）：

```bash
flutter run -d ios / android
flutter build apk / ios
```

**环境配置**（`.dev.env` Debug / `.env` Release）：

```env
NTP_HOST="time.google.com"
API_BASE_URL="http://localhost:6700"        # app-service
SSE_URL="http://localhost:6701/events"      # SSE
CONNECTION_TIMEOUT=3000
RECEIVE_TIMEOUT=3000
AES_KEY="f51d66a73d8a0927"
SENTRY_DSN="https://ingest.sentry.io/"
```

> 环境变量通过 `flutter_dotenv` 加载，在 `Environments` 类（`core/config/environments.dart`）中统一导出。Debug 模式加载 `.dev.env`，Release 模式加载 `.env`。

### 2.2 路由架构

使用 **GoRouter** 实现声明式路由。路由定义在 `lib/src/app_router/app_router.dart`，路径常量在 `lib/src/core/constants/router_paths.dart`。

- **ShellRoute**（`OaShellPage`，`pages/shell/oa_shell_page.dart`）：底部导航四 Tab——`/oa/tasks`（审批）/ `/oa/notifications`（通知）/ `/oa/attendance`（考勤）/ `/oa/me`（个人中心）。`OaShellPage` 为 `StatelessWidget`，据 `currentRoute` 高亮 Tab，不持有业务状态。`/oa/tasks` 下挂子路由 `detail/:id`（任务详情）。
- **非 Shell 全屏路由**：`/oa/apply`（通用申请）、`/oa/leave`、`/oa/expense`、`/oa/business-trip`、`/oa/overtime`、`/oa/seal-application`、`/oa/outing`、`/oa/directory`（通讯录）、`/oa/profile`（资料编辑）、`/oa/settings`（设置）、`/login`。
- **守卫 `_guard`**：依 `GetIt.instance<UserAuthCache>().hasLogin` 判定；未登录访问任意非登录路由 → `/login`；已登录访问 `/login` 或 `/` → `/oa/tasks`。

> `router_paths.dart` 中 `signIn`/`signUp` 常量已定义但未接任何 `GoRoute`（死常量）。路由路径与名称常量集中管理，不要硬编码。

**路由目录结构**：

```
lib/src/app_router/
├── app_router.dart          # GoRouter 配置（ShellRoute + 非_shell 路由定义）
└── route_names.dart         # 路由名称常量
```

### 2.3 三层 API 架构

API 层遵循**生成层 → 服务层 → 调用层**的三层分离架构：

```
lib/generated/api/                     # [自动生成] protoc-gen-dart-http via buf 产出
├── transport.dart                     #   ClientTransport 抽象
└── app/service/v1/index.dart          #   ApiClient（13 个 ServiceClient getter）+ 全部类型

lib/src/features/{auth,oa}/services/   # [服务封装] 业务逻辑、错误处理
```

**第一层 — 自动生成的客户端**（`generated/api/`）：由 `backend/api/buf.app.dart.gen.yaml` 模板经 `protoc-gen-dart-http` 从 `protos/app/service/v1/` 下 wrapper proto 生成，不应手动编辑。`index.dart` 内 `ApiClient` 暴露 13 个 `*ServiceClient` 懒加载 getter，类型名带包前缀（`OaServiceV1*` / `IdentityServiceV1*` / `AuthenticationServiceV1*` / `Internal_messageServiceV1*` / `StorageServiceV1*`），枚举成员为小写。

**第二层 — 服务封装**（`features/{auth,oa}/services/`）：基于 `ApiClient` 封装业务方法，统一错误处理（`handleDioError`）。示例：

```dart
class WorkflowService extends BaseService {
  WorkflowService() : super(tag: 'WorkflowService');

  oaApi.WorkflowServiceClient get _api =>
      GetIt.instance<oaApi.ApiClient>().workflowService;

  Future<dynamic> pendingTasks(int page, int pageSize) async {
    try {
      return await _api.getMyTasks(oaApi.PaginationPagingRequest(
        page: page, pageSize: pageSize, /* ... */
      ));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }
}
```

**第三层 — 页面调用**：页面组件直接实例化 Service 并调用方法，不额外封装 Hook。

> 生成层路径为 `app/service/v1`（非 `oa/v1`）。

### 2.4 Dio — HTTP 通信内核

`lib/src/core/transport/http/http_client.dart` 的 `createDio()` 构建全局单例 Dio（经 `transport/init.dart` 注册到 GetIt），`_configureOptions` 从 `Environments` 读取 `baseUrl`/超时，`_configureInterceptors` 仅注册 `AuthenticationInterceptor`。`lib/src/core/transport/http/dio_client_transport.dart` 的 `DioClientTransport` 实现 buf 生成的 `ClientTransport` 接口，`unary` 转发到 `_dio.request`（`serverStream`/`duplexStream` 抛 `UnimplementedError`），把生成的 `ApiClient` 调用桥接到 Dio。

**BaseService 错误处理**：

```dart
// core/services/base_service.dart
abstract class BaseService {
  Status handleDioError(DioException e) {
    final data = e.response?.data;
    if (data is Map<String, dynamic>) {
      return Status(code: ..., reason: ..., message: ..., metadata: ...);
    }
    return Status(code: ..., reason: e.type.name, message: e.message);
  }
}
```

二开时如需对接不同后端，只需修改 `.env` 中的 `API_BASE_URL` 并重跑 buf 生成。

### 2.5 状态管理 — BLoC / Cubit 模式

采用 `flutter_bloc` 的 Cubit 模式，用于主题、语言等全局状态：

```dart
// main.dart
void main() async {
  await init();
  runApp(
    MultiBlocProvider(
      providers: [BlocProvider(create: (_) => AppThemeCubit())],
      child: const OAApp(),
    ),
  );
}
```

`OAApp`（`app.dart`）经 `context.watch` 读取 `AppThemeCubit` 的 `theme`/`darkTheme`/`themeMode`/`locale`。页面内局部状态（列表数据、加载状态等）使用 `StatefulWidget` 的 `setState` 管理，不引入额外状态管理库。

### 2.6 偏好持久化 — UserPreferenceCache

`UserPreferenceCache`（`core/preference/user_preference_cache.dart`）基于 `SharedPreferences` 封装，管理用户偏好设置（主题模式、主题色、语言）。

主题支持三种模式：`light`（亮色）、`dark`（暗色）、`system`（跟随系统），通过 Material 3 的 `ColorScheme.fromSeed` 动态生成色板。

### 2.7 国际化体系

使用 **flutter_intl** (intl + ARB 文件) 实现国际化：

```
lib/l10n/
├── intl_zh_CN.arb       # 中文翻译
└── intl_en_US.arb       # 英文翻译

lib/generated/           # [自动生成] intl_utils 产出
├── l10n.dart            # S 类（统一导出所有翻译方法）
└── intl/
    ├── messages_zh_CN.dart
    └── messages_en_US.dart
```

**代码生成命令**：

```bash
flutter pub run intl_utils:generate
```

**使用方式**：

```dart
Text(S.of(context).appName)           // 获取当前语言的翻译
```

---

## 三、关键模块深度解析

### 3.1 响应式布局系统

项目随基座保留响应式设施，采用断点 + 双视图模式：

| 设备类型      | 屏幕宽度        | 布局策略               |
|-----------|-------------|--------------------|
| 手机 Mobile | < 600 dp    | 纵向单栏 + 底部导航栏    |
| 平板 Tablet | 600~1024 dp | 双栏布局               |
| 网页 Web    | > 1024 dp   | 三栏/居中布局 |

**核心组件**（`core/widgets/responsive_layout.dart`、`core/utils/responsive_utils.dart`、`core/constants/breakpoints.dart`）：

```dart
ResponsiveLayout(
  mobileBody: _buildMobileView(),
  webBody: _buildWebView(),
)

ResponsiveUtils.isMobile(context)     // 是否手机
```

> 实际 OA 业务页（移动端目标）基本未消费 `ResponsiveLayout`/`WebContentCenter`——响应式设施是基座遗产，移动端 UI 直接用 ScreenUtil。详见 [ui-adaptation.md](./ui-adaptation.md)。

**ScreenUtil 策略**：
- 手机端：使用 `.w` / `.h` / `.sp` 适配不同手机分辨率
- Web 端：`app.dart` 的 `ScreenUtilInit` builder 内 `kIsWeb` 分支将 `designSize` 动态设为当前视窗尺寸，使 `.w/.h/.sp` 始终 1:1，字体不随窗口缩放

### 3.2 主题系统

采用 **Material 3 + ColorScheme.fromSeed** 方案，支持动态主题色切换：

```dart
// core/themes/light_theme.dart
ThemeData getLightTheme({Color? seedColor}) {
  final colorScheme = ColorScheme.fromSeed(
    seedColor: seedColor ?? kDefaultSeedColor,
    brightness: Brightness.light,
  );
  return ThemeData(colorScheme: colorScheme, useMaterial3: true, ...);
}
```

`AppThemeCubit` 管理主题模式与主题色，状态持久化于 `UserPreferenceCache`。

### 3.3 认证流程

```
用户登录 → 存储 accessToken + refreshToken（SharedPreferences + AES 加密）
    ↓
每次请求 → AuthenticationInterceptor 注入 Authorization 头
    ↓
401 响应 → 自动用 refresh token 换新 access token，经同一 Dio 实例重试
    ↓ 成功               ↓ 失败
更新 Token 继续请求    清除凭证 → 跳转登录页
```

> 修复历史：此前拦截器被注释、`autoRefreshToken=false`、重试用新建 Dio 实例（无 baseUrl/无拦截器必失败），刷新链断裂。现均修复，见 `http_client.dart` 注释与仓库 `docs/oa-workflow-design.md` §2.1 自描述 RT 段。

**登录状态管理**：通过 `UserAuthCache`（基于 GetIt 单例）+ `ValueNotifier` 实现响应式登录状态。

### 3.4 业务页面

OA 业务页面位于 `features/oa/pages/`，覆盖审批任务列表与详情、提交申请、考勤打卡、请假、报销、出差/加班/用印/外出、通讯录（只读）、通知、个人中心、资料编辑、设置。各业务页与对应 service 的实现状态见仓库 `docs/oa-mobile-design.md` §4。

---

## 四、项目目录结构与职责

```
lib/
├── main.dart                    # 程序入口（init + MultiBlocProvider）
├── src/
│   ├── app.dart                 # OAApp（ScreenUtilInit + MaterialApp.router）
│   ├── app_router/
│   │   ├── app_router.dart      #   GoRouter 路由配置
│   │   └── route_names.dart     #   路由名称常量
│   ├── core/                    # 核心基础设施（基座）
│   │   ├── config/environments.dart     # 环境变量管理
│   │   ├── constants/                   # breakpoints / router_paths / index
│   │   ├── preference/                  # UserPreferenceCache（SharedPreferences）
│   │   ├── repositories/user_auth_cache.dart  # 登录状态 + Token 管理
│   │   ├── services/                    # base_service（异常处理）+ pagination_query
│   │   ├── themes/                      # cubit/（AppThemeCubit）+ light/dark_theme + fonts
│   │   ├── transport/http/              # Dio + dio_client_transport + 拦截器 + status
│   │   ├── utils/responsive_utils.dart  # 响应式工具
│   │   └── widgets/                     # responsive_layout / app_back_button / error_page / not_found_page
│   └── features/
│       ├── auth/
│       │   ├── pages/login_page.dart    # 登录页
│       │   └── services/authentication_service.dart
│       └── oa/
│           ├── pages/                   # OA 业务页面（审批/考勤/请假/报销/出差/加班/用印/外出/通讯录/通知/个人中心/资料/设置/提交申请/详情）
│           └── services/                # OA 业务 service（见 api-guide.md §9 清单）
├── generated/                        # [自动生成]
│   ├── l10n.dart                     #   国际化 S 类（intl_utils）
│   └── api/                          #   transport + app/service/v1（buf）
└── l10n/                             # 国际化 ARB 文件
    ├── intl_zh_CN.arb
    └── intl_en_US.arb
```

> `features/` 下含 `auth/` 与 `oa/`。基座代码（`lib/src/core/`）随仓初始化从模板拷入，与上游模板无 git 关联，升级需手动 diff 同步。

---

## 五、二次开发导引

### 5.1 环境搭建

```bash
flutter pub get                              # 安装依赖
flutter pub run intl_utils:generate          # 生成 i18n
cd backend/api && buf generate --template buf.app.dart.gen.yaml   # 生成 Dart API 客户端
flutter run -d android / ios                 # 移动端开发
flutter analyze                              # 代码分析
flutter test                                 # 测试
```

**环境变量配置**（`.dev.env`）：见 §2.1。修改 `.env` 后需重启应用。

### 5.2 新增一个业务页面

以「会议预订」模块为例：

**Step 1 — 生成 API 客户端**

在 `backend/api/protos/app/service/v1/` 下新增/更新 wrapper proto（定义带 `google.api.http` 注解的 service，引用 `oa.service.v1` 消息类型），然后运行代码生成：

```bash
cd backend/api && buf generate --template buf.app.dart.gen.yaml
```

这会在 `lib/generated/api/app/service/v1/index.dart` 的 `ApiClient` 上暴露对应的 `*ServiceClient` getter 及相关类型。

**Step 2 — 封装服务层**

创建 `lib/src/features/oa/services/meeting_service.dart`：

```dart
import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;

class MeetingService extends BaseService {
  MeetingService() : super(tag: 'MeetingService');

  oaApi.MeetingServiceClient get _api =>
      GetIt.instance<oaApi.ApiClient>().meetingService;

  Future<dynamic> list([PaginationQuery? query]) async {
    final q = query ?? const PaginationQuery();
    try {
      return await _api.listMeeting(oaApi.PaginationPagingRequest(
        page: q.page, pageSize: q.pageSize,
        /* q.toRawParams() */
      ));
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }
}
```

**Step 3 — 创建页面**

创建 `lib/src/features/oa/pages/meeting/meeting_page.dart`（`StatefulWidget` + `setState`，移动端 UI 用 ScreenUtil）。

**Step 4 — 注册路由**

在 `app_router.dart` 添加 `GoRoute`，在 `route_names.dart` 和 `router_paths.dart` 添加对应常量。

### 5.3 新增一种语言

**Step 1** — 在 `lib/l10n/` 下创建新 ARB 文件（如 `intl_ja_JP.arb`），复制现有文件并翻译。

**Step 2** — 生成代码：
```bash
flutter pub run intl_utils:generate
```

**Step 3** — 在 `core/themes/cubit/app_theme_cubit.dart` 的 locale 选项中添加新语言。

### 5.4 自定义主题配色

修改 `core/themes/const.dart` 的默认色与设置页的预设主题色列表。所有使用 `theme.colorScheme.*` 的组件会自动跟随变化，因色板由 `ColorScheme.fromSeed` 动态生成。

### 5.5 对接不同后端

1. **`.dev.env` / `.env`** — 修改 `API_BASE_URL`
2. **`backend/api/protos/app/service/v1/`** — 更新 wrapper proto
3. **重新生成 API 代码** — `cd backend/api && buf generate --template buf.app.dart.gen.yaml`
4. **`features/{auth,oa}/services/*.dart`** — 调整请求参数格式和响应结构

认证流程可通过修改 `transport/http/interceptors/` 下的拦截器自定义。

---

## 六、开发规范与注意事项

### 6.1 响应式适配规范

- 移动端 UI 可使用 `flutter_screenutil` 的 `.w` / `.h` / `.sp` 适配
- **Web 端禁止使用 `.w` / `.h` / `.sp`**（Web 端 ScreenUtil designSize 被设为视窗尺寸，1:1 映射，使用固定值）
- 详细规范见 [ui-adaptation.md](./ui-adaptation.md)

### 6.2 API 服务规范

- 所有 Service 必须继承 `BaseService`，使用 `handleDioError` 统一处理 `DioException`
- 使用 `PaginationQuery` 封装分页参数，不要手动拼接 query 字符串
- 调用方必须判断返回类型（生成响应类型 vs `Status`）

### 6.3 路由注意事项

- 使用 `context.go()` 进行顶级路由切换
- 使用 `context.push()` 进行子页面导航
- 返回按钮使用 `AppBackButton`（内置 canPop 检查 + fallback）
- 路由路径常量集中管理在 `router_paths.dart` 和 `route_names.dart`

### 6.4 代码生成

修改以下内容后需重新生成代码：

| 修改内容 | 生成命令 |
|---|---|
| `protos/app/service/v1/` 下 wrapper proto | `cd backend/api && buf generate --template buf.app.dart.gen.yaml` |
| ARB 翻译文件 | `flutter pub run intl_utils:generate` |

---

## 七、技术亮点总结

1. **类型安全的 API 层**：proto wrapper 经 buf 自动生成 → `ApiClient` 类型安全客户端 → `BaseService` 统一错误处理
2. **响应式断点设计**：三级断点（手机/平板/Web）+ `ResponsiveLayout` 双视图组件（基座保留）
3. **ShellRoute 持久化导航**：底部导航栏贯穿 OA 各业务 Tab
4. **Material 3 动态主题**：`ColorScheme.fromSeed` 实现动态主题色
5. **Token AES 加密持久化**：SharedPreferences + AES 对称加密，安全存储认证凭证
6. **flutter_bloc Cubit 模式**：轻量级状态管理，主题/语言状态全局共享
7. **PaginationQuery 统一分页**：封装 locale 自动注入、query JSON 序列化
8. **自描述 refresh token 刷新链**：RT 携带身份，刷新链在 BFF 边端不被截断（见后端文档 §2.1）

---

> **快速开始**：`flutter pub get && flutter pub run intl_utils:generate && flutter run -d android`。
