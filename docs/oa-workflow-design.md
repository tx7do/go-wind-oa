# go-wind-oa · 轻量级工作流审批引擎 — 架构设计文档

> 本文档面向维护者，记录 `go-wind-oa` 后端的架构决策、三 service 分離結構、proto 域分離，以及当前实现的边界与已知约束。读者应已熟悉 Kratos + Wire + Ent 的基本范式。
>
> 本文与代码同步，权威架构来源为代码本身；本文為導覽性說明。

---

## 1. 设计目标与范围

`go-wind-oa` 是一个**轻量级、线性、状态机驱动**的工作流审批引擎，以 `go-wind-cms` 的 core/admin/app 三 service 架構為基座，自包含 `pkg/`（無 cms 模塊依賴）。

**范围内（v1）**
- 流程定义（含节点序列、动态表单 schema）的存储与查询。
- 申请提交 → 首节点任务派生 → 审批/驳回/转办 → 状态机推进或终结。
- “我的待办 / 已办 / 我的申请”三类列表。
- 审批流转的异步通知，經 `internal_message` 站內信通道（落库 + SSE 推送）。

**范围外（v1，刻意未实现）**
- 会签 / 并行分支 / 子流程 —— 当前为严格线性状态机。
- 角色级 / 部门主管级审批人指派 —— `node_config` 仅支持 `approver_type=="USER"` 指定用户。
- 流程定义的启用/禁用独立接口（定义落 `DRAFT`，校验 `ENABLED`；切换接口未提供）。
- 流程定义的版本管理 UI、回滚等。

---

## 2. 三 service 架構

仿 `go-wind-cms` 的 core/admin/app 模式，三 service 各司其職：

### 2.1 core-service（純 gRPC，工作流引擎落點）

`backend/app/core/service/`，`appid = serviceid.CoreService`，consul key `go-wind-oa/core/service`。

- `internal/server/grpc_server.go`：創建 gRPC server，中間件鏈 `logging + ent`（core 是被 admin/app 調用的後端，無 auth/authz）。註冊四個 gRPC 服務：
  - `internalMessageV1.RegisterInternalMessageServiceServer` + Category + Recipient（`internal_message.service.v1`，站內信）
  - `oaV1.RegisterWorkflowServiceServer`（`oa.service.v1`，工作流引擎）
- `internal/data/`：ent 倉庫層。infra 客戶端（`client.NewRedisClient` / `NewEntClient` / `NewDiscovery`）+ 四個 workflow 倉庫 + 三個 internal_message 倉庫。ProviderSet 見 `data/providers/wire_set.go`。
- `internal/data/ent/schema/`：四張 workflow 表 schema + 三張 internal_message 表 schema（見 §3）。
- `internal/service/`：四個 gRPC 服務實現——`WorkflowService`（狀態機驅動，見 §4）+ 三個 `InternalMessageService`（含 `InternalMessagePublisher` SSE 推送）。ProviderSet 見 `service/providers/wire_set.go`。
- 無 HTTP 端點、無 openapi 生成（core 為 gRPC-only）。

### 2.2 admin-service（HTTP 边端，admin 前端轉發）

`backend/app/admin/service/`，`appid = serviceid.AdminService`，consul key `go-wind-oa/admin/service`。

- `internal/server/rest_server.go`：創建 HTTP server，中間件鏈 `logging → auth.Server + authz.Server（白名單匹配）→ entmiddleware.Server()`。**auth 必須在 ent 之前**：`auth.Server` 對非白名單請求注入 `OperatorMetadata`，`entmiddleware.Server` 據此構建 `UserViewer`，`TenantPrivacy` 策略才生效；順序顛倒則 ent 兜底 `SystemViewer`，租戶隔離失效。
  - 註冊五個 HTTP 服務（`adminV1.Register*HTTPServer`）：
    - `AuthenticationService`（鑑權轉發，`admin.service.v1.i_authentication.proto`）
    - `InternalMessageService` / Category / Recipient（站內信轉發）
    - `WorkflowService`（工作流轉發，`admin.service.v1.i_workflow.proto`）
  - 白名單：`Login` / `GenerateCaptcha` / `VerifyCaptcha` 經 `rpc.AddWhiteList` 放行（雞生蛋：獲取驗證碼時無 token）。
