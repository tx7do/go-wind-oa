# go-wind-oa · 轻量级工作流审批引擎 — 架构设计文档

> 本文档面向维护者，记录 `go-wind-oa` 后端的架构决策、三 service 分離結構、proto 域分離，以及当前实现的边界与已知约束。读者应已熟悉 Kratos + Wire + Ent 的基本范式。
>
> 本文与代码同步，权威架构来源为代码本身；本文為導覽性說明。
>
> **⚠ 2026-08-20 勘误**：§1「范围外」中的会签/或签、主管级审批人指派、定义启用切换接口，
> 以及 §5 描述的「仅 USER 单审批人线性状态机」均**已实现**——见文末 §9「实现状态（2026-08-20）」。

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

- `internal/server/grpc_server.go`：創建 gRPC server，中間件鏈 `logging + ent`（core 是被 admin/app 調用的後端，自身不做請求級 auth/authz）。註冊六個 gRPC 服務：
  - `internalMessageV1.RegisterInternalMessageServiceServer` + Category + Recipient（`internal_message.service.v1`，站內信）
  - `oaV1.RegisterWorkflowServiceServer`（`oa.service.v1`，工作流引擎）
  - `oaV1.RegisterAttendanceServiceServer`（`oa.service.v1`，考勤打卡與圍欄/Wi-Fi 庫）
  - `authenticationV1.RegisterAuthenticationServiceServer`（`authentication.service.v1`，鑑權服務端，admin/app BFF 經服務發現調用）
- `internal/data/`：ent 倉庫層。infra 客戶端（`client.NewRedisClient` / `NewEntClient` / `NewDiscovery`）+ 四個 workflow 倉庫 + 三個 internal_message 倉庫 + 三個 attendance 倉庫 + 鑑權件（`Authenticator` JWT 簽發/驗證、`UserTokenCache` Redis 令牌存儲、`UserRepo` / `UserCredentialRepo`）。ProviderSet 見 `data/providers/wire_set.go`。
- `internal/data/ent/schema/`：四張 workflow 表 + 三張 internal_message 表 + 三張 attendance 表 + 兩張鑑權表（`sys_users` / `sys_user_credentials`）。注意 workflow 三張父子表的 O2M 邊（definition→instances、instance→tasks/logs）**不可加 Required()**——ent 語義為「建父記錄時必須已存在子記錄」，會導致定義/實例無法插入。
- `internal/service/`：六個 gRPC 服務實現——`WorkflowService`（狀態機驅動，見 §4）+ 三個 `InternalMessageService`（含 `InternalMessagePublisher` SSE 推送）+ `AttendanceService` + `AuthenticationService`。ProviderSet 見 `service/providers/wire_set.go`。
- 無 HTTP 端點、無 openapi 生成（core 為 gRPC-only）。

#### 鑑權服務端（AuthenticationService，以 cms 為基座裁剪）

- 令牌機制完整保留自 cms：JWT access token（admin/app 兩套獨立 key，`configs/authenticator.yaml`）+ 不透明 refresh token，Redis String key 存儲（前綴 `goa:`），刷新輪換經 Lua 腳本原子「驗 RT→刪 RT→刪 AT」，支持黑名單（`bl:{jti}`）與按 jti/用戶撤銷。
- 登錄鏈路：password grant → `FindUserCredential`（AES 解密 → bcrypt 校驗，恆定時間防用戶名枚舉）→ 用戶狀態檢查 → 簽發令牌對。密碼傳輸格式與雙端前端共享 `crypto.DefaultAESKey`（AES-CBC，key=IV，PKCS7，base64）。
- 裁剪掉的 cms 能力：RBAC 鏈（角色→權限→`sys:access_backend`）、租戶解析（OA v1 單租戶 tenant 0，`tenant_code` 忽略）、`RegisterUser` / `WhoAmI`（雙端 BFF 未暴露對應端點，落 Unimplemented）。
- 種子數據：服務啟動時 `sys_users` 為空則建立初始管理員 `admin/admin`（**tenant 1 默認租戶**，bcrypt 入庫）。不可用 tenant 0（平台域）：OA 各服務的 callerFromContext 對 tid==0 一律 fail-closed 拒絕，種子必須落在真實租戶上。
- 令牌角色：v1 無角色表，簽發時為 payload 填入 `platform:admin` 角色碼——admin BFF 的 authz 中間件對空 subject 直接 403（noop 引擎僅在有 subject 時放行）。identity/permission 域落地後改為真實角色。
- admin 前端的用戶信息從本地 access token JWT payload（`uid`/`sub`/`tid`/`roc`）解出，僅作展示；OA 路由不設 `meta.authority`（後端 authz 為 noop），待 identity/permission 域落地後恢復。

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


