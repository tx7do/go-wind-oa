# go-wind-oa · 轻量级工作流审批引擎 — 架构设计文档

> ⚠️ **本文档已过时。** 撰写于旧的单 service 架构时期（`app/oa/service`），描述的是「OA 模块作为 go-wind-cms 的扩展」模型。当前实际架构为 core/admin/app 三 service 分離，proto 域分離，`pkg/` 自包含——与本文描述的目录结构、proto 路径（`oa/v1`）、生成模板名（`buf.vue-element.oa.typescript.gen.yaml`）、`swagger_parser` 链等均不符。以 `README.md` 与代码为权威架构来源。
>
> 本文档面向维护者，记录 `go-wind-oa` 工作流模块的架构决策、与 `go-wind-cms` 底座的集成点，以及当前实现的边界与已知约束。读者应已熟悉 Kratos + Wire + Ent 的基本范式。

---

## 1. 设计目标与范围

`go-wind-oa` 在 `go-wind-cms` 底座之上扩展出一个**轻量级、线性、状态机驱动**的工作流审批引擎。它复用 cms 既有的多租户隔离、鉴权与站内信通知组件，不自建通知通道、不重写租户策略，遵循“积木式架构(bricket-like architecture)”。

**范围内（v1）**
- 流程定义（含节点序列、动态表单 schema）的存储与查询。
- 申请提交 → 首节点任务派生 → 审批/驳回/转办 → 状态机推进或终结。
- “我的待办 / 已办 / 我的申请”三类列表。
- 审批流转的异步通知，复用 cms `internal_message` 站内信通道（落库 + SSE 推送）。

**范围外（v1，刻意未实现）**
- 会签 / 并行分支 / 子流程 —— 当前为严格线性状态机。
- 角色级 / 部门主管级审批人指派 —— `node_config` 仅支持 `approver_type=="USER"` 指定用户。对接 cms identity（角色、组织树）以支持更丰富的指派策略是后续工作。
- 流程定义的启用/禁用独立接口（定义落 `DRAFT`，校验 `ENABLED`；切换接口未提供）。
- 流程定义的版本管理 UI、回滚等。

---

## 2. 与 go-wind-cms 的集成点（砖块复用）

OA 模块作为独立 Go module（`go-wind-oa`），通过 `go.mod` 依赖 `go-wind-cms`，复用其以下组件，**不在 OA 内重写**：

| 组件 | 来源（go-wind-cms 路径） | OA 中的用途 |
|---|---|---|
| `auth.Server` 鉴权中间件 | `pkg/middleware/auth/auth.go` | HTTP 请求 JWT 校验，注入 `OperatorMetadata` |
| `entmiddleware.Server` viewer 中间件 | `pkg/middleware/ent/ent.go` | 依 `OperatorMetadata` 构建带租户作用域的 `UserViewer`，驱动 `TenantPrivacy` |
| `TenantID` / `TimeAt` / `OperatorID` / `AutoIncrementId` mixins | `github.com/tx7do/go-crud/entgo/mixin` | 为四张表注入 `tenant_id` 等审计列与租户策略 |
| `entCrud.EntClient` / `entCrud.Repository` / `mapper` | `github.com/tx7do/go-crud/entgo`、`go-utils` | ORM 会话与 DTO↔entity 映射（与 cms repos 同构） |
| `internal_message` gRPC 客户端 | `api/gen/go/internal_message/service/v1` | `SendMessage` 投递审批通知（NOTIFICATION 类型） |
| 服务发现 | `github.com/tx7do/kratos-bootstrap/registry` | 经 `core-service` 定位 cms 以建上述 gRPC 连接 |
| `kratos-bootstrap` 应用骨架 | `github.com/tx7do/kratos-bootstrap/bootstrap` | `NewApp` / `RunApp` / 配置加载 |

**关键安全约束：中间件顺序不可颠倒。** HTTP 中间件链必须是 `logging → auth.Server → entmiddleware.Server`。理由见 `app/oa/service/internal/server/rest_server.go` 注释：`auth.Server` 对非白名单请求注入 `OperatorMetadata`，`entmiddleware.Server` 据此构建 `UserViewer`，`TenantPrivacy` 策略才生效。顺序颠倒则 `entmiddleware` 以 `md==nil` 兜底为 `SystemViewer`，**租户隔离失效**。