- `internal/data/`：data 層持 gRPC 客戶端打 core-service（`NewAuthenticationServiceClient` / `NewInternalMessage*ServiceClient` / `NewWorkflowServiceClient`），經服務發現定位 `CoreService`。infra：`NewRedisClient` / `NewCaptcha` / `NewDiscovery` / `NewClientType` / `NewAuthorizer` / `NewTranslator` + `auth.NewTokenChecker`。ProviderSet 見 `data/providers/wire_set.go`。
- `internal/service/`：五個轉發層 service（各方法為 HTTP 請求 → gRPC 調 core）。ProviderSet 見 `service/providers/wire_set.go`。
- `cmd/server/assets/`：`openapi.yaml` 由 `buf.admin.openapi.gen.yaml` 生成，`assets.go` embed 供 swagger UI。

### 2.3 app-service（HTTP 边端，移动端轉發）

`backend/app/app/service/`，`appid = serviceid.AppService`，consul key `go-wind-oa/app/service`。

- `internal/server/rest_server.go`：同 admin 中間件鏈與白名單模式。
  - 註冊兩個 HTTP 服務（`appV1.Register*HTTPServer`）：
    - `AuthenticationService`（鑑權轉發）
    - `WorkflowService`（工作流轉發，4 RPC：SubmitApply / AuditTask / GetMyTasks / GetTask）
  - 白名單：僅 `OperationAuthenticationServiceLogin`。
- `internal/data/`：data 層持 `NewAuthenticationServiceClient` + `NewWorkflowServiceClient`（打 core-service）。infra 無 `NewCaptcha` / `NewTranslator`（app 端不用）。ProviderSet 見 `data/providers/wire_set.go`。
- `internal/service/`：兩個轉發層 service。ProviderSet 見 `service/providers/wire_set.go`。
- `cmd/server/assets/`：`openapi.yaml` 由 `buf.app.openapi.gen.yaml` 生成。

### 2.4 sse_server.go

`app/app/service/internal/server/sse_server.go` 持 `AuthenticationServiceClient`，其 `WithAuthorizeFunc` 調 `ValidateToken` 驗 SSE 訂閱的 access token（admin-service 無此端點，因 admin 前端走輪詢不走 SSE）。

---

## 3. proto 域分離

`backend/api/protos/` 下六個保留域：

| 域 | 包名 | 內容 | 性質 |
|---|---|---|---|
| `oa/service/v1/` | `oa.service.v1` | `workflow.proto` + `oa_error.proto` | core 純 gRPC，**無 http annotation** |
| `admin/service/v1/` | `admin.service.v1` | `i_authentication.proto` + `i_internal_message*.proto` + `i_workflow.proto` + `admin_doc.proto` + `admin_error.proto` | HTTP wrapper，引用 oa.service.v1 / internal_message.service.v1 / authentication.service.v1 消息類型 |
| `app/service/v1/` | `app.service.v1` | `i_authentication.proto` + `i_workflow.proto` + `app_doc.proto` + `app_error.proto` | HTTP wrapper，引用 oa.service.v1 / authentication.service.v1 消息類型 |
| `internal_message/service/v1/` | `internal_message.service.v1` | 4 檔（cms 原樣保留） | 站內信消息類型，core 註冊 gRPC |
| `authentication/service/v1/` | `authentication.service.v1` | 9 檔（cms 原樣保留） | 鑑權消息類型，admin/app wrapper 引用 |
| `identity/service/v1/` | `identity.service.v1` | `user.proto` + `types.proto`（cms 原樣保留） | authentication 的傳遞閉包依賴 |

**核心分離原則**：core 的 `oa/service/v1/workflow.proto` 已剝離所有 `google.api.http` annotation，為純 gRPC。HTTP 路由注解定義在 `admin/service/v1/i_workflow.proto` 與 `app/service/v1/i_workflow.proto` wrapper proto，這兩者 `import "oa/service/v1/workflow.proto"` 僅引用消息類型，自身定義帶 http annotation 的 service。鑑權同理：`i_authentication.proto` 引用 cms `authentication.service.v1` 消息類型。

---

## 4. 数据模型（Ent Schema）

四張 workflow 表，均通过 `mixin.TenantID[uint32]{}` 注入 `tenant_id` 列并附加 `rule.TenantPrivacy` 策略。该策略由 `entmiddleware.Server` 注入的 `UserViewer.TenantID()` 驱动，自动在所有查询/写入上叠加 `tenant_id = viewer.tenant` 谓词 —— **代码层无需手写 tenant 过滤**。

