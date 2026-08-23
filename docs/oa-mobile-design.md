# go-wind-oa 移动端架构设计文档

> 本文档面向维护者，记录协同办公系统移动端（Flutter）的架构决策、API 生成链、路由结构、功能实现状态与已知边界。读者应已熟悉 Flutter + go_router + cached_query + Dio 的基本范式。
>
> 本文与代码同步。后端架构见 [oa-workflow-design.md](./oa-workflow-design.md)。

---

## 1. 仓结构

协同办公移动端代码落在 `go-wind-oa/frontend/mobile/`。包名为 `flutter_app`（pubspec `name`），故所有内部 import 前缀为 `package:flutter_app/...`。

```text
frontend/mobile/lib/
├── src/
│   ├── core/                  # 基座：transport / BaseService / UserAuthCache / 鉴权拦截器 / 路由守卫基础设施
│   ├── app_router/            # go_router 路由定义
│   └── features/oa/
│       ├── services/          # 各业务服务（workflow / attendance / leave / expense / file_upload / notification / user_profile）
│       └── pages/             # 审批任务列表与详情、提交申请、考勤打卡、请假、报销、出差/加班/用印/外出、通讯录、通知、个人中心、资料编辑、设置
├── generated/                 # buf / intl_utils 生成产物（勿手改）
└── l10n/                      # intl_{zh_CN,en_US}.arb
```

> 基座代码（`lib/src/core/`）随仓初始化时拷入，与 CMS 无 git 关联，后续基座升级需手动 diff 同步。

---

## 2. API 代码生成链

移动端的 Dart API 客户端由 **buf.app.dart.gen.yaml** 直接生成，无 swagger_parser / openapi 中间层：

```text
protos/app/service/v1/*.proto
        │  (buf.app.dart.gen.yaml + protoc-gen-dart-http)
        ▼
frontend/mobile/lib/generated/api/app/service/v1/index.dart
```

**触发命令**：

```bash
cd backend/api && buf generate --template buf.app.dart.gen.yaml
```

**生成产物**：`lib/generated/api/app/service/v1/index.dart`，含聚合 `ApiClient`，按服务名暴露懒加载属性。按域归类，当前属性集合如下：

| 域 | 属性 | 说明 |
|---|---|---|
| OA 审批/业务 | `workflowService` / `attendanceService` / `leaveService` / `expenseService` / `businessTripService` / `overtimeService` / `sealApplicationService` / `outingService` | 工作流引擎 + 七类业务单据（请假/报销/出差/加班/用印/外出 + 考勤）的提交、审批、查询、撤回 |
| 站内信 | `internalMessageService` | `listMyMessages` 收件箱查询（app-service 无 `SendMessage`，仅读） |
| 通讯录（只读） | `orgUnitService` / `userService` | app 侧只读 wrapper，仅 List/Get，引用 `identity.service.v1`，带 `redact` 脱敏 |
| 用户资料 | `userProfileService` | 个人资料读写（`getUser` 拉本人资料、`changePassword` 改密），引用 `identity.service.v1`；`me`/`profile` 页消费 |
| 鉴权 | `authenticationService` | Login / Logout / RefreshToken / GenerateCaptcha / VerifyCaptcha |
| 文件传输 | `fileTransferService` | 报销发票上传（`storage.service.v1` 的 multipart `POST /app/v1/file/upload`，返回 file_id 回填明细 invoiceFileId）与下载；非 CMS 内容域 |

> 上表是 `ApiClient` 当前暴露的**全集**（见 `index.dart` line 8991-9056 的 `*ServiceClient get` 属性块）。属性集合随 `protos/app/service/v1` 下 wrapper proto 增减而变；CMS 业务域随后端清除后，其对应 wrapper 已从该目录移除，不再生成对应 client。

**关键约定**：

