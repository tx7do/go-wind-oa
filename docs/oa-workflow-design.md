# go-wind-oa · 协同办公系统 — 后端架构设计文档

> 本文档面向维护者，记录 `go-wind-oa` 后端的架构决策、三服务分离结构、proto 域分离，以及当前实现的边界与已知约束。读者应已熟悉 Kratos + Wire + Ent 的基本范式。
>
> 本文与代码同步，权威架构来源为代码本身；本文为导览性说明。

---

## 1. 系统定位

`go-wind-oa` 是一套**协同办公系统**，覆盖审批流转、人事考勤、请假与报销等日常办公场景。系统以 `go-wind-cms` 的 core/admin/app 三服务架构为基座，自包含 `pkg/`（无 CMS 模块依赖）。

工作流引擎是协同办公系统的一个**子系统**，作为审批流转的基础能力被请假、报销等业务模块复用，并非系统定位本身。各业务模块通过进程内事件注册表在审批终态回调业务逻辑，引擎本身不感知具体业务语义。

### 1.1 业务模块

| 模块 | 表 | 引擎挂钩 | 状态 |
|------|----|----------|------|
| 工作流引擎 | workflow_definition / workflow_instance / workflow_task / workflow_log | — | 已实现 |
| 请假 | oa_leave_type / oa_leave_balance / oa_leave_application | LEAVE v1 流程，审批通过扣额度 | 已实现 |
| 报销 | oa_expense_application / oa_expense_item | EXPENSE v1 流程 | 已实现 |
| 考勤 | oa_attendance_record / oa_attendance_setting / oa_holiday | — | 已实现 |
| 站内信 | internal_message / internal_message_category / internal_message_recipient | — | 已实现 |

> 工作流引擎的寻人、状态机、事件回调机制是请假与报销审批链的共享底座。

---

## 2. 三服务架构

仿 `go-wind-cms` 的 core/admin/app 模式，三服务各司其职：

### 2.1 core-service（纯 gRPC，业务实现落点）

`backend/app/core/service/`，`appid = serviceid.CoreService`，consul key `go-wind-oa/core/service`。

- `internal/server/grpc_server.go`：创建 gRPC server，中间件链 `logging + ent`（core 是被 admin/app 调用的后端，自身不做请求级 auth/authz）。注册如下 gRPC 服务：
  - `oaV1.RegisterWorkflowServiceServer` / `RegisterAttendanceServiceServer`（`oa.service.v1`）
  - `internalMessageV1.RegisterInternalMessageServiceServer` + Category + Recipient（`internal_message.service.v1`）
  - `authenticationV1.RegisterAuthenticationServiceServer`（`authentication.service.v1`）
- `internal/data/`：ent 仓库层。infra 客户端（`client.NewRedisClient` / `NewEntClient` / `NewDiscovery`）+ 各业务域仓库。ProviderSet 见 `data/providers/wire_set.go`。
- `internal/data/ent/schema/`：协同办公域各业务表。注意 workflow 三张父子表的 O2M 边（definition→instances、instance→tasks/logs）**不可加 Required()**——ent 语义为「建父记录时必须已存在子记录」，会导致定义/实例无法插入。
- `internal/service/`：各 gRPC 服务实现。ProviderSet 见 `service/providers/wire_set.go`。
- 无 HTTP 端点、无 openapi 生成（core 为 gRPC-only）。

#### 鉴权服务端（AuthenticationService，以 CMS 为基座裁剪）

