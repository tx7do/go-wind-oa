# go-wind-oa 移动端架构设计文档

> 本文档面向维护者，记录 OA 移动端（Flutter）的架构决策、API 生成链、路由结构、四功能实现状态与后端待办。读者应已熟悉 Flutter + go_router + cached_query + dio 的基本范式。
>
> 本文与代码同步。后端架构见 `docs/oa-workflow-design.md`。

---

## 1. 仓结构

OA 移动端代码落在 `go-wind-oa/frontend/mobile/`。包名為 `flutter_app`（pubspec `name`），故所有內部 import 前綴為 `package:flutter_app/...`。

基座代碼（`lib/src/core/`：transport / BaseService / UserAuthCache / 鑑權攔截器 / 路由守衛基礎設施）隨倉初始化時拷入，與 cms 無 git 關聯，後續基座升級需手動 diff 同步。

---

## 2. API 代码生成链

OA 移動端的 Dart API 客戶端由 **buf.flutter.oa.dart.gen.yaml** 直接生成，无 swagger_parser / openapi 中间层：

```
protos/app/service/v1/*.proto
        │  (buf.flutter.oa.dart.gen.yaml + protoc-gen-dart-http)
        ▼
frontend/mobile/lib/generated/api/app/service/v1/index.dart
```

**触发命令**：
```bash
cd backend/api && buf generate --template buf.flutter.oa.dart.gen.yaml
```

**生成产物**：`lib/generated/api/app/service/v1/index.dart`，含聚合 `ApiClient`，按服務名暴露懶加載屬性：
- `apiClient.workflowService` —— OA 工作流（SubmitApply / AuditTask / GetMyTasks / GetTask）
- `apiClient.authenticationService` —— 鑑權（Login / Logout / RefreshToken / GenerateCaptcha / VerifyCaptcha）

**关键约定**：
- `buf.flutter.oa.dart.gen.yaml` 的 `inputs` 僅覆蓋 `protos/app/service/v1`（即 `i_authentication.proto` + `i_workflow.proto` wrapper proto）。這些 wrapper 的 `google.api.http` 注解定義了 `/app/v1/oa/workflow/...` 與 `/app/v1/login` 等路徑， Dart 生成器據此產生帶路徑的 client 方法。消息類型引用 `oa.service.v1` / `authentication.service.v1`，生成器自動跟隨 import 解析。
- 生成的客户端经 `lib/src/core/transport/http/dio_client_transport.dart` 适配为 `ClientTransport`，复用基座的 dio + 鉴权拦截器（自动注入 `Authorization: Bearer <token>`）。
- 類型名帶包前綴（`OaServiceV1*`），枚舉成員為小寫（`pending` / `submitted` / `approve` / `reject` / `forward`），與生成器的命名約定一致。

> 與舊鏈路對比：舊設計（已廢棄）經 protoc-gen-openapi 生成 openapi.yaml，再由 swagger_parser 消費生成 Dart 客戶端。現改為 buf 模板直接生成，無中間層，路徑為 `app/service/v1`（非 `oa/v1`）。

---

## 3. 路由与 Shell

OA 移动端路由定義在 `lib/src/app_router/app_router.dart`，路由路徑常量在 `lib/src/core/constants/router_paths.dart`。

- **ShellRoute**（`OaShellPage`，`lib/src/features/oa/pages/shell/oa_shell_page.dart`）：底部導航三 Tab——`/oa/tasks`（工作流）/ `/oa/notifications`（通知）/ `/oa/attendance`（考勤）。`OaShellPage` 為 `StatelessWidget`，據 `currentRoute` 高亮 Tab，`_onTap(BuildContext, int)` 點擊切路由；不持有业务状态。
- **子路由**：`/oa/tasks/detail/:id`（任務詳情），掛在 `/oa/tasks` 下，進同一 Shell（高亮工作流 Tab）。
- **非 Shell 路由**：`/oa/apply`（提交申請，全屏表單）、`/login`（登錄）。
- **守衛 `_guard`**：依 `GetIt.instance<UserAuthCache>().hasLogin` 判定；未登錄訪問任意路由 → `/login`；已登錄訪問 `/login` → `/`（工作流 Tab）。

---

## 4. 四功能实现状态