- `buf.app.dart.gen.yaml` 的 `inputs` 覆盖 `protos/app/service/v1` 全目录（OA 业务 wrapper + 鉴权 wrapper + 只读通讯录 wrapper + 文件传输 wrapper）。各 wrapper 的 `google.api.http` 注解定义了 `/app/v1/...` 路径，Dart 生成器据此产生带路径的 client 方法。消息类型引用 `oa.service.v1` / `internal_message.service.v1` / `authentication.service.v1` / `identity.service.v1` / `storage.service.v1`，生成器自动跟随 import 解析。
- 生成的客户端经 `lib/src/core/transport/http/dio_client_transport.dart` 适配为 `ClientTransport`，复用基座的 Dio + 鉴权拦截器（自动注入 `Authorization: Bearer <token>`）。
- 类型名带包前缀（`OaServiceV1*` / `Internal_messageServiceV1*` / `IdentityServiceV1*` 等），枚举成员为小写，与生成器的命名约定一致。

> 与旧链路对比：旧设计（已废弃）经 protoc-gen-openapi 生成 openapi.yaml，再由 swagger_parser 消费生成 Dart 客户端。现改为 buf 模板直接生成，无中间层，路径为 `app/service/v1`（非 `oa/v1`）。

---

## 3. 路由与 Shell

移动端路由定义在 `lib/src/app_router/app_router.dart`，路由路径常量在 `lib/src/core/constants/router_paths.dart`。

- **ShellRoute**（`OaShellPage`，`lib/src/features/oa/pages/shell/oa_shell_page.dart`）：底部导航四 Tab——`/oa/tasks`（审批）/ `/oa/notifications`（通知）/ `/oa/attendance`（考勤）/ `/oa/me`（个人中心）。`OaShellPage` 为 `StatelessWidget`，据 `currentRoute` 高亮 Tab，`_onTap` 点击切路由；不持有业务状态。
- **子路由**：`/oa/tasks/detail/:id`（任务详情），挂在 `/oa/tasks` 下，进同一 Shell（高亮审批 Tab）。
- **非 Shell 路由**（全屏，路径常量定义于 `lib/src/core/constants/router_paths.dart`）：`/oa/apply`（通用申请表单）、`/login`（登录），以及下列 OA 业务单据提交页——`/oa/leave`、`/oa/expense`、`/oa/business-trip`、`/oa/overtime`、`/oa/seal-application`、`/oa/outing`、`/oa/directory`（通讯录）；另有 `/oa/profile`（个人资料编辑）与 `/oa/settings`（设置）。业务单据提交页的入口在 `task_list/oa_task_list_page.dart` 的 AppBar `PopupMenuButton` 中（与"通用申请"并列）；`/oa/profile` 与 `/oa/settings` 的入口在 `/oa/me` 个人中心页。
- **守卫 `_guard`**：依 `GetIt.instance<UserAuthCache>().hasLogin` 判定；未登录访问任意路由 → `/login`；已登录访问 `/login` → `/`（审批 Tab）。

---

## 4. 功能实现状态

### 4.1 审批工作流 ✅

- `WorkflowService`（`services/workflow_service.dart`）：extends `BaseService`，持有生成的 `apiClient.workflowService`；提供 `pendingTasks` / `submittedTasks` / `doneTasks` 三个列表方法（均直接调用、返回 `Future<dynamic>` 原始响应，由页面侧提取 `.items`；`doneTasks` 查已办）+ `audit`（同意/驳回/转交）+ `submitApply`（提交申请）+ `withdraw`（撤回）。`pendingTasksQuery` / `submittedTasksQuery` 为 cached_query 的 `Query` 包装（`doneTasks` 无 query 包装，页面直接调原始方法）。
- 列表页（`task_list/oa_task_list_page.dart`）：**三 Tab**（待我审批 / 已办 / 我发起的），`TabController(length: 3)`。各 Tab 用 `Future` + `setState` + `ListView.separated` 渲染（非 `FutureBuilder`）。行展示 `instanceId`（标题「申请 #${instanceId}」）/ `statusLabel`（副标题）/ `createdAt`（尾栏，经 `DateTimeUtils.formatDateTime` 格式化，非裸 ISO 文本）。仅「待我审批」Tab 行可点击进详情；「我发起的」Tab 行内嵌撤回按钮（调 `withdraw`）。FAB 进提交申请。
- 详情页（`task_detail/oa_task_detail_page.dart`）：`FutureBuilder` 驱动。审批按钮仅 同意 / 驳回 / 转交 三枚（**无撤回**——撤回在「我发起的」列表 Tab 内）。转交弹 dialog 收 `forwardToUserId`，调 `audit`。审批日志类型为 `OaServiceV1WorkflowLog`、字段 `createdAt`（`String?`），经 `DateTimeUtils.formatDateTime` 格式化渲染（无 `AuditLogEntry`/`occurredAt` 类型，非裸 ISO 文本）。
- 提交申请页（`submit_apply/oa_submit_apply_page.dart`）：表单收 `code` / `version` / `formData`（**无 `title` 字段**），并按服务端返回的 `form_schema` 动态渲染字段控件；调 `submitApply`（服务签名仅 `code`/`version`/`formData`）。

