# go-wind-oa 移动端架构设计文档

> ⚠️ **本文档已过时。** 撰写于旧的单 service 架构时期（`app/oa/service`），描述的是 `swagger_parser` 消费 openapi 生成 Dart 客户端的链路、`oa/v1` proto 路径、`oa/v1/authentication.proto` 等。当前实际架构为 core/admin/app 三 service 分離，Dart 客户端由 `buf.flutter.oa.dart.gen.yaml` 直接生成于 `app/service/v1`，鉴權走 cms `i_authentication.proto` wrapper，无 swagger_parser 中间层。以 `README.md` 与代码为权威架构来源。
>
> 本文档面向维护者，记录 OA 移动端（Flutter）的架构决策、与基座的关系、四功能的实现状态与后端待办。读者应已熟悉 Flutter + go_router + cached_query + dio 的基本范式。

---

## 1. 仓结构决策

OA 移动端代码落在 `go-wind-oa/frontend/mobile/`。该目录由 `go-wind-cms/frontend/app/flutter_app` **拷贝而来**（非 git submodule、非 pub 依赖），即"拷基座"模式。

**决策理由**：OA 仓自包含前后端，与 cms 仓解耦；代价是基座代码（`lib/src/core/`：transport / BaseService / UserAuthCache / 鉴权拦截器 / 路由守卫基础设施）与 cms 重复，后续基座升级不会自动同步——此取舍已在前期 plan 阶段经用户确认接受。

**拷贝后所做的剥离**：
- `lib/src/features/cms/` 整目录删除（cms 内容展示页、TagService、PostService 等）。
- `lib/generated/api/` 清空（cms 的 `app/service/v1` 生成产物），由 swagger_parser 消费 OA openapi 重新生成。
- `lib/src/app_router/app_router.dart` 重写为 OA 路由（见 §3）。
- `lib/src/app_router/route_names.dart`、`lib/src/core/constants/router_paths.dart` 重写，仅留 OA + login + notFound。
- `swagger_parser.yaml` 的 `schema_path` 改指向 OA 后端的 `backend/app/oa/service/cmd/server/assets/openapi.yaml`。

**保留不动**：`lib/src/core/`（除上述常量文件）、`lib/src/features/auth/`（登录页 + AuthenticationService，对接 cms JWT 签发——见 §5 鉴权缺口）、`lib/l10n/`（arb 国际化框架，仅新增 OA 字串）。

---

## 2. API 代码生成链

OA 移动端的 Dart API 客户端由 **swagger_parser** 生成，消费 OA 后端经 **protoc-gen-openapi** 产出的 OpenAPI v3 文档：

```
protos/oa/v1/*.proto
        │  (buf.oa.openapi.gen.yaml + protoc-gen-openapi)
        ▼
backend/app/oa/service/cmd/server/assets/openapi.yaml
        │  (swagger_parser.yaml schema_path 指向此)
        ▼
frontend/mobile/lib/generated/api/.../index.dart
```

**触发命令**：
```bash
# 后端：生成 openapi.yaml
cd backend/app/oa/service && make openapi

# 前端：生成 Dart 客户端（需 Flutter SDK）
cd frontend/mobile && dart run bin/clean_and_gen.dart
```

**关键约定**：
- `swagger_parser.yaml` 配 `squash_clients: true` + `root_client: true`，故生成单一 `ApiClient`，按服务名暴露懒加载属性（OA 即 `apiClient.workflowService`）。
- 生成的客户端经 `lib/src/core/transport/http/dio_client_transport.dart` 适配为 `ClientTransport`，复用基座的 dio + 鉴权拦截器（自动注入 `Authorization: Bearer <token>`）。
- 生成产物的具体子路径（`.../oa/v1/index.dart` 还是 `.../service/v1/index.dart`）由 openapi 的 servers/paths 命名空间决定。**本文档撰写时未能在本机跑 swagger_parser 验证（Flutter SDK 缺失），故 OA service 与页面 import 用的是 `package:flutter_app/generated/api/oa/v1/index.dart`**——若实际生成路径不同，调整 `workflow_service.dart` 与 `task_detail_page.dart` 的 import 即可。

---

## 3. 路由与 Shell

OA 移动端路由（`lib/src/app_router/app_router.dart`）：

