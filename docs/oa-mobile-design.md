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
│       ├── services/          # 各业务服务（workflow / attendance / leave / expense / file_upload / notification）
│       └── pages/             # 登录、审批任务列表与详情、提交申请、考勤打卡、请假、报销、通知
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

**生成产物**：`lib/generated/api/app/service/v1/index.dart`，含聚合 `ApiClient`，按服务名暴露懒加载属性：

| 属性 | 服务 | 方法 |
|------|------|------|
| `apiClient.workflowService` | 工作流 | SubmitApply / AuditTask / GetMyTasks / GetTask / WithdrawApply |
| `apiClient.authenticationService` | 鉴权 | Login / Logout / RefreshToken / GenerateCaptcha / VerifyCaptcha |

**关键约定**：

- `buf.app.dart.gen.yaml` 的 `inputs` 仅覆盖 `protos/app/service/v1`（即 `i_authentication.proto` + `i_workflow.proto` wrapper proto）。这些 wrapper 的 `google.api.http` 注解定义了 `/app/v1/oa/workflow/...` 与 `/app/v1/login` 等路径，Dart 生成器据此产生带路径的 client 方法。消息类型引用 `oa.service.v1` / `authentication.service.v1`，生成器自动跟随 import 解析。
- 生成的客户端经 `lib/src/core/transport/http/dio_client_transport.dart` 适配为 `ClientTransport`，复用基座的 Dio + 鉴权拦截器（自动注入 `Authorization: Bearer <token>`）。
- 类型名带包前缀（`OaServiceV1*`），枚举成员为小写（`pending` / `submitted` / `approve` / `reject` / `forward`），与生成器的命名约定一致。

> 与旧链路对比：旧设计（已废弃）经 protoc-gen-openapi 生成 openapi.yaml，再由 swagger_parser 消费生成 Dart 客户端。现改为 buf 模板直接生成，无中间层，路径为 `app/service/v1`（非 `oa/v1`）。

---

## 3. 路由与 Shell

移动端路由定义在 `lib/src/app_router/app_router.dart`，路由路径常量在 `lib/src/core/constants/router_paths.dart`。

- **ShellRoute**（`OaShellPage`，`lib/src/features/oa/pages/shell/oa_shell_page.dart`）：底部导航三 Tab——`/oa/tasks`（审批）/ `/oa/notifications`（通知）/ `/oa/attendance`（考勤）。`OaShellPage` 为 `StatelessWidget`，据 `currentRoute` 高亮 Tab，`_onTap` 点击切路由；不持有业务状态。
- **子路由**：`/oa/tasks/detail/:id`（任务详情），挂在 `/oa/tasks` 下，进同一 Shell（高亮审批 Tab）。
- **非 Shell 路由**：`/oa/apply`（提交申请，全屏表单）、`/login`（登录）。
- **守卫 `_guard`**：依 `GetIt.instance<UserAuthCache>().hasLogin` 判定；未登录访问任意路由 → `/login`；已登录访问 `/login` → `/`（审批 Tab）。

---

## 4. 功能实现状态

### 4.1 审批工作流 ✅

- `WorkflowService`（`services/workflow_service.dart`）：extends `BaseService`，持有生成的 `apiClient.workflowService`；提供 `pendingTasks` / `submittedTasks`（列表直接调用，返回 `List<OaServiceV1MyTaskItem>`，从 response 的 `items` 字段提取）+ `audit`（审批/驳回/转交）+ `submitApply`（提交申请）+ `withdraw`（撤回）。`pendingTasksQuery` / `submittedTasksQuery` 为 cached_query 的 `Query` 包装。
- 列表页（`task_list/oa_task_list_page.dart`）：两 Tab，各 `FutureBuilder` 驱动 `ListView`，行展示 title/status_label/occurred_at，点行进详情，FAB 进提交申请。
- 详情页（`task_detail/oa_task_detail_page.dart`）：审批按钮（同意/驳回/转交/撤回），转交弹 dialog 收 forwardToUserId，调 `audit`。`AuditLogEntry.occurredAt` 为 `String?`（ISO 时间戳文本），直接渲染。
- 提交申请页（`submit_apply/oa_submit_apply_page.dart`）：表单收 definition_code/version/title/form_data，调 `submitApply`。

### 4.2 站内信通知 ✅

- 通知列表页从后端 `GET /app/v1/internal-message/my-messages` 拉取收件箱（core `ListMyMessages`，收件人过滤、排除已删除/已撤销）。
- SSE 实时推送：app-service 的 `sse_server.go` 持 `AuthenticationServiceClient` 验 token 后允许订阅，事件投递到通知 stream。

### 4.3 考勤打卡 ✅

- `AttendanceService`（`services/attendance_service.dart`）：调用后端 `POST /app/v1/oa/attendance/check-in`，提交 GPS 经纬度与 WiFi BSSID（`geolocator` + `wifi_iot`），后端落库并按工时设置结算。
- 打卡页（`attendance/oa_attendance_page.dart`）：当日首次=签到、第二次=签退；409 已签退。

### 4.4 请假 ✅

- `LeaveService`：提交请假申请（类型/额度校验后建单，进程内直调引擎 SubmitApply 挂 LEAVE v1 流程）。半日粒度 start_half/end_half。
- 移动端请假页收集类型/起止时间/半日/事由，调服务提交。

### 4.5 报销 ✅

- `ExpenseService`：提交报销申请（多行明细 + 发票文件 ID）。
- `FileUploadService`：multipart `POST /app/v1/file/upload` 拍照/相册（image_picker）→ 压缩 → Dio multipart → 返回 file_id 自动回填明细 invoiceFileId。
- 报销页收集明细行与发票附件，调服务提交，挂 EXPENSE v1 流程。

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

- 列表接口不分页（items+total 直返）。
- 通知仅文本，无富媒体附件。
