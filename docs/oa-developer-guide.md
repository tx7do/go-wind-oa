# GoWind OA 开发者教程：新增一类审批业务

本教程以新增「**采购申请（PURCHASE）**」为例，走完从 proto 定义到三端可用的完整链路。现有的出差/加班/用印/外出四类单据与采购申请完全同构，做每一步时都有现成参照文件可抄。

架构背景见 [后端架构设计](./oa-workflow-design.md)，引擎的节点类型与状态机细节也在那篇文档里，本篇不重复。

## 全景：一次扩展要动哪些地方

| 层 | 文件 | 参照 |
|----|------|------|
| core proto | `backend/api/protos/oa/service/v1/purchase.proto` | `business_trip.proto` |
| ent schema | `backend/app/core/service/internal/data/ent/schema/purchase_application.go` | `business_trip.go`（同目录） |
| core 业务实现 | `backend/app/core/service/internal/{data,service}/purchase_*.go` | `business_trip_repo.go` / `business_trip_service.go` |
| admin wrapper proto | `backend/api/protos/admin/service/v1/i_purchase.proto` | `i_business_trip.proto` |
| app wrapper proto | `backend/api/protos/app/service/v1/i_purchase.proto` | 同上 |
| 边端转发 service | `backend/app/{admin,app}/service/internal/service/purchase_service.go` | 各自目录下的 `business_trip_service.go` |
| 服务发现客户端 | `backend/app/{admin,app}/service/internal/data/data.go` | `NewBusinessTripServiceClient` |
| 依赖装配 | `backend/app/{core,admin,app}/service/cmd/server/wiring.go` | 手写接线（无 wire/注入框架） |
| 管理后台 | `frontend/admin/src/` 下 composable、页面、路由、i18n | business_trip 同名文件 |
| 移动端 | `frontend/mobile/lib/src/features/oa/` 下 service、页面、路由、l10n | business_trip 同名文件 |
| 演示数据 | `backend/sql/postgresql-demo-data.sql` | OA 各表 INSERT 段 |

## 代码生成链总览

改完 proto / ent schema 后，按这个顺序生成与构建：

```bash
# 1. proto → Go 桩（core + admin + app 三端一起生成）
cd backend/api && buf generate

# 2. ent schema → ORM（core 独有；--feature privacy 不可省，是租户隐私层生效前提）
cd backend/app/core/service && make ent

# 3. 逐个构建验证
cd backend/app/core/service  && make build
cd backend/app/admin/service && make build
cd backend/app/app/service   && make build

# 4.（可选）重新生成客户端与 OpenAPI
cd backend/api && buf generate --template buf.admin.typescript.gen.yaml   # 管理后台 TS 客户端
cd backend/api && buf generate --template buf.app.dart.gen.yaml           # 移动端 Dart 客户端
cd backend/app/admin/service && make openapi                              # Swagger 文档
cd backend/app/app/service   && make openapi
```

构建顺序约束：core 依赖 ent，admin/app 依赖 core 的 gRPC 桩——先 buf、再 ent、再构建。

## 第一步：core proto

新建 `backend/api/protos/oa/service/v1/purchase.proto`，参照 `business_trip.proto` 的结构：

- 消息：`PurchaseApplication`（单据本体）、`SubmitPurchaseApplicationRequest/Response`、`ListPurchaseApplicationsRequest/Response`、`GetPurchaseApplicationRequest`。
- service 三个方法，与同类单据一致：`SubmitPurchaseApplicationApplication` 提交、`ListPurchaseApplications` 我的申请列表、`GetPurchaseApplication` 详情。

两个容易踩的 proto 细节：

- **半日/枚举类可选字段要显式 `optional`**：proto3 里非 optional 字段缺省值与「没传」无法区分（历史上半天请假被算成 0.5 天就是这个原因）。
- **蛇形 json_name 要显式声明**：`tenant_code` 这类想以蛇形出现在 HTTP JSON 里的字段，需显式 `json_name = "tenant_code"`，否则客户端按驼峰发送会被静默忽略。

## 第二步：ent schema

新建 `purchase_application.go`，同型单据的表结构模式：

- 租户字段（隐私层 `TenantPrivacy` 依据 viewer 上下文自动隔离，schema 里按 `business_trip.go` 抄即可）；
- 业务字段 + **四态状态枚举**（`PENDING / APPROVED / REJECTED / WITHDRAWN`）；
- `instance_id` 关联流程实例；
- O2M 明细边（如有）。

