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
| 报销 | oa_expense_application / oa_expense_item | EXPENSE v1 流程，审批终态仅同步状态 | 已实现 |
| 出差 | oa_business_trip_application | BUSINESS_TRIP v1 流程，审批终态仅同步状态 | 已实现 |
| 加班 | oa_overtime_application | OVERTIME v1 流程，审批终态仅同步状态 | 已实现 |
| 用印 | oa_seal_application | SEAL_APPLICATION v1 流程，审批终态仅同步状态 | 已实现 |
| 外出 | oa_outing_application | OUTING v1 流程，审批终态仅同步状态 | 已实现 |
| 考勤 | oa_attendance_record / oa_attendance_setting / oa_holiday / oa_geofence / oa_wifi_fingerprint | — | 已实现 |
| 站内信 | internal_message / internal_message_category / internal_message_recipient | — | 已实现 |
| 公告发布 | （复用 internal_message，无独立表） | — | 已实现 |
| 通讯录 | （复用 identity.org_unit/user，无独立表） | — | 已实现（app 侧只读 wrapper，带 redact 脱敏） |

> 工作流引擎的寻人、状态机、事件回调机制是请假、报销及出差/加班/用印/外出等审批单据的共享底座。
>
> 公告发布复用站内信 `SendMessage` 的扇出机制：`target_all` 全员广播，`target_user_ids` 多播；按部门发布时经 `UserService.ListUserIDsByOrgUnitIDs`（core 实现，强制 `excludeExpired=true`）将 org_unit_ids 展开为 user_ids。无独立表。
>
> 通讯录为 app 侧新增只读 wrapper（`i_org_unit`/`i_user`，仅 List/Get，带 `redact` 脱敏），引用 `identity.service.v1` 消息类型，无 CRUD。admin 与 mobile 双端 UI 提供组织树浏览 + 成员列表。

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

- 令牌机制：access token 与 refresh token 均为 JWT（admin/app 两套独立 key，`configs/authenticator.yaml`），Redis 键统一前缀 `go_wind_oa:`。access token 经 Redis 缓存校验当前有效 token（`at:{ct}:{uid}`），支持黑名单（`bl:{jti}`）与按 jti/用户撤销。refresh token 为**自描述 JWT**——携带 uid/jti（与对应 access token 共用同一 jti 对）及 `cat=rt` 类别声明，过期时间取 RT 专属时长；Redis 中仍以 `rt:{ct}:{uid}` 键存该 RT 作为最终授权凭据，刷新轮换经 Lua 脚本原子「比对 RT→删 RT→删 AT」。刷新链身份由 RT 自身携带：刷新请求不依赖调用方上下文提供 userId/jti，`Authenticator.ParseRefreshToken` 从 RT 提取可信身份（签名/类别/过期校验通过）后交原子轮换完成。旧「不透明 RT」模式下 RT 不携带身份、刷新须从请求上下文取身份，致使刷新链在 BFF 边端被截断——此为刷新链修复的根因。
- 登录链路：password grant → `FindUserCredential`（AES 解密 → bcrypt 校验，恒定时间防用户名枚举）→ 用户状态检查 → 签发令牌对。密码传输格式与双端前端共享 `crypto.DefaultAESKey`（AES-CBC，key=IV，PKCS7，base64）。
- RBAC：登录要求用户有角色且角色含 `sys:access_backend` 权限码。identity/permission 域落地后改用真实角色表。
- 种子数据：core 启动种子 `admin/admin`（`configs/data.yaml`，bcrypt 入库）落 tenant 1 真实租户。`callerFromContext`（`workflow_service.go`）**仅对 `uid==0` fail-closed，不检查 `tid==0`**——即 tenant 0 的平台超管只要 uid 非 0 即经 viewer 放行，可访问多数 OA 端点（列表/详情/待办/考勤/额度等）。`backend/sql/postgresql-demo-data.sql` 因此提供平台超管测试组（`tenant_id=0`、`user/created_by/assignee=1`，账号 `admin`），与 tenant 1 租户管理员组（`tenant_id=1`、`user=2`）并列，用于验证两端可见性隔离。
- 原文档称「裁剪掉」的三项能力现均已实现：
  - **租户解析**：`doGrantTypePassword`（`authentication_service.go:203-215`）按 `tenant_code` 经 `tenantRepo.Get` 定位租户（空则视 tenant 0 平台域），查不到或非启用统一返回同一文案防枚举，解析出的 tenantID 限定后续凭证查询范围。
  - **`RegisterUser`**：完整实现（`authentication_service.go:397-469`），事务内建用户 + 凭证 + 分配租户管理员角色；admin BFF 已将其加入白名单暴露（见 §2.2）。
  - **`WhoAmI`**：完整实现（`authentication_service.go:527-549`），从 viewer 上下文解析调用者身份并查其 username，不接受客户端传入标识。
  - 实际未实现的是 `client_credentials` grant（`doGrantTypeClientCredentials` 恒返回 `invalid grant type`）。

