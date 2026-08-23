# AGENTS.md — go-wind-oa 移动端（Flutter）开发指南

> 本文件是 `frontend/mobile/` 子项目的 AI 编码规范单一事实源，适用于所有支持 AGENTS.md 的 AI 编码工具。Claude Code 通过同级 `CLAUDE.md` 中的 `@AGENTS.md` 引用加载。
>
> 本项目为协同办公系统（OA）的移动端。后端架构见仓库 `docs/oa-workflow-design.md`，移动端架构导览见 `docs/oa-mobile-design.md`。

## 项目概览

基于 **Flutter** 的协同办公移动端，一套 Dart 代码编译为 iOS / Android（兼可编 Web/macOS/Windows/Linux，但实际部署目标是移动端）。

**核心技术栈**：Flutter 3.x (Dart 3.x) + flutter_bloc/Cubit（状态管理）+ GoRouter（路由）+ GetIt（IoC）+ Dio（HTTP）+ protoc-gen-dart-http via buf（API 生成）+ cached_query（缓存）+ flutter_screenutil（响应式）+ flutter_intl（i18n）+ Material 3。

**代码生成工具链**：
- API 客户端：`protoc-gen-dart-http` 经 `backend/api/buf.app.dart.gen.yaml` 生成。
- i18n：`intl_utils`（flutter-intl）从 ARB 生成 `S` 类。

> `pubspec.yaml` 中 `swagger_parser` / `retrofit` / `retrofit_generator` / `freezed` / `json_serializable` / `build_builder` 等依赖无对应引用，可清理。

## 关键架构认知

### Feature-First 模块化架构

```
lib/
├── main.dart                    # 入口（init + MultiBlocProvider）
├── src/
│   ├── app.dart                 # OAApp（ScreenUtilInit + MaterialApp.router）
│   ├── app_router/              # GoRouter 路由配置 + 路由名称常量
│   ├── core/                    # 核心基础设施（基座，随仓初始化从模板拷入）
│   │   ├── config/              #   environments.dart（环境变量）
│   │   ├── constants/           #   breakpoints / router_paths
│   │   ├── preference/          #   UserPreferenceCache（SharedPreferences）
│   │   ├── repositories/        #   user_auth_cache（登录态 + Token）
│   │   ├── services/            #   base_service（统一错误处理）+ pagination_query
│   │   ├── themes/              #   cubit/（AppThemeCubit）+ light/dark_theme + fonts
│   │   ├── transport/http/      #   Dio + 拦截器 + dio_client_transport + status
│   │   ├── utils/               #   responsive_utils（响应式，基座保留）
│   │   └── widgets/             #   responsive_layout / app_back_button / error_page / not_found_page
│   └── features/                # ★ Feature-First 业务模块
│       ├── auth/                #   pages/ + services/（AuthenticationService）+ widgets/
│       └── oa/
│           ├── pages/           #   审批/考勤/请假/报销/出差/加班/用印/外出/通讯录/通知/个人中心/资料/设置/提交申请/详情
│           └── services/        #   各业务 service（见下「Service 清单」）
├── generated/                   # [自动生成] l10n.dart + api/（transport + app/service/v1）+ intl/
└── l10n/                        # i18n ARB 文件（intl_zh_CN.arb / intl_en_US.arb）
```

> `features/` 下含 `auth/` 与 `oa/` 两个域。基座代码（`lib/src/core/`）随仓初始化时从模板拷入，与上游模板无 git 关联，升级需手动 diff 同步。

### 路由与 Shell

`lib/src/app_router/app_router.dart` 定义 `GoRouter`，路径常量在 `lib/src/core/constants/router_paths.dart`。

- **ShellRoute**（`OaShellPage`，`pages/shell/oa_shell_page.dart`）：底部导航四 Tab——`/oa/tasks`（审批）/ `/oa/notifications`（通知）/ `/oa/attendance`（考勤）/ `/oa/me`（个人中心）。`OaShellPage` 为 `StatelessWidget`，据 `currentRoute` 高亮 Tab；不持有业务状态。`/oa/tasks` 下挂一个子路由 `detail/:id`（任务详情）。
- **非 Shell 全屏路由**：`/oa/apply`（通用申请）、`/oa/leave`、`/oa/expense`、`/oa/business-trip`、`/oa/overtime`、`/oa/seal-application`、`/oa/outing`、`/oa/directory`（通讯录）、`/oa/profile`（资料编辑）、`/oa/settings`（设置）、`/login`。
- **守卫 `_guard`**：依 `GetIt.instance<UserAuthCache>().hasLogin` 判定；未登录访问任意非登录路由 → `/login`；已登录访问 `/login` 或 `/` → `/oa/tasks`。