> **大坑：O2M 的 `edge.To(...)` 不要加 `.Required()`**。加了会阻断父记录创建（没有子记录就不让建父记录），历史上流程定义、请假、报销三处都栽过。

```bash
cd backend/app/core/service && make ent
```

## 第三步：core 业务实现

### repo 与 service

参照 `business_trip_repo.go` / `business_trip_service.go`。要点：

- **提交（Submit）**：写单据（PENDING）→ `ensureWorkflowDefinition` 兜底流程定义 → 启动流程实例（`form_data` 透传落盘）→ 回写单据 `instance_id`。三步在同一事务。
- **列表/详情**：repo 谓词必须是 `tenant_id = 调用方租户 AND user_id = 调用方用户` 的「我的」语义，租户与用户身份从 viewer 上下文取（`callerFromContext`，uid 或 tid 为 0 一律 fail-closed）。
- **ensureWorkflowDefinition**：租户内不存在 `PURCHASE` 定义时按默认模板创建并启用（提交给申请人主管 `LEADER`，会签 `ALL`）。抄 `leave_service.go` 的 `ensureWorkflowDefinition`。

### 注册业务挂钩

流程实例到终态时要回写单据状态，通过 `WorkflowEventRegistry` 注册回调（APPROVED / REJECTED / WITHDRAWN 三个终态同步回调）。回调里必须做两件事，否则会出数据一致性问题：

1. **校验单据上的 `instance_id` 与回调的实例一致**（防伪造）；
2. **只处理仍处于 PENDING 的单据**（幂等，防止重复回调覆盖终态）。

请假（有额度扣减副作用）和报销（纯状态同步）的挂钩实现都在各自 service 文件里，采购申请抄报销那种「仅同步状态」的即可。

### 装配

core 的依赖装配是手写的：`backend/app/core/service/cmd/server/wiring.go` 里 new 出 repo 与 service 并注册 gRPC。照着 `BusinessTripService` 的接线抄三行即可。

## 第四步：HTTP 边端（admin / app wrapper)

core 是纯 gRPC，HTTP 路由全部定义在两个边端的 wrapper proto 里：

1. 新建 `backend/api/protos/admin/service/v1/i_purchase.proto` 与 `app/service/v1/i_purchase.proto`，引用 `oa.service.v1` 的消息类型，用 `google.api.http` 注解声明路由（如 `GET /admin/v1/oa/purchase/applications`、`GET /admin/v1/oa/purchase/applications/{id}`、`POST .../applications`），照抄 `i_business_trip.proto`。
2. 两个边端各写转发层 `internal/service/purchase_service.go`：方法体就是「取请求 → 调 core gRPC client → 返回」，不含业务逻辑。
3. `internal/data/data.go` 加 `NewPurchaseServiceClient`（经 etcd discovery 构造 core 客户端），照抄 `NewBusinessTripServiceClient`。
4. `cmd/server/wiring.go` 把 client 传给转发 service、把 service 注册进 HTTP server——三处照抄 business_trip 的接线。
5. `make openapi` 重新生成两端 Swagger。

## 第五步：管理后台前端

1. 重新生成 TS 客户端：`cd backend/api && buf generate --template buf.admin.typescript.gen.yaml`（输出到 `frontend/admin/src/api/generated/`）。
2. `src/api/composables/oa.ts` 加 TanStack Query hooks（list / get / submit），照抄 business_trip 的 hooks。
3. 新页面 `src/pages/app/oa/purchase/index.vue`：ProTable 只读列表 + 状态 Tag，抄 `business_trip/index.vue`。
4. 路由 `src/router/routes/modules/app/oa.ts` 加一条（title 走 i18n key，如 `routes.oaPurchase`）。
5. `src/locales/zh-CN/routes.json` 与 `en-US/routes.json` 补标题文案。

> 前端约定：表格列定义里每列都要有唯一 `prop`（映射到 vxe 表格 `field`，同时也是 Vue 的 key），不写会触发 vxe 告警。枚举值显示一律走 i18n（`enum.<域>.<枚举名>.<值>` 命名空间），不要硬编码英文枚举名；动态键用 `$t`，字面量键用 `useI18n().t`。状态标签用 `oa.ts` 共享的 `workflowInstanceStatusToName/ToTag`，搜索下拉用 `workflowInstanceStatusOptions`；日期一律走 `@/utils/date` 的 `formatDate/formatDateTime`（直接字符串切片有时区错位）。列表接真分页（ProPage `pagination: true` + listAction 透传 `page/pageSize`，total 用后端返回值）；用户选择的场景复用 `fetchUsers` + `userDisplayName`（列表页与抽屉之间经 prop 共享，避免重复拉取）。