### 2.2 admin-service（HTTP 边端，管理后台转发）

`backend/app/admin/service/`，`appid = serviceid.AdminService`，consul key `go-wind-oa/admin/service`。

- `internal/server/rest_server.go`：创建 HTTP server，中间件链 `logging → auth.Server + authz.Server（白名单匹配）→ entmiddleware.Server()`。**auth 必须在 ent 之前**：`auth.Server` 对非白名单请求注入 `OperatorMetadata`，`entmiddleware.Server` 据此构建 `UserViewer`，`TenantPrivacy` 策略才生效；顺序颠倒则 ent 兜底 `SystemViewer`，租户隔离失效。
  - 注册 HTTP 服务（`adminV1.Register*HTTPServer`）：§3 admin wrapper 清单所示全部 wrapper 均注册 HTTP server（平台管理 + OA 业务 + 站内信 + 鉴权 + 文件传输），实现层为 `internal/service/` 转发层（HTTP → gRPC core）。
  - 白名单（`rpc.AddWhiteList`，5 项）：`OperationAuthenticationServiceLogin` / `OperationAuthenticationServiceRefreshToken` / `OperationAuthenticationServiceGenerateCaptcha` / `OperationAuthenticationServiceVerifyCaptcha` / `OperationAuthenticationServiceRegisterUser`。RefreshToken 与 RegisterUser 在内即放行，故刷新链在 BFF 边端不被 401 拦截（自描述 RT 机制见 §2.1）。
- `internal/data/`：data 层持 gRPC 客户端打 core-service（经服务发现定位 `CoreService`）。ProviderSet 见 `data/providers/wire_set.go`。
- `internal/service/`：转发层 service（各方法为 HTTP 请求 → gRPC 调 core）。
- `cmd/server/assets/`：`openapi.yaml` 由 `buf.admin.openapi.gen.yaml` 生成，`assets.go` embed 供 Swagger UI。

### 2.3 app-service（HTTP 边端，移动端转发）

`backend/app/app/service/`，`appid = serviceid.AppService`，consul key `go-wind-oa/app/service`。