- 令牌机制完整保留自 CMS：JWT access token（admin/app 两套独立 key，`configs/authenticator.yaml`）+ 不透明 refresh token，Redis String key 存储（前缀 `goa:`），刷新轮换经 Lua 脚本原子「验 RT→删 RT→删 AT」，支持黑名单（`bl:{jti}`）与按 jti/用户撤销。
- 登录链路：password grant → `FindUserCredential`（AES 解密 → bcrypt 校验，恒定时间防用户名枚举）→ 用户状态检查 → 签发令牌对。密码传输格式与双端前端共享 `crypto.DefaultAESKey`（AES-CBC，key=IV，PKCS7，base64）。
- RBAC：登录要求用户有角色且角色含 `sys:access_backend` 权限码。identity/permission 域落地后改用真实角色表。
- 种子数据：服务启动时 `sys_users` 为空则建立初始管理员 `admin/admin`（tenant 1 默认租户，bcrypt 入库）。不可用 tenant 0（平台域）：OA 各服务的 callerFromContext 对 tid==0 一律 fail-closed 拒绝，种子必须落在真实租户上。
- 裁剪掉的 CMS 能力：租户解析（OA v1 单租户 tenant 0，`tenant_code` 忽略）、`RegisterUser` / `WhoAmI`（双端 BFF 未暴露对应端点，落 Unimplemented）。

### 2.2 admin-service（HTTP 边端，管理后台转发）

`backend/app/admin/service/`，`appid = serviceid.AdminService`，consul key `go-wind-oa/admin/service`。

- `internal/server/rest_server.go`：创建 HTTP server，中间件链 `logging → auth.Server + authz.Server（白名单匹配）→ entmiddleware.Server()`。**auth 必须在 ent 之前**：`auth.Server` 对非白名单请求注入 `OperatorMetadata`，`entmiddleware.Server` 据此构建 `UserViewer`，`TenantPrivacy` 策略才生效；顺序颠倒则 ent 兜底 `SystemViewer`，租户隔离失效。
  - 注册 HTTP 服务（`adminV1.Register*HTTPServer`）：AuthenticationService、InternalMessageService / Category / Recipient、WorkflowService。
  - 白名单：`Login` / `GenerateCaptcha` / `VerifyCaptcha` 经 `rpc.AddWhiteList` 放行。
- `internal/data/`：data 层持 gRPC 客户端打 core-service（经服务发现定位 `CoreService`）。ProviderSet 见 `data/providers/wire_set.go`。
- `internal/service/`：转发层 service（各方法为 HTTP 请求 → gRPC 调 core）。
- `cmd/server/assets/`：`openapi.yaml` 由 `buf.admin.openapi.gen.yaml` 生成，`assets.go` embed 供 Swagger UI。

### 2.3 app-service（HTTP 边端，移动端转发）

`backend/app/app/service/`，`appid = serviceid.AppService`，consul key `go-wind-oa/app/service`。

- `internal/server/rest_server.go`：同 admin 中间件链与白名单模式。注册 HTTP 服务：AuthenticationService、WorkflowService。白名单仅 `OperationAuthenticationServiceLogin`。
- app BFF 匿名请求按 Host→租户 domain 解析（fail-closed）。`LoginRequest.tenant_code` 的 json_name 是蛇形（传 tenantCode 会被静默忽略）。
- `internal/data/`：data 层持 `NewAuthenticationServiceClient` + `NewWorkflowServiceClient`（打 core-service）。ProviderSet 见 `data/providers/wire_set.go`。
- `cmd/server/assets/`：`openapi.yaml` 由 `buf.app.openapi.gen.yaml` 生成。

### 2.4 sse_server.go

`app/app/service/internal/server/sse_server.go` 持 `AuthenticationServiceClient`，其 `WithAuthorizeFunc` 调 `ValidateToken` 验 SSE 订阅的 access token（admin-service 无此端点，因管理后台走轮询不走 SSE）。

---

## 3. proto 域分离

`backend/api/protos/` 下保留域：