> `router_paths.dart` 中 `signIn`/`signUp`/`notFound` 常量已定义但 `signIn`/`signUp` 未接任何 `GoRoute`（死常量）。路由路径与名称常量集中管理，不要硬编码。

### 三层 API 架构

```
lib/generated/api/                     # [自动生成] protoc-gen-dart-http via buf 产出
├── transport.dart                     #   ClientTransport 抽象（unary/serverStream/duplexStream）
└── app/service/v1/index.dart          #   ApiClient（13 个 *ServiceClient 懒加载 getter）+ 全部请求/响应类型
lib/src/features/{auth,oa}/services/   # [服务封装] 继承 BaseService，封装业务 + 异常处理
```

- **第一层 — 自动生成的客户端**（`generated/api/`）：由 `backend/api/buf.app.dart.gen.yaml` 模板经 `protoc-gen-dart-http` 从 `protos/app/service/v1/` 下 wrapper proto 生成。`generated/api/app/service/v1/index.dart` 内 `ApiClient` 暴露 13 个 service client getter（`attendanceService` / `authenticationService` / `businessTripService` / `expenseService` / `fileTransferService` / `internalMessageService` / `leaveService` / `orgUnitService` / `outingService` / `overtimeService` / `sealApplicationService` / `userProfileService` / `userService` / `workflowService`）。类型名带包前缀（`OaServiceV1*` / `IdentityServiceV1*` / `AuthenticationServiceV1*` / `Internal_messageServiceV1*` / `StorageServiceV1*`），枚举成员为小写。**不应手动编辑**。
- **第二层 — 服务封装**（`features/{auth,oa}/services/`）：基于 `ApiClient` 封装业务方法，统一错误处理（`handleDioError`）。
- **第三层 — 页面调用**：页面组件直接实例化 Service 并调用方法，不额外封装 Hook。

> 生成层路径为 `app/service/v1`（非 `oa/v1`）。feature service 文件名与 `ApiClient` getter 名不一一对应：`directory_service` ↔ `orgUnitService`、`file_upload_service` ↔ `fileTransferService`、`notification_service` ↔ `internalMessageService`。

### Dio + GetIt — HTTP 通信

`lib/src/core/transport/http/http_client.dart` 的 `createDio()` 构建全局单例 Dio（经 `transport/init.dart` 注册到 GetIt），`_configureOptions` 从 `Environments` 读取 `baseUrl`/超时，`_configureInterceptors` 仅注册 `AuthenticationInterceptor`。`lib/src/core/transport/http/dio_client_transport.dart` 的 `DioClientTransport` 实现 buf 生成的 `ClientTransport` 接口，`unary` 转发到 `_dio.request`（`serverStream`/`duplexStream` 抛 `UnimplementedError`）。

`BaseService.handleDioError` 统一把 `DioException` 转 `Status` 对象（`code`/`reason`/`message`/`metadata`）。所有继承 `BaseService` 的 service 都可用。

### 认证拦截器

`lib/src/core/transport/http/interceptors/authentication_interceptor.dart`：
- `onRequest`：除 `/login` 路径外，自动注入 `Authorization: Bearer <token>`。
- `onError`：401 且 `autoRefreshToken=true` 时，用 refresh token 换新 access token 并经**同一 Dio 实例**（`_dio.fetch`）重试——确保重试请求继承 baseUrl 与完整拦截器链。单次重试守卫（`__authRetried`）防递归；refresh 请求自身失败直接退出。刷新经单飞 `Completer` 锁防并发。

> 历史坑：此前该拦截器被整段注释、`autoRefreshToken=false`、重试用 `new Dio()` 新建实例（无 baseUrl/无拦截器必失败），导致 Flutter 端所有需鉴权的 API 调用不带 Authorization 头、且刷新链断裂。现均修复，见 `http_client.dart` 注释。

### 状态管理 — BLoC / Cubit

`main.dart` 以 `MultiBlocProvider` 注入唯一一个 `AppThemeCubit`。`AppThemeCubit`（`core/themes/cubit/app_theme_cubit.dart`）管理主题模式（亮/暗/跟随系统）、主题色（seedColor）、语言切换，状态持久化于 `UserPreferenceCache`。`OAApp`（`app.dart`）经 `context.watch` 读取 `theme`/`darkTheme`/`themeMode`/`locale`。页面内局部状态用 `StatefulWidget` + `setState`。

### 主题系统（Material 3 + ColorScheme.fromSeed）

`core/themes/light_theme.dart` / `dark_theme.dart` 经 `ColorScheme.fromSeed` 生成色板，`useMaterial3: true`。`core/themes/const.dart` 定义 `kDefaultFontFamily = 'Noto Sans SC'`，`core/themes/fonts.dart` 的 `AppFont` 按平台提供 `fontFamilyFallback`。预设主题色在设置页切换。