- `internal/server/rest_server.go`：同 admin 中间件链与白名单模式。注册 HTTP 服务：§3 app wrapper 清单所示全部 wrapper 均注册 HTTP server。白名单（2 项）：`OperationAuthenticationServiceLogin` / `OperationAuthenticationServiceRefreshToken`。
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
| `oa/service/v1/` | `oa.service.v1` | `attendance.proto` + `business_trip.proto` + `expense.proto` + `leave.proto` + `oa_error.proto` + `outing.proto` + `overtime.proto` + `seal_application.proto` + `workflow.proto` | core 纯 gRPC，**无 http annotation** |
| `admin/service/v1/` | `admin.service.v1` | 实际 wrapper 清单（全量，见仓内 `protos/admin/service/v1`）：`i_admin_portal`/`i_api`/`i_api_audit_log`/`i_attendance`/`i_authentication`/`i_business_trip`/`i_dashboard`/`i_data_access_audit_log`/`i_dict_entry`/`i_dict_type`/`i_expense`/`i_file`/`i_file_transfer`/`i_internal_message`/`i_internal_message_category`/`i_internal_message_recipient`/`i_language`/`i_leave`/`i_login_audit_log`/`i_login_policy`/`i_menu`/`i_mfa`/`i_operation_audit_log`/`i_org_unit`/`i_outing`/`i_overtime`/`i_permission`/`i_permission_audit_log`/`i_permission_group`/`i_plan`/`i_plan_module`/`i_plan_quota`/`i_policy_evaluation_log`/`i_position`/`i_redis_cache_monitor`/`i_role`/`i_seal_application`/`i_task`/`i_tenant`/`i_user`/`i_user_profile`/`i_workflow`，另含 `admin_doc`/`admin_error`。其中 OA 业务域 wrapper（`i_attendance`/`i_business_trip`/`i_expense`/`i_leave`/`i_outing`/`i_overtime`/`i_seal_application`/`i_workflow`）引用 `oa.service.v1`，站内信 wrapper（`i_internal_message*`）引用 `internal_message.service.v1`，鉴权 wrapper（`i_authentication`）引用 `authentication.service.v1`，通讯录/用户 wrapper（`i_org_unit`/`i_user`/`i_user_profile`）引用 `identity.service.v1`；其余为平台管理（套餐/配额/字典/菜单/角色/权限/审计日志/MFA/登录策略/仪表盘/Redis 监控等）wrapper，引用 `identity.service.v1` 等共享消息类型。CMS 业务域（原 category/comment/post/section/tag/page/navigation/site 等）已从该目录移除，不再生成对应 client。 | HTTP wrapper |
| `app/service/v1/` | `app.service.v1` | 实际 wrapper 清单（全量，见仓内 `protos/app/service/v1`）：`i_attendance`/`i_authentication`/`i_business_trip`/`i_expense`/`i_file_transfer`/`i_internal_message`/`i_leave`/`i_org_unit`/`i_outing`/`i_overtime`/`i_seal_application`/`i_user`/`i_user_profile`/`i_workflow`，另含 `app_doc`/`app_error`。其中 OA 业务域 wrapper（`i_attendance`/`i_business_trip`/`i_expense`/`i_leave`/`i_outing`/`i_overtime`/`i_seal_application`/`i_workflow`）引用 `oa.service.v1`，站内信 wrapper（`i_internal_message`）引用 `internal_message.service.v1`，鉴权 wrapper（`i_authentication`）引用 `authentication.service.v1`，只读通讯录 wrapper（`i_org_unit`/`i_user`，带 `redact` 脱敏）与用户资料 wrapper（`i_user_profile`）引用 `identity.service.v1`，文件传输 wrapper（`i_file_transfer`）引用 `storage.service.v1`。CMS 业务域（原 category/comment/post/section/tag/page/navigation 等）已从该目录移除，不再生成对应 client。 | HTTP wrapper |
| `internal_message/service/v1/` | `internal_message.service.v1` | 4 档（CMS 原样保留） | 站内信消息类型，core 注册 gRPC |
| `authentication/service/v1/` | `authentication.service.v1` | 9 档（CMS 原样保留） | 鉴权消息类型，admin/app wrapper 引用 |
| `identity/service/v1/` | `identity.service.v1` | `user.proto` + `types.proto`（CMS 原样保留） | authentication 的传递闭包依赖 |

**核心分离原则**：core 的 `oa/service/v1/*.proto` 已剥离所有 `google.api.http` annotation，为纯 gRPC。HTTP 路由注解定义在 `admin/service/v1/` 与 `app/service/v1/` 下各 `i_*` wrapper proto（如 `i_workflow`、`i_attendance`、`i_business_trip` 等）——这些 wrapper `import` 上表 core 域 proto 仅引用消息类型，自身定义带 http annotation 的 service。鉴权、站内信、通讯录同理：`i_authentication.proto` 引用 `authentication.service.v1`，`i_internal_message*.proto` 引用 `internal_message.service.v1`，`i_org_unit`/`i_user`（app 只读通讯录）引用 `identity.service.v1`。

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

实例携带 `business_type`/`business_id`；`WorkflowEventRegistry`（进程内 map）在三个终态（APPROVED / REJECTED / WITHDRAWN）同步回调业务模块。回调须校验单据.instance_id 关联（防伪造）且仅处理 PENDING 单据（幂等）。已注册挂钩的业务类型：

| business_type | 模块 | 终态回调行为 |
|---|---|---|
| `LEAVE` | 请假 | 审批通过扣减额度（`AddUsedDays`，半日 0.5 步进）；驳回/撤回仅同步状态 |
| `EXPENSE` | 报销 | 审批终态仅同步单据状态（额度无关） |
| `BUSINESS_TRIP` | 出差 | 同上报销：仅同步状态，无额度副作用 |
| `OVERTIME` | 加班 | 同上 |
| `SEAL_APPLICATION` | 用印 | 同上 |
| `OUTING` | 外出 | 同上 |

> 出差/加班/用印/外出四类同型审批单据，其业务表仅含四态状态枚举 + `instance_id` 关联 + `form_schema`，与报销同构；引擎通过 `ensureWorkflowDefinition` 在租户缺定义时按默认模板（提交给申请人主管 LEADER，会签）自动创建并启用。

### 5.5 异步通知