| 域 | 包名 | 内容 | 性质 |
|---|---|---|---|
| `oa/service/v1/` | `oa.service.v1` | `workflow.proto` + `attendance.proto` + `leave.proto` + `expense.proto` + `oa_error.proto` | core 纯 gRPC，**无 http annotation** |
| `admin/service/v1/` | `admin.service.v1` | `i_authentication.proto` + `i_internal_message*.proto` + `i_workflow.proto` + `admin_doc.proto` + `admin_error.proto` | HTTP wrapper，引用 oa.service.v1 / internal_message.service.v1 / authentication.service.v1 消息类型 |
| `app/service/v1/` | `app.service.v1` | `i_authentication.proto` + `i_workflow.proto` + `app_doc.proto` + `app_error.proto` | HTTP wrapper，引用 oa.service.v1 / authentication.service.v1 消息类型 |
| `internal_message/service/v1/` | `internal_message.service.v1` | 4 档（CMS 原样保留） | 站内信消息类型，core 注册 gRPC |
| `authentication/service/v1/` | `authentication.service.v1` | 9 档（CMS 原样保留） | 鉴权消息类型，admin/app wrapper 引用 |
| `identity/service/v1/` | `identity.service.v1` | `user.proto` + `types.proto`（CMS 原样保留） | authentication 的传递闭包依赖 |

**核心分离原则**：core 的 `oa/service/v1/*.proto` 已剥离所有 `google.api.http` annotation，为纯 gRPC。HTTP 路由注解定义在 `admin/service/v1/i_workflow.proto` 与 `app/service/v1/i_workflow.proto` wrapper proto，这两者 `import "oa/service/v1/workflow.proto"` 仅引用消息类型，自身定义带 http annotation 的 service。鉴权同理：`i_authentication.proto` 引用 CMS `authentication.service.v1` 消息类型。

---

## 4. 数据模型（Ent Schema）

协同办公域各业务表均通过 `mixin.TenantID[uint32]{}` 注入 `tenant_id` 列并附加 `rule.TenantPrivacy` 策略。该策略由 `entmiddleware.Server` 注入的 `UserViewer.TenantID()` 驱动，自动在所有查询/写入上叠加 `tenant_id = viewer.tenant` 谓词 —— **代码层无需手写 tenant 过滤**。

### 4.1 工作流引擎表

| 表 | 说明 | 关键字段 | Mixin 组成 |
|---|---|---|---|
| `WorkflowDefinition` | 流程模板：有序节点配置 + 表单 schema | `node_config`(any), `form_schema`(any), `code`, `version`, `definition_status` | AutoIncrementId / TimeAt / OperatorID / TenantID / Remark |
| `WorkflowInstance` | 一次申请实例 | `form_data`(any), `instance_status`, `current_node_index`, `business_type`, `business_id` | 同上 |
| `WorkflowTask` | 节点上对指派审批人产生的待办 | `node_index`, `assignee_user_id`, `task_status` | AutoIncrementId / TimeAt / OperatorID / TenantID |
| `WorkflowLog` | append-only 审计日志 | `node_index`, `log_action`, `comment` | 同上 |

此外 core-service `ent/schema/` 还含三张 CMS 保留的 internal_message schema，供 `InternalMessageService` 落库。

**`field.Any` 的选择**：`node_config` / `form_schema` / `form_data` 结构动态，采用 ent `field.Any(name)`，DB 以 JSON 落盘、Go 侧 `any`。仓库层在 `Create` 时 `json.Unmarshal` 文本→any、在定向查询时 `json.Marshal` any→文本，显式转换，不依赖 mapper。

**外键与级联**：`Definition 1—N Instance`、`Instance 1—N Task`、`Instance 1—N Log` 均以 `edge.To(...).Required().Annotations(entsql.Annotation{OnDelete: entsql.DeleteCascade})` 声明，父行删除时级联清子行。

**索引**：每张表都有 `idx_*_tenant`；`WorkflowDefinition` 另有 `(tenant_id, code, version)` 唯一索引；`WorkflowTask` / `WorkflowLog` / `WorkflowInstance` 各有按 `(tenant_id, assignee_user_id, task_status)` / `(tenant_id, created_by)` 的检索索引，对应待办/已办/我的申请三类视图。

---

## 5. 工作流引擎状态机（core `workflow_service.go`）