### 国际化（flutter_intl + ARB）

```
lib/l10n/intl_zh_CN.arb / intl_en_US.arb   # 翻译源
lib/generated/l10n.dart                     # [生成] S 类（intl_utils 产出）
```

`OAApp` 经 `S.delegate` + `localeListResolutionCallback`（精确匹配 → 语言代码匹配 → 兜底）解析 locale。使用：`Text(S.of(context).xxx)`。

### 响应式适配（基座保留）

`core/constants/breakpoints.dart`（`Breakpoints`，600/1024 阈值）、`core/widgets/responsive_layout.dart`（`ResponsiveLayout` / `WebContentCenter`）、`core/utils/responsive_utils.dart`（`ResponsiveUtils`）随基座保留。`app.dart` 的 `ScreenUtilInit` 用 `Breakpoints.designSize`（375×812），`kIsWeb` 分支将 designSize 重置为视窗尺寸使 `.w/.h/.sp` 在 Web 端 1:1。详见 [ui-adaptation.md](./docs/ui-adaptation.md)。

> 实际 OA 业务页（移动端目标）基本未消费 `ResponsiveLayout`/`WebContentCenter`——响应式设施是基座遗产，移动端 UI 直接用 ScreenUtil。详见 [ui-adaptation.md](./docs/ui-adaptation.md)。

### 环境变量

`core/config/environments.dart` 经 `flutter_dotenv` 从 `.dev.env`（Debug）/ `.env`（Release）加载，导出：`ntpHost` / `apiBaseUrl` / `sseUrl` / `connectionTimeout` / `receiveTimeout` / `iosAppId` / `aesKey` / `sentryDSN` / `enableInAppLocalhostServer`。当前 `.dev.env` 指 `http://localhost:6700`（app-service）/ `http://localhost:6701/events`（SSE）。

## 关键约定（必须遵守）

1. **Service 必须继承 `BaseService`** — 用 `handleDioError` 统一处理 `DioException`。
2. **分页用 `PaginationQuery`** — `core/services/pagination_query.dart` 封装 `page`/`pageSize`/`formValues`/`fieldMask`/`orderBy`/`locale`/`skipLocale`/`isTenantUser`，`toPagingRequest()` 生成 `PaginationPagingRequest`。不要手动拼接 query 字符串。
3. **禁止手改 `lib/generated/`** — `transport.dart` + `index.dart` 由 buf 模板生成，`l10n.dart` 由 intl_utils 生成。
4. **路由用 `context.go()`（顶级切换）/ `context.push()`（子页面）** — 返回用 `AppBackButton`（内置 canPop 检查）。路由路径集中管理在 `router_paths.dart` + `route_names.dart`。
5. **Web 端禁止 `.w`/`.h`/`.sp`** — Web 端 ScreenUtil designSize 设为视窗尺寸（1:1），用固定值；移动端可用。断点用 `Breakpoints` 常量，不硬编码屏宽数值。

## 代码生成（改后必须重新生成）

| 修改内容 | 命令 |
|---|---|
| `protos/app/service/v1/` 下 wrapper proto 变更 | `cd backend/api && buf generate --template buf.app.dart.gen.yaml` |
| ARB 翻译文件 | `flutter pub run intl_utils:generate` |

## 开发命令

```bash
flutter pub get                              # 安装依赖
flutter pub run intl_utils:generate          # 生成 i18n
cd ../../backend/api && buf generate --template buf.app.dart.gen.yaml   # 生成 Dart API 客户端
flutter run                                  # 运行（需 Flutter SDK）
flutter analyze                              # 代码分析
flutter test                                 # 测试
```

> 若开发机用 fvm 管理 Flutter 版本，需先 `fvm use <version>` 再运行。包名为 `flutter_app`（pubspec `name`），所有内部 import 前缀为 `package:flutter_app/...`。

## 文档索引

- [docs/api-guide.md](./docs/api-guide.md) — API 层开发指南（生成链 / Service / 分页 / 拦截器）
- [docs/FLUTTER_TECHNICAL_GUIDE.md](./docs/FLUTTER_TECHNICAL_GUIDE.md) — 架构技术解析与二次开发导引
- [docs/ui-adaptation.md](./docs/ui-adaptation.md) — UI 适配规范（断点 / ScreenUtil / 字体）
- 仓库 [docs/oa-mobile-design.md](../../../docs/oa-mobile-design.md) — 移动端架构设计文档（与本文同源，面向维护者）
- 仓库 [docs/oa-workflow-design.md](../../../docs/oa-workflow-design.md) — 后端架构设计文档
