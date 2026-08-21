<div align="center">

# GoWind OA｜风行协同办公

**开箱即用的企业级协同办公系统**

> **让协同办公如风般自由 — GoWind OA**

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vuedotjs)](https://vuejs.org/)
[![Flutter](https://img.shields.io/badge/Flutter-3.x-02569B?logo=flutter)](https://flutter.dev/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)](https://www.docker.com/)

[English](./README.en-US.md) | **中文** | [日本語](./README.ja-JP.md)

</div>

---

## 项目简介

GoWind OA 是一套面向企业的**协同办公系统**，覆盖日常办公场景下的审批流转、人事考勤、请假与报销等业务。系统内置一个轻量级工作流引擎作为审批流转的基础能力，并配套管理后台与移动端，实现办公流程的数字化。

后端基于 [go-kratos](https://go-kratos.dev/) 微服务框架，采用 core / admin / app 三服务分离架构；前端管理后台使用 Vue3 + Element Plus，移动端使用 Flutter。

## 功能模块

| 模块 | 说明 | 状态 |
|------|------|------|
| 工作流引擎 | 线性状态机、会签/或签、撤回、主管与职位寻人、业务挂钩 | ✅ |
| 审批中心 | 待办/已办/我发起、详情审批与转办、可视化流程定义编辑器 | ✅ |
| 人事考勤 | GPS/WiFi 打卡、迟到/早退/旷工/请假联动结算、节假日表、每日定时结算 | ✅ |
| 请假管理 | 类型与额度、半日粒度、审批通过自动扣减额度、流程定义自动引导 | ✅ |
| 报销管理 | 多行明细、发票凭证拍照直传、流程定义自动引导 | ✅ |
| 出差 / 加班 / 用印 / 外出 | 四类同型审批单据，建单挂对应 v1 流程（流程定义自动引导），审批终态仅同步状态、无额度副作用 | ✅ |
| 站内信 | 审批通知落库、收件箱查询；SSE 推送仅 admin 自身 SendMessage 路径生效，工作流通知经 core 落库后不触发 SSE | ✅ |
| 公告发布 | 复用站内信 SendMessage 扇出（全员 target_all / 按部门 ListUserIDsByOrgUnitIDs 展开 target_user_ids），无独立表 | ✅ |
| 通讯录 | app 侧只读 wrapper（带 redact 脱敏）+ admin/mobile 双端组织树浏览与成员列表 | ✅ |
| 表单引擎 | 基于字段描述的动态表单渲染（移动端生成、审批端键值对展示） | ✅ |

> 工作流引擎是协同办公系统的一个子系统，用于驱动各类审批流程，并非系统定位本身。

## 技术栈

<table>
<tr><th>层级</th><th>技术</th></tr>
<tr><td><strong>后端框架</strong></td><td><code>Golang</code> · <code>go-kratos v2</code> · <code>Wire</code> · <code>Protobuf / Buf</code></td></tr>
<tr><td><strong>ORM</strong></td><td><code>Ent</code>（含隐私层与多租户隔离） · <code>PostgreSQL</code></td></tr>
<tr><td><strong>中间件</strong></td><td><code>Redis</code> · <code>MinIO</code>（S3 兼容对象存储） · <code>Etcd</code>（服务注册发现） · <code>Jaeger</code>（链路追踪）</td></tr>
<tr><td><strong>认证授权</strong></td><td><code>JWT</code> · <code>RBAC</code> · <code>验证码</code> · 多租户数据隔离</td></tr>
<tr><td><strong>管理后台</strong></td><td><code>Vue 3</code> · <code>TypeScript</code> · <code>Vite</code> · <code>Element Plus</code> · <code>Pinia</code> · <code>TanStack Query</code></td></tr>
<tr><td><strong>移动端</strong></td><td><code>Flutter</code> · <code>Dart</code> · <code>Dio</code> · <code>GetIt</code></td></tr>
<tr><td><strong>代码生成</strong></td><td><code>Ent Schema → ORM</code> · <code>Protobuf → Go API / TypeScript / Dart 客户端 / OpenAPI</code> · <code>Wire 依赖注入</code></td></tr>
<tr><td><strong>部署运维</strong></td><td><code>Docker</code> · <code>Docker Compose</code> · <code>Swagger UI</code></td></tr>
</table>

## 仓库结构

```text
go-wind-oa/
├── backend/                   # 协同办公系统后端
│   ├── api/                   # Protobuf 定义与代码生成模板
│   │   ├── protos/            # 各业务域 proto（oa / internal_message / authentication / identity 等）
│   │   └── gen/               # buf 生成的 Go 桩、OpenAPI、TS/Dart 客户端
│   ├── app/                   # 三服务主目录
│   │   ├── core/service/      # 纯 gRPC：业务逻辑与数据持久化
│   │   ├── admin/service/     # HTTP 边端：管理后台鉴权与业务转发
│   │   └── app/service/       # HTTP 边端：移动端鉴权与业务转发
│   ├── pkg/                   # 自包含基座（middleware / crypto / viewer / oss 等）
│   └── scripts/               # 部署与运维脚本
├── frontend/
│   ├── admin/                 # 管理后台（Vue3 + Element Plus）
│   └── mobile/                # 移动端（Flutter）
└── docs/                      # 设计文档
```

## 架构概览

三服务分离，各司其职：

- **core-service**：纯 gRPC 后端，持有 ent 仓库与业务逻辑实现。注册各业务域 gRPC 服务（工作流、考勤、请假、报销、站内信等）。中间件 `logging + ent`（隐私层基于 viewer 上下文做租户隔离）。
- **admin-service**：HTTP 边端，为管理后台暴露鉴权与业务 HTTP 端点，各方法为转发层经 gRPC 客户端调用 core-service。中间件 `logging → auth+authz(白名单) → ent`，auth 必须在 ent 之前（顺序颠倒则 ent 兜底 SystemViewer，租户隔离失效）。
- **app-service**：HTTP 边端，为移动端暴露鉴权与业务 HTTP 端点，转发层与中间件顺序同 admin。

Proto 域分离：core 的 `oa/service/v1/*.proto` 为纯 gRPC（HTTP 注解剥离）；HTTP 路由注解定义在 `admin/service/v1/i_*.proto` 与 `app/service/v1/i_*.proto` wrapper proto 中，引用 `oa.service.v1` 消息类型。

详见 [docs/oa-workflow-design.md](./docs/oa-workflow-design.md)。

## 后端构建

每个 service 目录下都有 Makefile（`include ../../../app.mk`），`SERVICE_NAME` 决定 buf openapi 模板选择：

```bash
cd backend/app/core/service && make ent wire api build   # core 无 openapi（纯 gRPC）
cd backend/app/admin/service && make openapi wire api build
cd backend/app/app/service && make openapi wire api build
```

`make ent` 生成 ent ORM（`--feature privacy` 不可省，是 `TenantPrivacy` 策略生效前提）。`make api` 生成 Go proto 桩。`make openapi` 按 service 选择 `buf.admin.openapi.gen.yaml` / `buf.app.openapi.gen.yaml`（core 跳过）。`make wire` 生成 `wire_gen.go`。

> 详细的后端环境准备、项目结构、部署流程，见 [backend/README.md](./backend/README.md)。

## 管理后台

Vue3 + Vite + TypeScript + Element Plus + Pinia + TanStack Query。OA 模块代码：

- `src/api/composables/oa.ts` — TanStack Query hooks，封装生成的 `apiClient` 各业务服务。
- `src/api/generated/admin/service/v1/index.ts` — 由 `backend/api/buf.admin.typescript.gen.yaml` 生成。
- `src/pages/app/oa/` — 审批中心、考勤记录、节假日设置、请假/报销管理、出差/加班/用印/外出等同型单据管理、公告发布、通讯录、流程定义编辑器。
- `src/router/routes/modules/app/oa.ts` — 前端路由模块（自动 glob 注册）。

```bash
cd backend/api && buf generate --template buf.admin.typescript.gen.yaml   # 生成 TS 客户端
cd frontend/admin && pnpm i && pnpm dev
```

## 移动端

Flutter + Dio + GetIt。OA feature：

- `lib/src/features/oa/services/` — 各业务服务（工作流、考勤、请假、报销、出差/加班/用印/外出等同型单据、站内信通知、通讯录、文件上传）。完整清单见 [docs/oa-mobile-design.md](./docs/oa-mobile-design.md) §2/§4。
- `lib/src/features/oa/pages/` — 登录、审批任务列表与详情、通用申请（动态表单）、各业务单据提交页、考勤打卡、通知、通讯录。详见同上文档 §3/§4。
- `lib/generated/api/app/service/v1/index.dart` — 由 `backend/api/buf.app.dart.gen.yaml` 生成。

```bash
cd backend/api && buf generate --template buf.app.dart.gen.yaml   # 生成 Dart 客户端
cd frontend/mobile && flutter run
```

## 开发环境

所需基础工具与中间件，以及 Docker Compose 一键启动方式，见 [backend/README.md](./backend/README.md) 的「前置环境要求」与「中间件」章节。

## 相关文档

- [后端架构设计](./docs/oa-workflow-design.md)
- [移动端设计](./docs/oa-mobile-design.md)
- [后端 README](./backend/README.md)

## License

[MIT](./LICENSE)