`WorkflowService` 实现 kratos 生成的 `WorkflowServiceServer`（gRPC，`oa.service.v1`），注入仓库 + `*InternalMessageService`（同进程直接调用，非跨进程 gRPC 客户端）。引擎为**线性状态机**模型，节点支持会签/或签多审批人，但节点间仍严格线性推进（无并行分支/回退）。

### 5.1 节点配置格式

`node_config` 新格式（旧单人格式自动归一化兼容）：

```json
[{
  "approvers": [
    {"type": "USER", "id": 123},
    {"type": "LEADER"},
    {"type": "POSITION", "id": 456}
  ],
  "strategy": "ALL"
}]
```

- `strategy`：`ALL`=会签（全员通过才推进，任一驳回即驳回）；`ANY`=或签（一人通过即推进并取消其余，全员驳回才驳回）。
- 审批人类型：`USER`（显式用户）、`LEADER`（申请人主组织单元负责人，user_org_unit → org_unit.leader_id）、`POSITION`（职位在职持有者，user_position，可展开多人）。解析结果去重后每审批人一条并行 task。
- 申请人为审批人时自动跳过（对自己视为自动同意）；整节点全为申请人则写「自动通过」APPROVE 日志后跳过该节点继续推进，可连续跳多节点直至越界终结。

### 5.2 状态流转

```
SubmitApply ──> Instance(PENDING, idx=0) + 节点0 N个并行Task(PENDING) + Log(SUBMIT) + notify(A0..An)
   │
   ▼ AuditTask(APPROVE)  ← 仅当 task.assignee==caller 且 task.PENDING 且 instance.PENDING
   │   按节点 strategy 分支：
   │     ALL：本 task 关闭，若节点全部 task 终结则推进 idx+1；任一 REJECT → 实例立即 REJECTED + 取消兄弟 task
   │     ANY：本 task APPROVE → 立即推进 + 取消其余 PENDING task；全部 REJECT 才 REJECTED
   │
   ├─ idx+1 < len(nodes): 关闭本 task → Instance(idx=idx+1, PENDING) + 新节点 N个Task(PENDING) + Log(APPROVE) + notify
   │
   └─ idx+1 >= len(nodes): 关闭本 task → Instance(APPROVED) + Log(APPROVE) + 事件回调(APPROVED) + notify(applicant)   [终结]

WithdrawApply  ← 仅申请人本人 + 实例 PENDING
   → Instance(WITHDRAWN) + 全部 PENDING task→CANCELLED + Log(WITHDRAW) + 事件回调(WITHDRAWN) + notify(原审批人)   [终结]

AuditTask(FORWARD) → Task.assignee ← forwardTo（状态保持 PENDING，idx 不变）+ Log(FORWARD) + notify(forwardTo)
```

### 5.3 关键不变量与校验

- 任务关闭与实例状态推进在 service 层成对发生。
- `current_node_index` 仅在 `instance_status==PENDING` 时有意义；终结态写 `nil` 清空。
- `callerFromContext` 从 viewer context 取 `(tenantID, userID)`，二者任一为 0 即 fail-closed。
- `AuditTask` 强校验 `task.assignee == caller` 且 `task.PENDING`，否则 `ErrorForbidden`。
- 申请表单数据 `form_data` 仅在 `SubmitApply` 时透传落盘，后续审批流程不读不写。

### 5.4 业务事件挂钩

实例携带 `business_type`/`business_id`；`WorkflowEventRegistry`（进程内 map）在三个终态（APPROVED / REJECTED / WITHDRAWN）同步回调业务模块。回调须校验单据.instance_id 关联（防伪造）且仅处理 PENDING 单据（幂等）。

- **请假**：审批通过回调扣减额度（`AddUsedDays`，半日 0.5 步进）；驳回/撤回仅同步状态。
- **报销**：审批终态仅同步单据状态（额度无关）。

### 5.5 异步通知