| 表 | 说明 | 关键字段 | Mixin 组成 |
|---|---|---|---|
| `WorkflowDefinition` | 流程模板：有序节点配置 + 表单 schema | `node_config`(any), `form_schema`(any), `code`, `version`, `definition_status` | AutoIncrementId / TimeAt / OperatorID / TenantID / Remark |
| `WorkflowInstance` | 一次申请实例 | `form_data`(any), `instance_status`, `current_node_index` | 同上 |
| `WorkflowTask` | 节点上对指派审批人产生的待办 | `node_index`, `assignee_user_id`, `task_status` | AutoIncrementId / TimeAt / OperatorID / TenantID |
| `WorkflowLog` | append-only 审计日志 | `node_index`, `log_action`, `comment` | 同上 |

此外 core-service `ent/schema/` 还含三張 cms 保留的 internal_message schema（`internal_message` / `internal_message_category` / `internal_message_recipient`），供 `InternalMessageService` 落庫。

**`field.Any` 的选择**：`node_config` / `form_schema` / `form_data` 结构动态，采用 ent `field.Any(name)`，DB 以 JSON 落盘、Go 侧 `any`。仓库层在 `Create` 时 `json.Unmarshal` 文本→any、在定向查询时 `json.Marshal` any→文本，显式转换，不依赖 mapper。

**外键与级联**：`Definition 1—N Instance`、`Instance 1—N Task`、`Instance 1—N Log` 均以 `edge.To(...).Required().Annotations(entsql.Annotation{OnDelete: entsql.DeleteCascade})` 声明，父行删除时级联清子行。

**索引**：每张表都有 `idx_*_tenant`；`WorkflowDefinition` 另有 `(tenant_id, code, version)` 唯一索引；`WorkflowTask` / `WorkflowLog` / `WorkflowInstance` 各有按 `(tenant_id, assignee_user_id, task_status)` / `(tenant_id, created_by)` 的检索索引，对应待办/已办/我的申请三类视图。

---

## 5. 状态机与服务层（core `workflow_service.go`）

`WorkflowService` 实现 kratos 生成的 `WorkflowServiceServer`（gRPC，`oa.service.v1`），注入四个仓库 + `*InternalMessageService`（同進程直接調用，非跨進程 gRPC 客戶端）。状态机为**线性、单实例单活跃任务**模型：

```
SubmitApply ──> Instance(PENDING, idx=0) + Task(node=0, assignee=A0, PENDING) + Log(SUBMIT) + notify(A0)
   │
   ▼ AuditTask(APPROVE)  ← 仅当 task.assignee==caller 且 task.PENDING 且 instance.PENDING
   │
   ├─ idx+1 < len(nodes): 关闭 Task → Instance(idx=idx+1, PENDING) + 新 Task(node=idx+1, assignee=A_{idx+1}, PENDING) + Log(APPROVE) + notify(A_{idx+1})
   │
   └─ idx+1 >= len(nodes): 关闭 Task → Instance(APPROVED, idx=nil) + Log(APPROVE) + notify(applicant)   [终结]

AuditTask(REJECT)  → 关闭 Task(REJECTED) → Instance(REJECTED, idx=nil) + Log(REJECT) + notify(applicant)   [终结]

AuditTask(FORWARD) → Task.assignee ← forwardTo（状态保持 PENDING，idx 不变）+ Log(FORWARD) + notify(forwardTo)
```

**关键不变量与校验**：
- 任务关闭与实例状态推进在 service 层成对发生。
- `GetState` 直读 entity 的 status/指针，绕过 mapper，避免 `field.Any` 的不确定指针行为影响状态判断。
- `current_node_index` 仅在 `instance_status==PENDING` 时有意义；终结态写 `nil` 清空。
- `callerFromContext` 从 viewer context 取 `(tenantID, userID)`，二者任一为 0 即 fail-closed。
- `AuditTask` 强校验 `task.assignee == caller` 且 `task.PENDING`，否则 `ErrorForbidden`。
- 申请表单数据 `form_data` 仅在 `SubmitApply` 时透传落盘，后续审批流程不读不写。

**异步通知**：`notifyAsync` 用 `context.WithoutCancel(ctx)` + 5s 超时 + `recover`，fire-and-forget 調用同進程 `InternalMessageService` 的 `SendMessage`（`Type=NOTIFICATION`、`RecipientUserId`/`TargetUserIds` 指向 recipient）。通知落 `internal_message_recipient` 表，並由 `InternalMessagePublisher` SSE 推送給在線客戶端。通知失败不回滚状态机。

---

## 6. 代码生成

每個 service 目錄的 `Makefile` 為 `include ../../../app.mk`，`SERVICE_NAME` 決定 buf openapi 模板選擇（core 跳過）。