---

## 9. 实现状态（2026-08-20，实测通过）

> 本章为当前权威状态，与 §1-§8 中过时陈述冲突时以本章为准。全部能力经本地端到端冒烟实证（非仅编译级）。

### 9.1 引擎能力（超出 v1 范围部分）

- **多审批人与会签/或签**：`node_config` 新格式 `{"approvers":[{"type":"USER"|"LEADER"|"POSITION","id":N}],"strategy":"ALL"|"ANY"}`（旧单人格式自动归一化兼容）。ALL=会签（全员通过才推进，任一驳回即驳回）；ANY=或签（一人通过即推进并取消其余，全员驳回才驳回）。每审批人一条并行 task。
- **审批人类型**：`USER` 显式指定；`LEADER`=申请人主组织单元负责人（user_org_unit → org_unit.leader_id）；`POSITION`=职位在职持有者（user_position，可展开多人）。寻人以申请人为基准，解析失败即提交失败（不挂起）。
- **申请人为审批人自动跳过**：`launchFromNode` 统一入口——解析后剔除申请人（对自己视为自动同意），整节点全为申请人则写「自动通过」APPROVE 日志后跳过该节点继续推进，可连续跳多节点直至越界终结。
- **撤回**：`WithdrawApply`（app 端 `POST /app/v1/oa/workflow/withdraw-apply`）。仅申请人本人 + 实例 PENDING；实例→WITHDRAWN、全部待办任务→CANCELLED、写 WITHDRAW 日志、通知原审批人。
- **业务挂鉤**：实例携带 `business_type`/`business_id`；`WorkflowEventRegistry`（进程内 map）在三个终态（APPROVED/REJECTED/WITHDRAWN）同步回调业务模块。回调须校验单据.instance_id 关联（防伪造）且仅处理 PENDING 单据（幂等）。
- **通知修复**：notifyManyAsync 用 `context.WithoutCancel(ctx)` 保留 viewer（SendMessage 从 viewer 推导发送者，防伪造），脱离已返回的 gRPC 请求生命周期。
- **枚举扩展**：InstanceStatus+WITHDRAWN、TaskStatus+CANCELLED、LogAction+WITHDRAW；MyTaskItem 补 task_id（待办列表填充，供进详情）；GetTaskResponse 补 form_data（审批人可见申请内容）。

### 9.2 请假（oa_leave_type / oa_leave_balance / oa_leave_application）

- 类型（租户内 code 唯一）+ 额度（用户×类型×年度，total/used 支持半日 0.5 步进）+ 申请单。
- 提交：校验额度 → 建单 → 进程内直调引擎 SubmitApply（code=LEAVE v1）。**流程定义自动引导**：租户内无 LEAVE 定义时按默认模板（提交给申请人主管 LEADER，会签）自动创建并启用——开箱即用。
- 半日粒度：start_half（默认 AM）/end_half（默认 PM），`computeLeaveDays` 半日算天（同日 PM起+AM止非法）。请求时间戳统一 `.In(time.Local)` 后截断（Timestamp.AsTime 恒 UTC 位置的坑）。
- 审批通过回调：单据 APPROVED + `AddUsedDays` 扣额度；驳回/撤回仅同步状态。
- 姓名回填：applicant_name（resolver 批量查 user.username）。

### 9.3 报销（oa_expense_application / oa_expense_item）