OA 的 HTTP 白名单为空 —— 所有工作流接口都必须携带有效 JWT，无公开端点。

---

## 3. 目录结构

```
go-wind-oa/
├── backend/
│   ├── api/
│   │   ├── oa/v1/
│   │   │   ├── workflow.proto        # 服务 + 全部消息（含 http 注解，单层）
│   │   │   └── oa_error.proto        # OaErrorReason → protoc-gen-go-errors
│   │   ├── buf.gen.yaml              # managed go_package: oa/v1 → oav1
│   │   └── buf.yaml
│   │   └── gen/go/oa/v1/             # buf 生成产物（不入库）
│   └── app/oa/service/
│       ├── Makefile                  # 仅 include 自带目标（ent/api/build/run）
│       ├── cmd/server/
│       │   ├── main.go               # kratos-bootstrap 应用入口
│       │   └── wire.go               # wire 注入器（wireinject tag）
│       └── internal/
│           ├── data/                 # ent 仓库层
│           │   ├── client/ent_client.go          # NewEntClient（复用 cms core 构造）
│           │   ├── discovery.go                   # NewDiscovery（服务发现）
│           │   ├── notification_client.go        # cms internal_message gRPC 客户端
│           │   ├── viewer.go                     # ViewerUserIDFromContext
│           │   ├── workflow_definition_repo.go
│           │   ├── workflow_instance_repo.go
│           │   ├── workflow_task_repo.go
│           │   ├── workflow_log_repo.go
│           │   ├── ent/schema/                   # 四张表 schema（见下节）
│           │   │   ├── workflow_definition.go
│           │   │   ├── workflow_instance.go
│           │   │   ├── workflow_task.go
│           │   │   └── workflow_log.go
│           │   └── providers/wire_set.go         # data ProviderSet（wireinject）
│           ├── server/
│           │   ├── rest_server.go                # 中间件链 + Register*HTTPServer
│           │   └── providers/wire_set.go         # server ProviderSet
│           └── service/
│               ├── workflow_service.go           # 状态机驱动（见 §5）
│               └── providers/wire_set.go         # service ProviderSet
└── docs/oa-workflow-design.md        # 本文档
```

---

## 4. 数据模型（Ent Schema）

四张表，均通过 `mixin.TenantID[uint32]{}` 注入 `tenant_id` 列并附加 `rule.TenantPrivacy` 策略。该策略由 `entmiddleware.Server` 注入的 `UserViewer.TenantID()` 驱动，自动在所有查询/写入上叠加 `tenant_id = viewer.tenant` 谓词 —— **代码层无需手写 tenant 过滤**，仓库层的 `Where(...)` 仅叠加业务谓词。

| 表 | 说明 | 关键字段 | Mixin 组成 |
|---|---|---|---|
| `WorkflowDefinition` | 流程模板：有序节点配置 + 表单 schema | `node_config`(any), `form_schema`(any), `code`, `version`, `definition_status` | AutoIncrementId / TimeAt / OperatorID / TenantID / Remark |
| `WorkflowInstance` | 一次申请实例 | `form_data`(any), `instance_status`, `current_node_index` | 同上 |
| `WorkflowTask` | 节点上对指派审批人产生的待办 | `node_index`, `assignee_user_id`, `task_status` | AutoIncrementId / TimeAt / OperatorID / TenantID |
| `WorkflowLog` | append-only 审计日志 | `node_index`, `log_action`, `comment` | 同上 |

**`field.Any` 的选择**：`node_config` / `form_schema` / `form_data` 结构动态，采用 ent 官方推荐的 `field.Any(name)`（`schema/field/field.go:114`），DB 以 JSON 落盘、Go 侧 `any`。仓库层在 `Create` 时 `json.Unmarshal` 文本→any、在定向查询时 `json.Marshal` any→文本，显式转换，不依赖 mapper。状态机推进时**不读 `form_data`**，只读节点配置 —— 故 `WorkflowInstance.GetState` 经 mapper 取 status/指针，`form_data` 由 mapper 跳过。

**外键与级联**：`Definition 1—N Instance`、`Instance 1—N Task`、`Instance 1—N Log` 均以 `edge.To(...).Required().Annotations(entsql.Annotation{OnDelete: entsql.Cascade})` 声明，父行删除时级联清子行，避免孤儿。