| 目标 | 命令 | 产物 |
|---|---|---|
| ent ORM | `make ent`（各 service）| `internal/data/ent/` 下全套生成代码 |
| proto 桩（Go）| `make api`（`cd ../../../api && buf generate`）| `api/gen/go/{oa,internal_message,authentication,identity,admin,app}/service/v1/*.pb.go` + `*_grpc.pb.go` + `*_errors.pb.go` + `*.pb.validate.go` |
| openapi v3 | `make openapi`（admin/app，core 跳過）| `app/{admin,app}/service/cmd/server/assets/openapi.yaml` |
| wire DI | `make wire`（各 service）| `cmd/server/wire_gen.go` |

### 6.1 buf 模板

`backend/api/` 下五個 buf 模板：

| 模板 | 生成器 | inputs | out |
|---|---|---|---|
| `buf.gen.yaml` | protoc-gen-go*（Go 桩）| 6 個保留域（`inputs.paths` 過濾）| `api/gen/go/`（per-domain `go_package` override 全指 `go-wind-oa/api/gen/go/...`）|
| `buf.admin.openapi.gen.yaml` | protoc-gen-openapi | `protos/admin/service/v1` | `app/admin/service/cmd/server/assets/` |
| `buf.app.openapi.gen.yaml` | protoc-gen-openapi | `protos/app/service/v1` | `app/app/service/cmd/server/assets/` |
| `buf.admin.typescript.gen.yaml` | protoc-gen-typescript-http | `protos/admin/service/v1` | `frontend/admin/src/api/generated/admin/service/v1/` |
| `buf.flutter.oa.dart.gen.yaml` | protoc-gen-dart-http | `protos/app/service/v1` | `frontend/mobile/lib/generated/api/app/service/v1/` |

`buf.yaml`（v2，`modules.path: protos`）deps 含 googleapis / kratos / gnostic / pagination / protoc-gen-validate / redact 六個 remote。首次拉取須 `buf dep update` 生成 `buf.lock`。

### 6.2 ent feature 标志

`make ent` 的五個 feature（`privacy` / `entql` / `sql/modifier` / `sql/upsert` / `sql/lock`）。**`privacy` 不可省略** —— 它是 `TenantID` mixin 附着的 `rule.TenantPrivacy` 策略生效的前提。

### 6.3 wire

`wire_gen.go` 具現化依賴圖：server ProviderSet + service ProviderSet + data ProviderSet + `newApp`。`wire.go` 攜帶 `//go:build wireinject` 標籤，正常構建時由 `wire_gen.go` 提供實體。

---

## 7. 前端生成链

### 7.1 Admin TS 客户端

`buf.admin.typescript.gen.yaml` 生成 `frontend/admin/src/api/generated/admin/service/v1/index.ts`，含 `ApiClient.workflowService` / `authenticationService` / `internalMessageService` 等。Composables（`src/api/composables/{oa,auth}.ts`）封裝為 Vue Query hooks。類型名帶包前綴（`oaservicev1_*` / `authenticationservicev1_*`）。

### 7.2 Mobile Dart 客户端

`buf.flutter.oa.dart.gen.yaml` 生成 `frontend/mobile/lib/generated/api/app/service/v1/index.dart`，含 `ApiClient.workflowService` / `authenticationService`。类型名带包前缀（`OaServiceV1*`）。枚举成员为小写（`pending` / `submitted` / `approve` / `reject` / `forward`）。

> 注：mobile 的 Dart 客户端由 buf 模板直接生成，无 swagger_parser 中间层。移动端架构细节见 `docs/oa-mobile-design.md`。

---

## 8. 已知边界与后续工作

1. **节点指派仅支持指定用户**。`resolveApprover` 拒绝非 `USER` 类型。扩展为角色/部门主管指派需：(a) `node_config` 增 `approver_role` / `approver_dept` 字段；(b) service 层調 `identity` 服務解析具体用户ID。
2. **定义启用/禁用接口未提供**。`CreateWorkflowDefinition` 一律落 `DRAFT`，`SubmitApply` 校验 `ENABLED`。需补 `UpdateWorkflowDefinition`（带 `update_mask` 限定 `definition_status`）或专用启用接口。
3. **无定时任务 / 超时处理**。长期 PENDING 任务不会自动催办或超时终结。
4. **无会签 / 并行 / 回退**。线性状态机的硬约束。
5. **通知仅文本**。`notifyAsync` 的 title/content 为固定文案。