### 4.1 工作流审批 ✅ 完整
- `WorkflowService`（`services/workflow_service.dart`）：extends `BaseService`，持有生成的 `apiClient.workflowService`；提供 `pendingTasks`/`submittedTasks`（列表直接調用，返回 `List<OaServiceV1MyTaskItem>`，從 response 的 `items` 字段提取）+ `audit`（審批/駁回/轉交）+ `submitApply`（提交申請）。`pendingTasksQuery` / `submittedTasksQuery` 為 cached_query 的 `Query` 包裝。
- 列表页（`task_list/oa_task_list_page.dart`）：两 Tab，各 `Future` 驱动 `ListView`，行展示 title/status_label/occurred_at，点行进详情，FAB 进提交申请。
- 详情页（`task_detail/oa_task_detail_page.dart`）：审批按钮三枚（同意/驳回/转交），转交弹 dialog 收 forwardToUserId，调 `audit`。`AuditLogEntry.occurredAt` 為 `String?`（ISO 時間戳文本），直接渲染。
- 提交申请页（`submit_apply/oa_submit_apply_page.dart`）：表单收 definition_code/version/title/form_data，调 `submitApply`。
- 审批完成后调 `Navigator.maybePop` 返回列表；列表 `initState` 重新拉取，故体现"少一笔"。

### 4.2 即時消息推送 🟡 骨架
- `NotificationService`（`services/notification_service.dart`）：`notificationStream` 為空 Stream（永不产出）。提供 `static instance` 單例供 UI 訪問。
- 页面（`notifications/oa_notifications_page.dart`）：`StreamBuilder` 监听该 stream，无数据时显示"推送未配置"占位。
- `pubspec.yaml` 未启用 `firebase_messaging` / `jpush_flutter`（待后端通道定稿）。

### 4.3 移動考勤與打卡 🟡 骨架
- `AttendanceService`：`checkIn` 永远返回 `notConfigured`，`isInFence` 永远 false。
- 页面：打卡按钮调 `checkIn`，展示"考勤服务未配置"占位。
- `pubspec.yaml` 未启用 `geolocator` / `wifi_iot`（待后端落地）。

### 4.4 運維微控制台 ❌ 剔除
经用户确认无"云手机"功能，本字段剔除。

---

## 5. 后端待办

### 5.1 推送通道（阻塞 4.2）
**现状**：`NotificationService.notificationStream` 为空 Stream。

**落地**：
- 后端：`internal_message` 的 SSE 通道（app-service 的 `sse_server.go` 持 `AuthenticationServiceClient` 驗 token 後允許訂閱）向 OA 移動端推送 notification 事件。
- 前端：新增 SSE 客戶端訂閱 app-service 的 SSE 端點，事件投遞到 `notificationStream`。
- FCM/JPush：需服务端 push token 注册接口 + 设备令牌与用户绑定。

### 5.2 考勤服务（阻塞 4.3）
**现状**：`AttendanceService.checkIn` 永远 `notConfigured`。

**落地**：
- 后端：考勤服务接口（打卡记录、班次规则）。
- 地理围栏库：公司方圆 N 米的多边形/半径定义。
- 公司 Wi-Fi 指纹库：SSID/BSSID 白名单。
- 前端：启用 `geolocator`（GPS）+ `wifi_iot`（SSID 读取），判定在围栏/白名单内才允许打卡。

### 5.3 定义启用/禁用接口（影响 admin 前端完整性）
**现状**：OA 后端 `CreateWorkflowDefinition` 一律落 `DRAFT`，`SubmitApply` 校验 `ENABLED`，但无切换接口。admin 前端列表仅展示状态 Tag，无法切换。

**落地**：补 `UpdateWorkflowDefinition`（带 `update_mask` 限定 `definition_status`）或专用启用接口。

---

## 6. 构建

```bash
# 1. 生成 Dart 客户端
cd backend/api && buf generate --template buf.flutter.oa.dart.gen.yaml

# 2. 运行（需 Flutter SDK）
cd frontend/mobile && flutter run
```

**注意**：当前开发机未安装 Flutter SDK（仅 fvm 壳）。用户本地 `fvm use <version>` 后执行步骤 2。

---

## 7. 维护提示

- 基座升级（cms flutter_app 更新）不会自动同步到本目录。需定期 diff 源目录与 `frontend/mobile/lib/src/core/`，手动 cherry-pick。
- `package:flutter_app/` 导入前缀保持不变（pubspec `name` 仍为 `flutter_app`），避免大面积重命名。如需改名，全局替换 import。
- `lib/generated/` 与 `lib/generated/intl/`（l10n）均为生成产物，由 buf 模板 / `flutter_intl`（intl_utils）重新生成。
- `oa` feature 的 l10n 鍵（`oaTask*` / `oaTab*` 等）定義在 `lib/l10n/intl_{zh_CN,en_US}.arb`，`S` 類由 `intl_utils` 生成於 `lib/generated/l10n.dart`。