- **ShellRoute**（`OaShellPage`）：底部导航三 Tab——`/oa/tasks`（工作流）/ `/oa/notifications`（通知）/ `/oa/attendance`（考勤）。`OaShellPage` 据 `currentRoute` 高亮 Tab，点击切路由；不持有业务状态。
- **子路由**：`/oa/tasks/detail/:id`（任务详情），挂在 `/oa/tasks` 下，进同一 Shell（高亮工作流 Tab）。
- **非 Shell 路由**：`/oa/apply`（提交申请，全屏表单）、`/login`（登录）。
- **守卫 `_guard`**：依 `GetIt.instance<UserAuthCache>().hasLogin` 判定；未登录访问任意路由 → `/login`；已登录访问 `/login` → `/`（工作流 Tab）。

---

## 4. 四功能实现状态

### 4.1 工作流审批 ✅ 完整
- `WorkflowService`（`services/workflow_service.dart`）：extends `BaseService`，持有生成的 `apiClient.workflowService`；提供 `pendingTasks`/`submittedTasks`（列表直接调用）+ `audit`（审批/驳回/转交）+ `submitApply`（提交申请）。
- 列表页（`task_list/oa_task_list_page.dart`）：两 Tab，各 `Future` 驱动 `ListView`，行展示 title/status_label/occurred_at，点行进详情，FAB 进提交申请。
- 详情页（`task_detail/oa_task_detail_page.dart`）：审批按钮三枚（同意/驳回/转交），转交弹 dialog 收 forwardToUserId，调 `audit`。
- 提交申请页（`submit_apply/oa_submit_apply_page.dart`）：表单收 definition_code/version/title/form_data，调 `submitApply`。
- 审批完成后调 `Navigator.maybePop` 返回列表；列表 `initState` 重新拉取，故体现"少一笔"。

### 4.2 即時消息推送 🟡 骨架
- `NotificationService`：`notificationStream` 为空 Stream（永不产出）。
- 页面：`StreamBuilder` 监听该 stream，无数据时显示"推送未配置"占位。
- `pubspec.yaml` 未启用 `firebase_messaging` / `jpush_flutter`（待后端通道定稿）。

### 4.3 移動考勤與打卡 🟡 骨架
- `AttendanceService`：`checkIn` 永远返回 `notConfigured`，`isInFence` 永远 false。
- 页面：打卡按钮调 `checkIn`，展示"考勤服务未配置"占位。
- `pubspec.yaml` 未启用 `geolocator` / `wifi_iot`（待后端落地）。

### 4.4 運維微控制台 ❌ 剔除
经用户确认无"云手机"功能，本字段剔除。

---

## 5. 后端待办

以下为本次移动端落地过程中暴露的接口/能力缺口。其中 §5.1 鉴权缺口与 §5.2 单任务详情端点均已闭合（详见各节）；§5.3–§5.4 仍未落地，每项标注影响的功能与最小落地路径。

### 5.1 鉴权缺口（已解决）
**现状**：路径 A 已落地。OA 后端新增 `AuthenticationService`（`api/protos/oa/v1/authentication.proto`，5 消息 + 3 枚举 + 5 RPC，wire 格式与 cms `authentication.service.v1` 逐字段对齐），三条生成链（go / TS / openapi）均收录。

**转发机制**：OA 后端 `internal/service/authentication_service.go` 实现 `AuthenticationServiceHTTPServer`，各方法经 `proto.Marshal`/`proto.Unmarshal` 做 oa↔cms 跨包翻译（wire-compatible，完整保留 `LoginRequest.Identifier` oneof 与所有 optional 字段），再转发至 cms `admin-service` 的 `authenticationServiceClient`（gRPC 客户端由 `internal/data/auth_client.go` 经服务发现定位 `admin-service` 构造，与 `notification_client.go` 定位 `core-service` 同款模式）。`Logout`/`RefreshToken` 的 `UserId`/`ClientType`/`Jti` 由 cms auth 中间件注入的 `OperatorMetadata`（`auth.FromContext`）回填——与 cms `admin-service` 同款流程。

**白名单**：`Login`/`GenerateCaptcha`/`VerifyCaptcha` 经 `rpc.AddWhiteList` 放行（获取验证码时尚无 token，鸡生蛋问题）；`Logout`/`RefreshToken` 不在白名单，需 JWT。

**两端 composable**：
- admin：`frontend/admin/src/api/composables/auth.ts` 已从 stub 还原为真实实现（对齐 `go-wind-admin` 同名文件，类型取自 `@/api/generated/oa/v1`，剔除 `RegisterUser`——依赖 cms `user.proto`，OA 未含）。
- mobile：基座 `lib/src/features/auth/` 复用，`swagger_parser` 从 OA `openapi.yaml` 重新生成的客户端含 `authenticationService`，`UserAuthCache` 可正常存取 token，路由守卫不再死锁。

**鉴权死锁已解除。**