### 4.2 站内信通知 ✅

- 通知列表页从后端 `GET /app/v1/internal-message/my-messages` 拉取收件箱（core `ListMyMessages`，收件人过滤、排除已删除/已撤销）。`NotificationService`（`services/notification_service.dart`）封装 `listMessages`；页面在 `initState` 单次加载收件箱，并提供 `RefreshIndicator` 下拉刷新——**无定时轮询**（无 `Timer`/周期拉取）。
- **无 SSE 实时推送**：app-service 的 `sse_server.go` 虽存在（持 `AuthenticationServiceClient` 验 token 后允许订阅），但 `InternalMessagePublisher` 只在 admin-service 注册、且仅对 admin 自身 HTTP `SendMessage` 路径触发。core 是 gRPC-only 无 SSE server，工作流通知经 core 进程内 `SendMessage` 落库后**不会**推 SSE。故移动端通知无实时推送，仅靠上述 REST 轮询延迟可见。详见 [oa-workflow-design.md](./oa-workflow-design.md) §5.5 与 §12。

### 4.3 考勤打卡 ✅

- `AttendanceService`（`services/attendance_service.dart`）：调用后端 `POST /app/v1/oa/attendance/check-in`，提交 GPS 经纬度（`geolocator`）与尽力采集的 Wi-Fi BSSID（`network_info_plus`，不可用为 null），后端落库并按工时设置结算。若租户配置了打卡围栏或 Wi-Fi 指纹白名单，服务端 `validateLocation` 会以 403 拒绝越界/非白名单打卡（消息文本原样透传给用户），门控机制见 [oa-workflow-design.md](./oa-workflow-design.md) §8.4。
- 打卡页（`attendance/oa_attendance_page.dart`）：当日首次=签到、第二次=签退；409 已签退。

### 4.4 请假 ✅

- `LeaveService`：提交请假申请（类型/额度校验后建单，进程内直调引擎 SubmitApply 挂 LEAVE v1 流程）。半日粒度 start_half/end_half。
- 移动端请假页收集类型/起止时间/半日/事由，调服务提交。

### 4.5 报销 ✅

- `ExpenseService`：提交报销申请（多行明细 + 发票文件 ID）。
- `FileUploadService`（`services/file_upload_service.dart`）：仅负责 multipart `POST /app/v1/file/upload`（Dio `FormData`/`MultipartFile`），返回 file_id。拍照/相册选取与压缩（`image_picker` 的 `maxWidth`/`imageQuality`）在报销页（`oa_expense_page.dart`）执行，非本 service；返回的 file_id 由页面回填明细 `invoiceFileId`。
- 报销页收集明细行与发票附件，调服务提交，挂 EXPENSE v1 流程。

### 4.6 出差 / 加班 / 用印 / 外出 ✅

- 四类同型审批单据，各对应一个 service（`business_trip_service.dart` / `overtime_service.dart` / `seal_application_service.dart` / `outing_service.dart`）与提交页（`business_trip/` / `overtime/` / `seal_application/` / `outing/`）。
- 提交链路同构：建单 → 进程内直调引擎 SubmitApply 挂对应 v1 流程（`BUSINESS_TRIP` / `OVERTIME` / `SEAL_APPLICATION` / `OUTING`）。流程定义均经 `ensureWorkflowDefinition` 兜底自动创建（默认 LEADER 会签）。审批终态仅同步单据状态，无额度副作用（与报销同构，区别于请假）。
- 枚举类型名采用生成器的嵌套命名（如 `OaServiceV1SealApplication$SealStatus`），成员小写。

