# API 层开发指南

本文档面向二开人员，介绍 go-wind-oa 移动端（Flutter）的 API 架构设计、代码生成流程、Service 编写规范和分页查询机制。

> 本项目为协同办公系统（OA）移动端。后端三服务架构与 proto 域分离见仓库 `docs/oa-workflow-design.md`，移动端整体架构见 `docs/oa-mobile-design.md`。

---

## 1. 整体架构

```
┌─────────────────────────────────────────────────────────┐
│  UI 层（Page / Widget）                                  │
│  调用 Service 方法，处理返回数据                            │
├─────────────────────────────────────────────────────────┤
│  Service 层（lib/src/features/{auth,oa}/services/）       │
│  业务封装，异常处理                                         │
│  继承 BaseService → handleDioError()                     │
├─────────────────────────────────────────────────────────┤
│  生成层（lib/generated/api/）                             │
│  transport.dart（ClientTransport 抽象）                   │
│  app/service/v1/index.dart（ApiClient + 各 ServiceClient │
│  + 全部请求/响应类型）                                     │
├─────────────────────────────────────────────────────────┤
│  传输层（lib/src/core/transport/）                        │
│  DioClientTransport 适配 buf ClientTransport → Dio       │
│  Dio 实例 + AuthenticationInterceptor                    │
│  环境配置（.env → Environments.apiBaseUrl）                │
└─────────────────────────────────────────────────────────┘
```

**调用链路：** `Page → Service → ApiClient.XxxService → DioClientTransport → Dio → HTTP`

---

## 2. 代码生成流程

Dart API 客户端由 **buf.app.dart.gen.yaml** 直接生成：

```
protos/app/service/v1/*.proto
        │  (buf.app.dart.gen.yaml + protoc-gen-dart-http)
        ▼
frontend/mobile/lib/generated/api/app/service/v1/index.dart   (+ transport.dart)
```

**触发命令**：

```bash
cd backend/api && buf generate --template buf.app.dart.gen.yaml
```

### 2.1 生成产物

```
lib/generated/api/
├── transport.dart                    # ClientTransport 抽象（unary/serverStream/duplexStream）+ TransportMeta
└── app/service/v1/
    └── index.dart                    # ApiClient（聚合入口）+ 各 *ServiceClient + 全部请求/响应类型
```

`ApiClient`（`index.dart`）按服务名暴露懒加载的 `*ServiceClient` getter，当前共 13 个：`attendanceService` / `authenticationService` / `businessTripService` / `expenseService` / `fileTransferService` / `internalMessageService` / `leaveService` / `orgUnitService` / `outingService` / `overtimeService` / `sealApplicationService` / `userProfileService` / `userService` / `workflowService`。类型名带包前缀（`OaServiceV1*` / `IdentityServiceV1*` / `AuthenticationServiceV1*` / `Internal_messageServiceV1*` / `StorageServiceV1*`），枚举成员为小写。

### 2.2 重要原则

- **禁止手动编辑** `lib/generated/api/` 下任何文件。
- 修改 `protos/app/service/v1/` 下 wrapper proto 后，重新运行 `buf generate --template buf.app.dart.gen.yaml`。
- 各 wrapper 的 `google.api.http` 注解定义了 `/app/v1/...` 路径，Dart 生成器据此产生带路径的 client 方法。消息类型引用 `oa.service.v1` / `internal_message.service.v1` / `authentication.service.v1` / `identity.service.v1` / `storage.service.v1`，生成器自动跟随 import 解析。

> `buf.app.dart.gen.yaml` 的 `inputs` 覆盖 `protos/app/service/v1` 全目录，各 wrapper 的 `google.api.http` 注解定义了 `/app/v1/...` 路径。

---

## 3. 传输层

### 3.1 Dio 初始化

文件：`lib/src/core/transport/http/http_client.dart`

```dart
Dio createDio() {
  final Dio dio = Dio();
  _configureOptions(dio);      // baseUrl/connectTimeout/receiveTimeout 从 Environments 读
  _configureInterceptors(dio); // 仅注册 AuthenticationInterceptor
  return dio;
}
```

Dio 实例通过 `transport/init.dart` 注册为全局单例：

```dart
getIt.registerLazySingleton<Dio>(() => createDio());
```

### 3.2 ClientTransport 适配

文件：`lib/src/core/transport/http/dio_client_transport.dart`

`DioClientTransport` 实现 buf 生成的 `ClientTransport` 接口：`unary` 转发到 `_dio.request`，`serverStream`/`duplexStream` 抛 `UnimplementedError`。由此把生成的 `ApiClient` 调用桥接到 Dio。

### 3.3 环境配置

文件：`lib/src/core/config/environments.dart`

通过 `flutter_dotenv` 从 `.env`（生产）或 `.dev.env`（开发）加载，导出 `apiBaseUrl` / `sseUrl` / `connectionTimeout` / `receiveTimeout` / `aesKey` 等。当前 `.dev.env` 指 `http://localhost:6700`（app-service）/ `http://localhost:6701/events`（SSE）。