- 申请单 + 多行明细（类别/金额/日期/说明/**发票文件 ID**）。O2M 边 Cascade；**不可加 Required()**（阻断父记录创建，同 workflow 边教训）。
- 明细 List 需 WithItems 预载。提交挂 EXPENSE v1 流程（同样自动引导）。
- **发票直传链路**：multipart POST /app/v1/file/upload（流式 reader 经 context 注入 minio）→ UploadFileResponse 返回 **file_id**（落库文件记录 ID）→ 明细 invoiceFileId 引用。移动端拍照/相册（image_picker）→ 压缩 → Dio multipart → 自动回填。
- multipart 三要点（修复记录）：请求头补 Accept: application/json（kratos 响应编码按 Accept 回退 Content-Type）；oneof Source 打 File 标记（字节走 ctx reader 不走 proto 字段）；StorageObject 缺省空对象（自动桶名/UUID 对象名前提）。app BFF 曾双注册（生成版遮蔽手改流式版）已修。

### 9.4 考勤（oa_attendance_record / oa_attendance_setting / oa_holiday）

- 打卡：当日首次=签到、第二次=签退并结算（GPS 经纬度 + WiFi BSSID 全程落库）。409 已签退。
- 结算：请假覆盖优先 ON_LEAVE；否则迟到（签到>上班时间）/早退（签退<下班时间）/正常。工时设置每租户一行（默认 09:00-18:00，admin 可改）。
- **节假日表**：HOLIDAY=法定假日休息（可落工作日）、WORKDAY=调休上班（可落周末），优先于周末判定；未设置按周六日。管理员按日设置（存在则覆盖）。
  - 休息日（节假日或周末）跳过结算物化；休息日打卡结算为 NORMAL（加班不计迟到）。
  - 定时结算调度器逐租户判定休息日。
- 每日定时结算：AttendanceScheduler（wire 注入常驻 goroutine）每 30 分钟检查，本地 00:30 后为「昨日」跑全租户结算。补结算仅处理仍 PENDING 的记录（幂等）。
- user_name 回填。

### 9.5 鉴权与站内信（CMS 基座随大重构就位）

- core 完整 AuthenticationService（Login/Logout/RefreshToken，密码=base64(AES(明文))，库内 bcrypt）+ RBAC（登录要求用户有角色且角色含 `sys:access_backend` 权限码）。
- admin 登录强制验证码（X-Captcha-Id/Value 头；答案在 Redis `gowind-cms:captcha:<id>`）。
- app BFF 匿名请求按 Host→租户 domain 解析（fail-closed）；**LoginRequest.tenant_code 的 json_name 是蛇形**（传 tenantCode 会被静默忽略）。
- app 端站内信收件箱恢复：core `ListMyMessages`（收件人过滤、排除已删除/已撤销）+ `GET /app/v1/internal-message/my-messages`。
- admin 审批闭环：admin 挂 `AuditTask`（POST /admin/v1/oa/workflow/audit-task）+ 审批中心页（三 Tab + 详情审批/转办）。

### 9.6 单元测试与冒烟

- 纯逻辑单测：`internal/service/oa_logic_test.go`（computeLeaveDays 半日矩阵 / parseHHMM / truncateDate / isWeekend / 节点归一化与策略 / 迟到早退语义）。`go test -vet=off`（包内存量 vet 告警）。
- 冒烟种子工具：`app/core/service/cmd/smokeseed`（租户/角色/权限/组织/双用户/请假类型额度，输出加密登录密码）。操作全手册见项目记忆。

### 9.7 已知边界（现行）

- 无事务多步写入（状态机/建单链）——与既有风险面一致。
- 请假天数按自然日（含周末），未做工作日扣减；半天粒度已支持但额度过期/结转无。
- 转办不校验目标用户存在性；或签部分驳回停留时无提醒。
- 列表接口不分页（items+total 直返）。
- form_schema 存储但无渲染器（表单为各端写死界面）；定义编辑为 JSON textarea+校验（非可视化编辑器）。