### 4.7 通讯录 ✅

- `DirectoryService`（`services/directory_service.dart`）：经 `apiClient.orgUnitService.list` 与 `apiClient.userService.list`（app 侧只读 wrapper，带 `redact` 脱敏）取组织树与成员。
- 通讯录页（`directory/oa_directory_page.dart`）：`ExpansionTile` 递归组织树 + 成员 `ListTile`（昵称/真名/部门标注）。仅浏览，无 CRUD。

### 4.8 个人中心 / 资料编辑 / 设置 ✅

三页位于 `/oa/me`、`/oa/profile`、`/oa/settings`，对应 `pages/me/`、`pages/profile/`、`pages/settings/`。前两者经 `UserProfileService`（`services/user_profile_service.dart`，封装 `apiClient.userProfileService`）读写个人资料，后者纯本地无 service。

- **个人中心页**（`me/oa_me_page.dart`，Shell 第四 Tab）：顶部展示当前用户头像与昵称——经 `UserProfileService.getUser` 拉取 `IdentityServiceV1User`（引用 `identity.service.v1`）。列表项为进入「资料编辑」「设置」的入口；含「退出登录」项，调 `AuthenticationService.clearTokens`（`features/auth`，非 OA 域，清 `UserAuthCache` 中令牌后回登录页）。
- **资料编辑页**（`profile/oa_profile_page.dart`）：经 `UserProfileService.getUser` 拉本人资料填充表单（昵称/性别/头像等枚举与文本字段），保存调 `UserProfileService.changePassword` 等写回端点。性别等枚举用生成器嵌套类型名 `IdentityServiceV1User$Gender`（成员 `male`/`female`/`secret`，小写）。
- **设置页**（`settings/oa_settings_page.dart`）：仅主题与语言切换，`BlocBuilder` 接基座 `theme` cubit；无 service 调用。

> 撤回、自动跳过申请人节点等引擎能力详见 [oa-workflow-design.md](./oa-workflow-design.md) §5。

---

## 5. 构建

```bash
# 1. 生成 Dart 客户端
cd backend/api && buf generate --template buf.app.dart.gen.yaml

# 2. 运行（需 Flutter SDK）
cd frontend/mobile && flutter run
```

> 若开发机用 fvm 管理 Flutter 版本，需先 `fvm use <version>` 再执行步骤 2。

---

## 6. 维护提示

- 基座升级（CMS `flutter_app` 更新）不会自动同步到本目录。需定期 diff 源目录与 `frontend/mobile/lib/src/core/`，手动 cherry-pick。
- `package:flutter_app/` 导入前缀保持不变（pubspec `name` 仍为 `flutter_app`），避免大面积重命名。如需改名，全局替换 import。
- `lib/generated/` 与 `lib/generated/intl/`（l10n）均为生成产物，由 buf 模板 / `flutter_intl`（intl_utils）重新生成。
- `oa` feature 的 l10n 键（`oaTask*` / `oaTab*` 等）定义在 `lib/l10n/intl_{zh_CN,en_US}.arb`，`S` 类由 `intl_utils` 生成于 `lib/generated/l10n.dart`。

---

## 7. 已知边界

- 通知仅文本，无富媒体附件。
- **移动端通知无 SSE 实时推送**：`InternalMessagePublisher`（SSE 推送）只在 admin-service 注册且仅对 admin 自身 `SendMessage` 路径生效；core gRPC-only 无 SSE server，app-service 的 `internal_message_service.go` 无 publisher。故工作流审批通知经 core 落库后不触发任何 SSE，移动端仅靠 REST 轮询 `listMyMessages` 延迟可见。机制与跨端影响详见 [oa-workflow-design.md](./oa-workflow-design.md) §5.5 与 §12。