```env
# .dev.env
API_BASE_URL="http://localhost:6700"
SSE_URL="http://localhost:6701/events"
CONNECTION_TIMEOUT=3000
RECEIVE_TIMEOUT=3000
```

---

## 4. Service 层编写规范

### 4.1 标准 Service 结构

继承 `BaseService`，通过 GetIt 取 `ApiClient` 访问对应 service client，方法体统一 `try-catch + handleDioError`：

```dart
// lib/src/features/oa/services/xxx_service.dart

class XxxService extends BaseService {
  XxxService() : super(tag: 'XxxService');

  oaApi.XxxServiceClient get _api =>
      GetIt.instance<oaApi.ApiClient>().xxxService;

  Future<dynamic> list([PaginationQuery? query]) async {
    final q = query ?? const PaginationQuery();
    try {
      return await _api.listXxx(oaApi.PaginationPagingRequest(
        page: q.page,
        pageSize: q.pageSize,
        // ...
      ));
    } on DioException catch (e) {
      return handleDioError(e);   // Dio 异常 → Status 对象
    }
  }
}
```

> import 生成代码：`import 'package:flutter_app/generated/api/app/service/v1/index.dart' as oaApi;`。类型用包前缀全名（如 `oaApi.OaServiceV1MyTaskItem`）。

### 4.2 异常处理模式

所有 Service 方法统一使用 `try-catch + handleDioError`：

```dart
Future<dynamic> get(int id) async {
  try {
    return await _api.getXxx(id: id);
  } on DioException catch (e) {
    return handleDioError(e);  // → Status(code, reason, message, metadata)
  }
}
```

`handleDioError` 定义在 `BaseService`（`core/services/base_service.dart`）中，将 `DioException` 转为统一的 `Status` 对象：

```dart
Status handleDioError(DioException e) {
  final data = e.response?.data;
  if (data is Map<String, dynamic>) {
    return Status(
      code: e.response?.statusCode,
      reason: data['reason'],
      message: data['message'] ?? e.message,
      metadata: ...,
    );
  }
  return Status(code: e.response?.statusCode, reason: e.type.name, message: e.message);
}
```

### 4.3 返回值约定

| 方法类型         | 成功返回                     | 失败返回        |
|--------------|--------------------------|-------------|
| 查询（list/get） | 生成响应类型或模型                 | `Status` 对象 |
| 创建/更新         | 创建/更新后的资源模型               | `Status` 对象 |
| 删除           | `null`                   | `Status` 对象 |

**调用方需要判断返回类型：**

```dart
final result = await _xxxService.list(query);
if (result is oaApi.OaServiceV1XxxResponse) {
  // 成功，使用 result.items
} else if (result is Status) {
  // 失败，显示 result.message
}
```

> 服务端鉴权失败（如 OA 列表对当前调用者不可见）会以 `Status`（403）返回，调用方按失败分支处理，详见 `docs/oa-workflow-design.md` 的可见性说明。

---

## 5. 分页查询（PaginationQuery）

### 5.1 概述

文件：`lib/src/core/services/pagination_query.dart`

统一封装所有 List API 的查询参数，`toPagingRequest()` 生成 buf 产出的 `PaginationPagingRequest`，`toRawParams()` 生成原始 Map。

### 5.2 核心参数

```dart
PaginationQuery(
  page: 1,                    // 页码（从 1 开始）
  pageSize: 10,               // 每页条数
  formValues: {               // 过滤条件 → 序列化为 JSON query 字符串
    'status': 'pending',
  },
  orderBy: ['-created_at'],   // 排序
  fieldMask: 'id,title',      // 字段白名单（SELECT）
  locale: 'zh-CN',            // 语言（不传则自动从用户偏好获取）
  skipLocale: false,           // 是否跳过 locale 注入
  isTenantUser: false,         // 是否清理租户字段
);
```

### 5.3 计算属性

| 属性                   | 说明                                               |
|----------------------|--------------------------------------------------|
| `noPaging`           | `page == null && pageSize == null` 时为 true（全量加载） |
| `queryString`        | `formValues` + `locale` 合并为 JSON 字符串，自动清理 null   |
| `orderByString`      | 排序列表 → JSON 数组字符串                                |

### 5.4 locale 自动注入

`PaginationQuery` 会自动从 `UserPreferenceCache` 读取用户语言偏好，注入到 `query` 的 `locale` 字段。不涉及多语言内容的 API 可 `skipLocale: true`。

---

## 6. 认证拦截器

文件：`lib/src/core/transport/http/interceptors/authentication_interceptor.dart`

`http_client.dart` 的 `_configureInterceptors` 注册该拦截器（`autoRefreshToken: true`，注入同一 Dio 实例）。