`notifyManyAsync` 用 `context.WithoutCancel(ctx)` + 5s 超时 + `recover`，fire-and-forget 调用同进程 `InternalMessageService` 的 `SendMessage`。`context.WithoutCancel` 保留 viewer（SendMessage 从 viewer 推导发送者，防伪造），脱离已返回的 gRPC 请求生命周期。通知落 `internal_message_recipient` 表。

> **SSE 投递的局限**：`InternalMessagePublisher`（SSE 推送）只在 admin-service 的 `internal_message_service.go` 里注册，且仅 admin-service 自身经 HTTP 暴露的 `SendMessage` 路径会触发它（写 recipient 后查收件人的 admin 会话 access token 并 Publish）。core 是 gRPC-only、无 SSE server；工作流通知走的是 core 进程内 `SendMessage`，因此**不会触发任何 SSE 推送**。该限制的接收端影响见 §12。通知失败不回滚状态机。

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

表：`oa_attendance_record` / `oa_attendance_setting` / `oa_holiday` / `oa_geofence` / `oa_wifi_fingerprint`。

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

### 8.4 打卡地理围栏与 Wi-Fi 指纹白名单

表：`oa_geofence`（name / latitude / longitude / radius_meters，**圆形围栏**——圆心+半径，非多边形）、`oa_wifi_fingerprint`（ssid 仅描述用、bssid 为比对键）。两表均 `mixin.TenantID[uint32]{}` 租户隔离，`(tenant_id)` 非唯一索引。

- **门控**：`CheckIn` 在落打卡记录前调 `validateLocation`，分围栏与 Wi-Fi 两支，各自独立判定。**任一支对应的租户表为空则该支放行；两表皆空则完全放行**（特性休眠，开箱默认不限制）：
  - 围栏支：`geofenceRepo.ListByTenant` 取租户全部围栏。有围栏时，若打卡点 `lat==0 && lon==0` → 403「定位不可用，无法判定打卡范围」；否则将 WGS-84 打卡点经 `wgs84ToGcj02` 转入 GCJ-02，逐围栏 `haversineMeters` 球面距离比对 `radius_meters`，落在任一围栏半径内放行，否则 403「您不在允许的打卡范围内」。
  - Wi-Fi 支：`wifiRepo.ListByTenant` 取租户白名单。有白名单时，`bssid==""` → 403「当前 Wi-Fi 不可识别」；否则线性比对 `w.Bssid`，命中放行，否则 403「当前 Wi-Fi 不在允许的打卡网络列表中」。
- **坐标系统一**：移动端 GPS（`geolocator`，WGS-84）与管理端围栏坐标（高德地图圈选产出，GCJ-02、原样入库）异源；后端比对前经 `geo.go` 的 `wgs84ToGcj02` 把打卡点转 GCJ-02，使两侧在同一坐标系下度量。
- **管理端配置**：core 暴露 `UpsertGeofence`/`DeleteGeofence`/`ListGeofences` 与 `UpsertWifiFingerprint`/`DeleteWifiFingerprint`/`ListWifiFingerprints` 六个 gRPC（`oa.service.v1`），经 admin BFF HTTP wrapper 转发。admin 后台围栏页用 `AmapCirclePicker`（高德 `AMap.Map` + `CircleEditor` 圈选圆心半径）配置；**未配置高德 Key/安全码时降级为三个数值输入框（纬度/经度/半径）手填**（`use-amap.ts` 读 `VITE_AMAP_KEY`/`VITE_AMAP_SECURITY_CODE`，缺失即 `AMAP_NOT_CONFIGURED` 触发降级）。Wi-Fi 白名单页为纯文本 SSID/BSSID 输入。
- **移动端无改动**：`attendance_service.dart` 仍以 `geolocator` 取 GPS、`network_info_plus` 尽力取 BSSID 提交；越界/非白名单时收上述 403，消息文本经 `Status` 分支原样透传给用户。

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

`buf.admin.typescript.gen.yaml` 生成 `frontend/admin/src/api/generated/admin/service/v1/index.ts`，内含 42 个 `create*ServiceClient` 工厂函数，按域归类如下（与 §3 admin wrapper 清单一一对应）：