**索引**：每张表都有 `idx_*_tenant`（按 tenant 检索）；`WorkflowDefinition` 另有 `(tenant_id, code, version)` 唯一索引防止同租户重复定义；`WorkflowTask` / `WorkflowLog` / `WorkflowInstance` 各有按 `(tenant_id, assignee_user_id, task_status)` / `(tenant_id, created_by)` 的检索索引，对应“待办 / 已办 / 我的申请”三类视图。

---

## 5. 状态机与服务层（`workflow_service.go`）

`WorkflowService` 实现 kratos 生成的 `WorkflowServiceHTTPServer`，注入四个仓库 + cms `internal_message` gRPC 客户端。状态机为**线性、单实例单活跃任务**模型：

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
- 任务关闭与实例状态推进在 service 层成对发生（`handleApprove`/`handleReject` 同时调 `taskRepo.UpdateStatus(taskID,...)` 与 `instanceRepo.UpdateStatus(instanceID,...)`；`handleForward` 仅 `taskRepo.UpdateAssignee(taskID,...)`）。
- `GetState` 直读 entity 的 status/指针，绕过 mapper，避免 `field.Any` 的不确定指针行为影响状态判断。
- `current_node_index` 仅在 `instance_status==PENDING` 时有意义；终结态（APPROVED/REJECTED）写 `nil` 清空。
- `callerFromContext` 从 viewer context 取 `(tenantID, userID)`，二者任一为 0 即 fail-closed（`ErrorBadRequest("missing viewer context")`）。
- `AuditTask` 强校验 `task.assignee == caller` 且 `task.PENDING`，否则 `ErrorForbidden("not your pending task")` —— 防止越权审批他人任务。
- 申请表单数据 `form_data` 仅在 `SubmitApply` 时透传落盘，**后续审批流程不读不写**，故其结构对引擎透明。

**异步通知**：`notifyAsync` 用 `context.WithoutCancel(ctx)` + 5s 超时 + `recover`，fire-and-forget 调用 `internalMessageV1.SendMessage`（`Type=NOTIFICATION`、`TargetUserIds=[recipient]`）。经 cms core 落 `internal_message_recipient` 收件箱行，并由 cms admin 网关的 SSE publisher 推送给在线客户端。通知失败不回滚状态机 —— 审批结果已持久化，通知仅为辅助提示。

**已办/我的申请视图的字段填充差异**：
- `ListType_PENDING`：`WorkflowTask` 行 → `MyTaskItem{task_id, instance_id, status_label, occurred_at=task.created_at}`。
- `ListType_DONE`：`WorkflowLog` 行（仅 APPROVE/REJECT/FORWARD，排除 SUBMIT）→ `MyTaskItem{log_id, instance_id, action_label, occurred_at=log.created_at}`。
- `ListType_SUBMITTED`：`WorkflowInstance` 行 → `MyTaskItem{instance_id, title, status_label, occurred_at=instance.created_at}`。

---

## 6. 代码生成

| 目标 | 命令 | 产物 |
|---|---|---|
| ent ORM | `make ent`（`app/oa/service/Makefile`）| `internal/data/ent/` 下全套生成代码 |
| proto 桩 | `make api`（`cd ../../../api && buf generate`）| `api/gen/go/oa/v1/*.pb.go`、`*_http.pb.go`、`*_grpc.pb.go`、`oa_error_errors.pb.go`、`*.pb.validate.go` |
| wire DI | `make wire`（`go run github.com/google/wire/cmd/wire ./cmd/server`）| `cmd/server/wire_gen.go`（提供 `main.go` 引用的 `initApp`） |

`buf.gen.yaml` 与 cms 同构，唯一差异是新增 `oa/v1` 路径的 managed `go_package` 覆盖，指向 `go-wind-oa/api/gen/go/oa/v1;oav1`。源 proto 不写 `go_package` 选项，由 managed 模式注入 —— 与 cms 各 `*_service` 路径处理一致。`buf.yaml` 同样与 cms 同构（`modules.path: protos`、相同的 `deps` 列表与 `disable` 块），proto 源码落 `api/protos/oa/v1/`，对齐 cms 的 `api/protos/<domain>/service/v1/` 布局；首次拉取远程 proto 依赖须执行 `buf dep update`（生成 `buf.lock`）。