## 第六步：移动端

1. 重新生成 Dart 客户端：`cd backend/api && buf generate --template buf.app.dart.gen.yaml`（输出到 `frontend/mobile/lib/generated/api/`）。
2. 新 service `lib/src/features/oa/services/purchase_service.dart`，照抄 `business_trip_service.dart`（走全局 Dio，自动带令牌与刷新拦截器）。
3. 新页面 `lib/src/features/oa/pages/purchase/oa_purchase_page.dart`：表单 + 提交 + 「我的申请」列表，照抄 business_trip 页面。
4. 注册路由三件套：`lib/src/core/constants/router_paths.dart`（路径常量）、`lib/src/core/app_router/route_names.dart`（路由名）、`lib/src/app_router/app_router.dart`（路由表）；全屏页不在底部 Shell 内。
5. 文案进 `lib/l10n/app_zh.arb` 与 `app_en.arb`（重新生成 `lib/generated/l10n.dart`）。枚举显示用 ARB 键，不要硬编码。

## 第七步：演示数据

`backend/sql/postgresql-demo-data.sql` 里给 `oa_purchase_application` 补一个 INSERT 段，注意两条铁律：

- **跨引用自洽**：`instance_id` 要指向同租户、同演示批次的流程实例行；`tenant_id` / `created_by` / `assignee_user_id` 取值与所在演示组一致（租户组 tid=1 用户 2/3，超管组 tid=0 用户 1）。
- **幂等可重跑**：脚本靠开头的 `TRUNCATE ... RESTART IDENTITY CASCADE` 保证重复导入不冲突，新表也要加进该列表。

## 验证清单

- [ ] `buf generate` 三模板全部无错（Go / TS / Dart）
- [ ] 三服务 `make build` 通过；`go vet ./...` 无新告警
- [ ] admin Swagger（`/docs/`）能看到新路由
- [ ] 提交一单 → 待办出现在审批人 → 通过 → 单据状态 APPROVED；驳回与撤回两条路径同样过一遍
- [ ] 换一个租户的账号看不到这单（租户隔离）
- [ ] 管理后台列表分页正常；移动端提交 → 管理后台审批中心可见 → 详情表单数据完整

## 高频坑位速查

| 坑 | 症状 | 规避 |
|----|------|------|
| O2M edge 加 `.Required()` | 父记录根本建不出来 | 同型抄现成 schema，别随手加约束 |
| proto3 可选字段没写 `optional` | 不传的字段被当成零值（半天假变 0.5 天那类） | 语义上「可选」的一律显式 `optional` |
| protojson FieldMask 大小写 | 更新接口 400 | FieldMask 路径必须是 camelCase（`definitionStatus`），蛇形会解析失败 |
| `Timestamp.AsTime()` 恒为 UTC | 日期差一天 / 覆盖判断错位 | 请求里的时间戳先 `.In(time.Local)` 再按天截断 |
| 外部工具 import ent 后 panic | 枚举校验器为 nil | 必须 blank import `_ ".../ent/runtime"` |
| 通知丢了 viewer | SendMessage 报「sender identity is required」 | 异步通知用 `context.WithoutCancel(ctx)` 保留 viewer 值 |
| 忘了服务发现 | admin/app 找不到 core | 新 gRPC 服务必须经 etcd 注册（wiring 里接线），边端用 discovery 客户端 |
| 前端 vxe 表格列无 prop | 控制台告警、列状态异常 | 每列唯一 `prop` / `field` |
| curl 冒烟带中文 | Git Bash 下 UTF-8 损坏 | 冒烟请求体用 ASCII 或走客户端 |
| repo List 只 Count 不切片 | 列表假分页：total 正确但返回全量 | `Offset/Limit` 只在传了分页参数时生效（不传=不限页），total 用 Clone().Count |
| proto3 枚举 0 值当过滤条件 | 「筛第一档枚举」恒返回全部 | 过滤枚举字段声明 `optional`，服务端按存在性判断（六模块 status 即此修法） |
| TenantPrivacy 覆盖写入 | 测试里种跨租户数据悄悄落到本租户 | 普通用户上下文 Create 会强制 tenant_id=viewer 租户；种跨租户数据用 system viewer |
| 测试里 import ent 后 hook 未初始化 | `uninitialized hook (forgotten import ent/runtime?)` | 测试文件 blank import `_ ".../ent/runtime"`；sqlite DSN 另需 `?_fk=1` |