| 域 | client 函数 |
|---|---|
| OA 业务 | `createAttendanceService` / `createBusinessTripService` / `createExpenseService` / `createLeaveService` / `createOutingService` / `createOvertimeService` / `createSealApplicationService` / `createWorkflowService` |
| 站内信 | `createInternalMessageService` / `createInternalMessageCategoryService` / `createInternalMessageRecipientService` |
| 鉴权 | `createAuthenticationService` |
| identity 平台管理 | `createAdminPortalService` / `createApiService` / `createApiAuditLogService` / `createDashboardService` / `createDataAccessAuditLogService` / `createDictEntryService` / `createDictTypeService` / `createLanguageService` / `createLoginAuditLogService` / `createLoginPolicyService` / `createMenuService` / `createMfaService` / `createOperationAuditLogService` / `createOrgUnitService` / `createPermissionService` / `createPermissionAuditLogService` / `createPermissionGroupService` / `createPlanService` / `createPlanModuleService` / `createPlanQuotaService` / `createPolicyEvaluationLogService` / `createPositionService` / `createRedisCacheMonitorService` / `createRoleService` / `createTaskService` / `createTenantService` / `createUserService` / `createUserProfileService` |
| 文件传输 | `createFileService` / `createFileTransferService` |

Composables（`src/api/composables/`）将上述 client 封装为 Vue Query hooks。类型名带包前缀（`oaservicev1_*` / `authenticationservicev1_*` / `identityservicev1_*` / `storageservicev1_*`），与 §10.2 所述 Dart 客户端同理。

### 10.2 移动端 Dart 客户端

`buf.app.dart.gen.yaml` 生成 `frontend/mobile/lib/generated/api/app/service/v1/index.dart`，含 `ApiClient.workflowService` / `authenticationService`。类型名带包前缀（`OaServiceV1*`）。枚举成员为小写（`pending` / `submitted` / `approve` / `reject` / `forward`）。

> 移动端的 Dart 客户端由 buf 模板直接生成，无 swagger_parser 中间层。移动端架构细节见 [oa-mobile-design.md](./oa-mobile-design.md)。

---

## 11. 测试与冒烟种子

- 纯逻辑单测：`internal/service/oa_logic_test.go`（computeLeaveDays 半日矩阵 / parseHHMM / truncateDate / isWeekend / 节点归一化与策略 / 迟到早退语义）。`go test -vet=off`（包内存量 vet 告警）。
- 冒烟种子工具：`app/core/service/cmd/smokeseed`（租户/角色/权限/组织/双用户/请假类型额度，输出加密登录密码）。

---

## 12. 已知边界与后续工作

- **状态机推进路径与建单链已包事务，定义管理仍无事务**：`SubmitApply` 建实例 + 写 SUBMIT 日志、以及 `launchFromNode`/`advanceInstance`/`handleReject`/`handleApprove`/`handleForward`/`WithdrawApply` 的跨 repo 多步写，均经 `WorkflowInstanceRepo.Txn` 包入单一 `ent.Tx`，原子提交/回滚。`SubmitApply` 后续的首节点启动走 `launchFromNode` 自身事务（不嵌套）。仍无事务的：`WorkflowDefinition` CRUD（定义管理，单表操作）、单步写路径。
- **请假额度无过期/结转机制**：半天粒度已支持；天数计算已扣除休息日（委托 `AttendanceService.isRestDay`，节假日表优先否则按周末，与考勤结算一致）。但额度按年度 total/used 无过期清理、无跨年结转。
- **工作流通知无 SSE 实时投递，admin 端有轮询兜底**。经 §5.5 所述，工作流引擎经 core 进程内 `SendMessage` 落库的通知不会触发 SSE（SSE publisher 只存在于 admin-service 且只对 admin 自身 HTTP `SendMessage` 路径生效）。各端实际表现：
  - **移动端**：app-service 的 `internal_message_service.go` 无 SSE publisher，也无 `SendMessage`；通知页经 REST `GET /app/v1/internal-message/my-messages` 轮询拉取收件箱。core 落库的工作流通知因此**延迟可见、不丢失**（取决于下次轮询时机），但无实时推送。
  - **管理后台**：`NoticeDropdown/useNotice.ts` 经 `InternalMessageRecipientService.ListUserInbox` 拉取收件箱，订阅 `globalSSEClient` 的 `"notification"` 事件（admin 自身 `SendMessage` 产生的新收件行实时到达），并以 30s 间隔定时轮询 `ListUserInbox` 兜底刷新。core 落库的工作流通知经此轮询在 ≤30s 内可见，不再需要手动重载；但仍非实时推送。
- **通讯录无 DataScope 授权收敛**：app 只读通讯录 wrapper（`i_org_unit`/`i_user`）与 admin 侧 user/org_unit 端点均按租户隔离暴露，但未做按数据范围（DataScope）的可见性收敛。目录页展示全体租户用户及其部门标注，未做按部门过滤。留后续。