`ent` 目标的五个 feature 标志（`privacy` / `entql` / `sql/modifier` / `sql/upsert` / `sql/lock`）与 cms 完全相同。**`privacy` 不可省略** —— 它是 `TenantID` mixin 附着的 `rule.TenantPrivacy` 策略生效的前提，省略后 `tenant_id` 列仍存在但无行级隔离，等于退化为软隔离。

`wire` 目标与 `go-wind-cms/backend/app.mk` 的同名目标同构。生成的 `wire_gen.go` 具现化完整依赖图：ent 客户端 → 四个仓库 + 两个 cms gRPC 客户端（`internal_message` 定位 `core-service`、`authentication` 定位 `admin-service`，均经服务发现）→ `WorkflowService` + `AuthenticationService` → `http.Server`。依赖图能成功解析本身即是对所有构造函数签名与 provider set 一致性的强校验。`wire.go` 携带 `//go:build wireinject` 标签，正常构建时由 `wire_gen.go` 提供实体；`wire_gen.go` 不入库则 `main.go` 的 `initApp` 无定义，构建失败。

### 6.1 前端客户端生成链（2026-08-17 新增）

OA 后端除了上述 Go 桩/wire 生成，还为两套前端各提供一条生成链，均经 buf 模板驱动，源 proto 同为 `api/protos/oa/v1/`：

| 目标 | 模板 | 命令 | 产物 |
|---|---|---|---|
| Admin TS 客户端 | `buf.vue-element.oa.typescript.gen.yaml` | `cd backend/api && buf generate --template buf.vue-element.oa.typescript.gen.yaml` | `frontend/admin/src/api/generated/oa/v1/index.ts`（含 `ApiClient.workflowService` 与 `ApiClient.authenticationService`，分别供 `src/api/composables/oa.ts` 与 `auth.ts` 封装为 Vue Query hooks） |
| Mobile OpenAPI v3 文档 | `buf.oa.openapi.gen.yaml` | `cd backend/app/oa/service && make openapi` | `backend/app/oa/service/cmd/server/assets/openapi.yaml`（供 `frontend/mobile` 的 swagger_parser 消费，生成 Dart 客户端） |

两条链分别对齐 `go-wind-admin` 与 `go-wind-cms` 的同名模板，差异仅在 `inputs`/`out` 指向 OA 自有 proto/产物路径。前端侧的代码归属、基座拷贝决策与四功能范围见 `docs/oa-mobile-design.md`（mobile）与本仓 `frontend/admin/` 的实现（admin）。

> 注：mobile 的 swagger_parser 执行依赖 Flutter SDK，当前开发机未安装；admin 的 TS 客户端已在仓内验证生成成功。

---

## 7. 已知边界与后续工作

1. **节点指派仅支持指定用户**。`resolveApprover` 拒绝非 `USER` 类型。扩展为角色/部门主管指派需：(a) `node_config` 增 `approver_role` / `approver_dept` 字段；(b) service 层调 cms `identity` 服务解析具体用户ID；(c) 处理“多候选审批人”时的任务派生策略（会签或选一）。
2. **定义启用/禁用接口未提供**。`CreateWorkflowDefinition` 一律落 `DRAFT`，`SubmitApply` 校验 `ENABLED`。需补 `UpdateWorkflowDefinition`（带 `update_mask` 限定 `definition_status`）或专用启用接口。
3. **无定时任务 / 超时处理**。长期 PENDING 任务不会自动催办或超时终结，需接 cms `task` 模块的定时调度。
4. **无会签 / 并行 / 回退**。线性状态机的硬约束；扩展需重设计 `current_node_index` 为节点集合，并改造 `WorkflowTask` 的“一实例一活跃任务”不变量。
5. **通知仅文本**。`notifyAsync` 的 title/content 为固定文案；如需携带申请详情链接，需扩 `SendMessageRequest` 或前端按 `instance_id` 自取。
6. **未接入 cms 审计日志**（`audit` 模块）。OA 的所有写操作目前只落 `WorkflowLog`，未额外写 cms `operation_audit_log`。如需统一审计，可在 service 层按 cms admin 网关的 `applogging` 模式接 `OperationAuditLogServiceClient`。

> 注：`make wire` 目标与 `wire_gen.go` 生成流程已在本轮接入（见 §6），不再列入待办。