`notifyManyAsync` 用 `context.WithoutCancel(ctx)` + 5s 超时 + `recover`，fire-and-forget 调用同进程 `InternalMessageService` 的 `SendMessage`。`context.WithoutCancel` 保留 viewer（SendMessage 从 viewer 推导发送者，防伪造），脱离已返回的 gRPC 请求生命周期。通知落 `internal_message_recipient` 表，并由 `InternalMessagePublisher` SSE 推送给在线客户端。通知失败不回滚状态机。

---

## 6. 请假子系统

表：`oa_leave_type` / `oa_leave_balance` / `oa_leave_application`。

- 类型（租户内 code 唯一）+ 额度（用户×类型×年度，total/used 支持半日 0.5 步进）+ 申请单。
- 提交：校验额度 → 建单 → 进程内直调引擎 SubmitApply（code=LEAVE v1）。**流程定义自动引导**：租户内无 LEAVE 定义时按默认模板（提交给申请人主管 LEADER，会签）自动创建并启用——开箱即用。
- 半日粒度：start_half（默认 AM）/ end_half（默认 PM），`computeLeaveDays` 半日算天（同日 PM起+AM止非法）。请求时间戳统一 `.In(time.Local)` 后截断（Timestamp.AsTime 恒 UTC 位置的坑）。
- 姓名回填：applicant_name（resolver 批量查 user.username）。

---

## 7. 报销子系统

表：`oa_expense_application` / `oa_expense_item`。