功能：
- `onRequest`：除 `/login` 路径外，自动注入 `Authorization: Bearer <token>`。
- `onError`：401 且 `autoRefreshToken=true` 时，用 refresh token 换新 access token 并经**同一 Dio 实例**重试（确保继承 baseUrl 与拦截器链）。单次重试守卫防递归，单飞 `Completer` 锁防并发刷新。
- 刷新失败触发 `authenticationFailed()` 回调。

> 修复历史：此前该拦截器被整段注释、`autoRefreshToken=false`、重试用 `new Dio()` 新建实例（无 baseUrl/无拦截器必失败），导致所有需鉴权的 API 调用不带 Authorization 头、刷新链断裂。现均修复，见 `http_client.dart` 注释。

---

## 7. 文件传输

文件：`lib/src/features/oa/services/file_upload_service.dart`

报销发票上传经 `fileTransferService`（`ApiClient.fileTransferService`，引用 `storage.service.v1`）的 multipart `POST /app/v1/file/upload`，返回 file_id。该 service 仅负责 multipart 上传；拍照/相册选取与压缩（`image_picker` 的 `maxWidth`/`imageQuality`）在调用方页面执行，返回的 file_id 由页面回填到明细 `invoiceFileId`。

```dart
final result = await _uploadService.uploadInvoice(bytes, fileName);
if (result is oaApi.StorageServiceV1UploadFileResponse) {
  _items[i].invoiceFileId = result.fileId;  // 回填明细
}
```

---

## 8. 新增 API 接入 Checklist

当后端新增或修改 `protos/app/service/v1/` 下 wrapper proto 后，按以下步骤操作：

1. **重新生成代码**
   ```bash
   cd backend/api && buf generate --template buf.app.dart.gen.yaml
   ```

2. **检查生成产物** — 确认 `lib/generated/api/app/service/v1/index.dart` 中 `ApiClient` 暴露了对应的 `*ServiceClient` getter 及请求/响应类型。

3. **新建或更新 Service**
   - 在 `lib/src/features/{auth,oa}/services/` 下新建或编辑 Service 文件
   - 继承 `BaseService`
   - import `package:flutter_app/generated/api/app/service/v1/index.dart`
   - 实现 CRUD 方法，统一 `try-catch + handleDioError`

4. **使用 Service**
   - 在 Page/Widget 中 import Service
   - 调用方法，判断返回类型（生成响应类型 vs `Status`）

---

## 9. 现有 Service 清单

| Service                  | 文件                          | 对应 ApiClient getter           |
|--------------------------|-----------------------------|-------------------------------|
| AuthenticationService    | `auth/services/authentication_service.dart` | `authenticationService` |
| WorkflowService          | `oa/services/workflow_service.dart` | `workflowService`       |
| AttendanceService        | `oa/services/attendance_service.dart` | `attendanceService`     |
| LeaveService             | `oa/services/leave_service.dart` | `leaveService`              |
| ExpenseService           | `oa/services/expense_service.dart` | `expenseService`        |
| BusinessTripService      | `oa/services/business_trip_service.dart` | `businessTripService`  |
| OvertimeService          | `oa/services/overtime_service.dart` | `overtimeService`        |
| SealApplicationService   | `oa/services/seal_application_service.dart` | `sealApplicationService`|
| OutingService            | `oa/services/outing_service.dart` | `outingService`            |
| FileUploadService        | `oa/services/file_upload_service.dart` | `fileTransferService`  |
| NotificationService      | `oa/services/notification_service.dart` | `internalMessageService`|
| DirectoryService         | `oa/services/directory_service.dart` | `orgUnitService` + `userService` |
| UserProfileService       | `oa/services/user_profile_service.dart` | `userProfileService`   |

> 文件名与 ApiClient getter 名不一一对应（`file_upload` ↔ `fileTransferService`、`notification` ↔ `internalMessageService`、`directory` ↔ `orgUnitService`/`userService`），因 service 按业务功能命名、getter 按 proto service 命名。

---

## 10. 相关文件索引

| 文件路径                                           | 说明                |
|------------------------------------------------|-------------------|
| `backend/api/buf.app.dart.gen.yaml`            | Dart API 生成模板     |
| `lib/generated/api/transport.dart`             | ClientTransport 抽象 |
| `lib/generated/api/app/service/v1/index.dart`  | ApiClient + ServiceClient + 类型 |
| `lib/src/core/services/base_service.dart`      | Service 基类（异常处理）  |
| `lib/src/core/services/pagination_query.dart`  | 分页查询封装            |
| `lib/src/core/transport/http/http_client.dart` | Dio 初始化           |
| `lib/src/core/transport/http/dio_client_transport.dart` | ClientTransport 适配 |
| `lib/src/core/transport/http/interceptors/authentication_interceptor.dart` | 认证拦截器 |
| `lib/src/core/transport/init.dart`             | 传输层注册             |
| `lib/src/core/config/environments.dart`        | 环境变量              |
| `.dev.env` / `.env`                            | 环境配置文件            |