### 5.2 单任务详情端点（已解决）
**现状**：`GetTask` 已落地。OA 后端 `api/protos/oa/v1/workflow.proto` 新增 `GetTask` RPC（HTTP `GET /admin/v1/oa/workflow/tasks/{task_id}`）+ `GetTaskRequest` / `GetTaskResponse` / `AuditLogEntry` 消息。三条生成链均收录。

**鉴权**：与 `AuditTask` 完全同款——调用 `taskRepo.GetState` 取 assignee / taskStatus，校验 `assignee == caller` 且 task 处于 PENDING，否则 `Forbidden`。此为纵深防御：`GetDetailByAssignee` 在 DB 层以 `IDEQ + AssigneeUserIDEQ + TaskStatusEQ(PENDING)` 三重谓词同样拒绝非归属任务，任一层失败即拒。tenant 由 `TenantPrivacy` 策略按 viewer 自动隔离。

**投影**：
- `taskRepo.GetDetailByAssignee` 投影 task + 关联实例的 `Title` / `FormData`（`FormData` 为 `field.Any`，按 `GetDefinitionNodeConfig` 同款 `any→string` 经 `json.Marshal` 落回原始 JSON 文本）；
- `logRepo.ListByInstance` 投影该实例的审批日志轨迹（`APPROVE/REJECT/FORWARD`，与 `ListByActor` 同款口径，`SUBMIT` 排除）。
- 其余字段（`definition_id` / `current_node_index` / `tenant_id` 等内部字段）不投影，对齐 `MyTaskItem` 的最小披露原则。

**移动端**：`workflow_service.dart` 增 `getTaskDetail(taskId)` 直接调用方法；`task_detail_page.dart` 经 `FutureBuilder` 调用并渲染申请标题、表单数据（只读 JSON 文本）、审批历史轨迹。新增 l10n 键 `oaTaskDetailSummaryTitle` / `oaTaskDetailFormDataTitle` / `oaTaskDetailHistoryTitle` / `oaTaskDetailHistoryEmpty`（en_US + zh_CN arb）。

**单任务详情端点缺口已闭合。**

### 5.3 推送通道（阻塞 4.2）
**现状**：`NotificationService.notificationStream` 为空 Stream。

**落地**：
- 后端：经 cms `internal_message` 的 SSE 通道（admin 网关 `InternalMessagePublisher`）向 OA 移动端推送 notification 事件。需 OA 后端转发 cms SSE 端点，或 cms 直接对 OA 移动端开 SSE。
- 前端：新增 SSE 客户端订阅 `/admin/v1/.../events`，事件投递到 `notificationStream`。
- FCM/JPush：需服务端 push token 注册接口 + 设备令牌与用户绑定。

### 5.4 考勤服务（阻塞 4.3）
**现状**：`AttendanceService.checkIn` 永远 `notConfigured`。

**落地**：
- 后端：考勤服务接口（打卡记录、班次规则）。
- 地理围栏库：公司方圆 N 米的多边形/半径定义。
- 公司 Wi-Fi 指纹库：SSID/BSSID 白名单。
- 前端：启用 `geolocator`（GPS）+ `wifi_iot`（SSID 读取），判定在围栏/白名单内才允许打卡。

### 5.5 定义启用/禁用接口（影响 admin 前端完整性）
**现状**：OA 后端 `CreateWorkflowDefinition` 一律落 `DRAFT`，`SubmitApply` 校验 `ENABLED`，但无切换接口。admin 前端列表仅展示状态 Tag，无法切换。

**落地**：补 `UpdateWorkflowDefinition`（带 `update_mask` 限定 `definition_status`）或专用启用接口。

---

## 6. 构建

```bash
# 1. 后端生成 openapi.yaml
cd backend/app/oa/service && make openapi

# 2. 前端生成 Dart 客户端（需 Flutter SDK，本机缺失时用户本地执行）
cd frontend/mobile && dart run bin/clean_and_gen.dart

# 3. 运行
flutter run
```

**注意**：步骤 2 依赖 Flutter SDK，当前开发机未安装（仅 fvm 壳）。用户本地 `fvm use <version>` 后执行。

---

## 7. 维护提示

- 基座升级（cms flutter_app 更新）不会自动同步到本目录。需定期 diff cms 源目录与 `frontend/mobile/lib/src/core/`，手动 cherry-pick。
- `package:flutter_app/` 导入前缀保持不变（pubspec `name` 仍为 `flutter_app`），避免大面积重命名。如需改名，全局替换 import。
- `lib/generated/` 与 `lib/generated/intl/`（l10n）均为生成产物，不入库；构建时由 `clean_and_gen.dart` 与 `flutter_intl` 重新生成。