- 申请单 + 多行明细（类别/金额/日期/说明/**发票文件 ID**）。O2M 边 Cascade；**不可加 Required()**（阻断父记录创建，同 workflow 边教训）。
- 明细 List 需 WithItems 预载。提交挂 EXPENSE v1 流程（同样自动引导）。

### 7.1 发票直传链路

multipart `POST /app/v1/file/upload`（流式 reader 经 context 注入 minio）→ `UploadFileResponse` 返回 **file_id**（落库文件记录 ID）→ 明细 invoiceFileId 引用。移动端拍照/相册（image_picker）→ 压缩 → Dio multipart → 自动回填。

multipart 三要点（修复记录）：请求头补 `Accept: application/json`（kratos 响应编码按 Accept 回退 Content-Type）；oneof Source 打 File 标记（字节走 ctx reader 不走 proto 字段）；StorageObject 缺省空对象（自动桶名/UUID 对象名前提）。app BFF 曾双注册（生成版遮蔽手改流式版）已修。

---

## 8. 考勤子系统

表：`oa_attendance_record` / `oa_attendance_setting` / `oa_holiday`。

### 8.1 打卡与结算

- 打卡：当日首次=签到、第二次=签退并结算（GPS 经纬度 + WiFi BSSID 全程落库）。409 已签退。
- 结算：请假覆盖优先 ON_LEAVE；否则迟到（签到>上班时间）/早退（签退<下班时间）/正常。工时设置每租户一行（默认 09:00-18:00，admin 可改）。

### 8.2 节假日表

| 类型 | 语义 |
|------|------|
| HOLIDAY | 法定假日休息（可落工作日） |
| WORKDAY | 调休上班（可落周末） |

节假日优先于周末判定；未设置按周六日。管理员按日设置（存在则覆盖）。

> 休息日（节假日或周末）跳过结算物化；休息日打卡结算为 NORMAL（加班不计迟到）。定时结算调度器逐租户判定休息日。

### 8.3 每日定时结算

`AttendanceScheduler`（wire 注入常驻 goroutine）每 30 分钟检查，本地 00:30 后为「昨日」跑全租户结算。补结算仅处理仍 PENDING 的记录（幂等）。

---

## 9. 代码生成

每个 service 目录的 `Makefile` 为 `include ../../../app.mk`，`SERVICE_NAME` 决定 buf openapi 模板选择（core 跳过）。

| 目标 | 命令 | 产物 |
|---|---|---|
| ent ORM | `make ent`（各 service）| `internal/data/ent/` 下全套生成代码 |
| proto 桩（Go）| `make api`（`cd ../../../api && buf generate`）| `api/gen/go/{oa,internal_message,authentication,identity,admin,app}/service/v1/*.pb.go` + `*_grpc.pb.go` + `*_errors.pb.go` + `*.pb.validate.go` |
| openapi v3 | `make openapi`（admin/app，core 跳过）| `app/{admin,app}/service/cmd/server/assets/openapi.yaml` |
| wire DI | `make wire`（各 service）| `cmd/server/wire_gen.go` |

### 9.1 buf 模板

`backend/api/` 下 buf 模板：

| 模板 | 生成器 | inputs | out |
|---|---|---|---|
| `buf.gen.yaml` | protoc-gen-go*（Go 桩）| 保留域（`inputs.paths` 过滤）| `api/gen/go/`（per-domain `go_package` override 全指 `go-wind-oa/api/gen/go/...`）|
| `buf.admin.openapi.gen.yaml` | protoc-gen-openapi | `protos/admin/service/v1` | `app/admin/service/cmd/server/assets/` |
| `buf.app.openapi.gen.yaml` | protoc-gen-openapi | `protos/app/service/v1` | `app/app/service/cmd/server/assets/` |
| `buf.admin.typescript.gen.yaml` | protoc-gen-typescript-http | `protos/admin/service/v1` | `frontend/admin/src/api/generated/admin/service/v1/` |
| `buf.app.dart.gen.yaml` | protoc-gen-dart-http | `protos/app/service/v1` | `frontend/mobile/lib/generated/api/app/service/v1/` |

`buf.yaml`（v2，`modules.path: protos`）deps 含 googleapis / kratos / gnostic / pagination / protoc-gen-validate / redact 六个 remote。首次拉取须 `buf dep update` 生成 `buf.lock`。

### 9.2 ent feature 标志

`make ent` 的五个 feature（`privacy` / `entql` / `sql/modifier` / `sql/upsert` / `sql/lock`）。**`privacy` 不可省略** —— 它是 `TenantID` mixin 附着的 `rule.TenantPrivacy` 策略生效的前提。

### 9.3 wire

`wire_gen.go` 具现化依赖图：server ProviderSet + service ProviderSet + data ProviderSet + `newApp`。`wire.go` 携带 `//go:build wireinject` 标签，正常构建时由 `wire_gen.go` 提供实体。

---

## 10. 前端生成链

### 10.1 管理后台 TS 客户端

`buf.admin.typescript.gen.yaml` 生成 `frontend/admin/src/api/generated/admin/service/v1/index.ts`，含 `ApiClient.workflowService` / `authenticationService` / `internalMessageService` 等。Composables（`src/api/composables/{oa,auth}.ts`）封装为 Vue Query hooks。类型名带包前缀（`oaservicev1_*` / `authenticationservicev1_*`）。

### 10.2 移动端 Dart 客户端

`buf.app.dart.gen.yaml` 生成 `frontend/mobile/lib/generated/api/app/service/v1/index.dart`，含 `ApiClient.workflowService` / `authenticationService`。类型名带包前缀（`OaServiceV1*`）。枚举成员为小写（`pending` / `submitted` / `approve` / `reject` / `forward`）。

> 移动端的 Dart 客户端由 buf 模板直接生成，无 swagger_parser 中间层。移动端架构细节见 [oa-mobile-design.md](./oa-mobile-design.md)。

---

## 11. 测试与冒烟种子

- 纯逻辑单测：`internal/service/oa_logic_test.go`（computeLeaveDays 半日矩阵 / parseHHMM / truncateDate / isWeekend / 节点归一化与策略 / 迟到早退语义）。`go test -vet=off`（包内存量 vet 告警）。
- 冒烟种子工具：`app/core/service/cmd/smokeseed`（租户/角色/权限/组织/双用户/请假类型额度，输出加密登录密码）。

---

## 12. 已知边界与后续工作

- **无事务多步写入**（状态机/建单链）——与既有风险面一致。
- **请假天数按自然日**（含周末），未做工作日扣减；半天粒度已支持但额度过期/结转无。
- **转办不校验目标用户存在性**；或签部分驳回停留时无提醒。
