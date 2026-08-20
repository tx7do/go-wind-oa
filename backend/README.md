<div align="center">

# GoWind OA Backend｜风行协同办公 · 后端

**协同办公系统后端服务**

</div>

---

## 技术栈

<table>
<tr><th>层级</th><th>技术</th></tr>
<tr><td><strong>后端框架</strong></td><td><code>Golang</code> · <code>go-kratos v2</code> · <code>Wire</code> · <code>Protobuf / Buf</code></td></tr>
<tr><td><strong>ORM</strong></td><td><code>Ent</code>（含隐私层与多租户隔离） · <code>PostgreSQL</code></td></tr>
<tr><td><strong>中间件</strong></td><td><code>Redis</code> · <code>MinIO</code>（S3 兼容对象存储） · <code>Etcd</code>（服务注册发现） · <code>Jaeger</code>（链路追踪）</td></tr>
<tr><td><strong>认证授权</strong></td><td><code>JWT</code> · <code>RBAC</code> · <code>验证码</code> · 多租户数据隔离</td></tr>
<tr><td><strong>代码生成</strong></td><td><code>Ent Schema → ORM</code> · <code>Protobuf → Go API / TypeScript / Dart 客户端 / OpenAPI</code> · <code>Wire 依赖注入</code></td></tr>
</table>

## 前置环境要求

### 基础环境

| 工具 | 最低版本 | 说明 |
|------|----------|------|
| [Go](https://go.dev/) | 1.22 | 编译后端服务 |
| [Buf](https://buf.build/) | 最新 | Protobuf API 巷道生成 |
| [Wire](https://github.com/google/wire) | 最新 | 依赖注入代码生成 |
| [Docker](https://www.docker.com/) | 最新 | 容器化部署 |
| [Make](https://www.gnu.org/software/make/) | 最新 | 构建脚本执行 |
| [Node.js](https://nodejs.org/) | 18+ | 管理后台构建 |
| [pnpm](https://pnpm.io/) | 最新 | 管理后台依赖管理 |
| [Flutter](https://flutter.dev/) | 3.x | 移动端构建 |

### 中间件

| 中间件 | 用途 | 默认端口 |
|--------|------|----------|
| [PostgreSQL](https://www.postgresql.org/) | 业务数据持久化 | 5432 |
| [Redis](https://redis.io/) | 缓存与会话存储 | 6379 |
| [MinIO](https://min.io/) | S3 兼容对象存储（发票凭证等文件） | 9000 / 9001 |
| [Etcd](https://etcd.io/) | 服务注册与发现 | 2379 |
| [Jaeger](https://www.jaegertracing.io/) | 分布式链路追踪 | 14268 / 16686 |

> 上述中间件可通过仓库根目录的 `docker-compose.yaml` 一键启动，无需逐一手动安装。

## 内置API文档

后端服务内置 Swagger UI 与 OpenAPI 规范文件，启动对应服务后可访问。

### Swagger UI

- [Admin Swagger UI](http://localhost:6600/docs/)
- [App Swagger UI](http://localhost:6700/docs/)

### openapi.yaml

- [Admin openapi.yaml](http://localhost:6600/docs/openapi.yaml)
- [App openapi.yaml](http://localhost:6700/docs/openapi.yaml)

> core-service 为纯 gRPC 后端，不暴露 HTTP 文档，故无 Swagger 入口。

## 项目目录结构

```text
backend/
├── api/                            # Protobuf 定义与代码生成巷道
│   ├── protos/                     # 各业务域 proto 定义
│   │   ├── oa/service/v1/          # 协同办公域：workflow / attendance / leave / expense / oa_error
│   │   ├── admin/service/v1/       # 管理后台 HTTP wrapper proto
│   │   ├── app/service/v1/         # 移动端 HTTP wrapper proto
│   │   ├── authentication/         # 认证域
│   │   ├── identity/               # 身份域（用户/组织/职位）
│   │   ├── internal_message/       # 站内信域
│   │   └── ...                     # 其余继承自 CMS 基座的业务域
│   ├── gen/                        # buf 生成的桩代码
│   │   └── go/                     # 各 service 的 *.pb.go / *_grpc.pb.go / *_http.pb.go
│   ├── buf.gen.yaml                # Go proto 生成配置
│   ├── buf.admin.openapi.gen.yaml  # 管理后台 OpenAPI 生成配置
│   ├── buf.app.openapi.gen.yaml    # 移动端 OpenAPI 生成配置
│   ├── buf.admin.typescript.gen.yaml
│   ├── buf.app.dart.gen.yaml
│   └── buf.yaml
├── app/                            # 三服务主目录
│   ├── core/service/               # 纯 gRPC：业务逻辑与数据持久化
│   │   ├── internal/
│   │   │   ├── biz/                # 业务逻辑层
│   │   │   ├── data/               # ent 仓库层 + 隐私层
│   │   │   ├── service/            # gRPC 服务实现
│   │   │   └── server/             # kratos 服务装配
│   │   └── main.go
│   ├── admin/service/              # HTTP 边端：管理后台
│   ├── app/service/                # HTTP 边端：移动端
│   └── app.mk                      # 共享 Makefile 片段（各 service include）
├── pkg/                            # 自包含基座
│   ├── middleware/                 # 鉴权、日志、租户等中间件
│   ├── crypto/                     # 加密工具
│   ├── viewer/                     # viewer 上下文（隐私层依赖）
│   └── ...
├── scripts/                        # 部署与运维脚本
├── sql/                            # 数据库初始化脚本
├── Dockerfile
└── Makefile
```

## 三服务架构

协同办公系统后端采用 core / admin / app 三服务分离架构，各服务职责如下：

| 服务 | 协议 | 职责 | 中间件顺序 |
|------|------|------|------------|
| core-service | 纯 gRPC | 业务逻辑实现、ent 仓库持久化 | `logging → ent` |
| admin-service | HTTP | 管理后台鉴权与业务转发 | `logging → auth + authz(白名单) → ent` |
| app-service | HTTP | 移动端鉴权与业务转发 | `logging → auth + authz(白名单) → ent` |

> **中间件顺序约束**：auth 必须位于 ent 之前。顺序颠倒则 ent 兜底 SystemViewer，导致租户隔离失效。

Proto 域分离：core 的 `oa/service/v1/*.proto` 为纯 gRPC（HTTP 注解已剥离）；HTTP 路由注解定义在 `admin/service/v1/i_*.proto` 与 `app/service/v1/i_*.proto` wrapper proto 中，引用 `oa.service.v1` 消息类型。详见 [docs/oa-workflow-design.md](../docs/oa-workflow-design.md)。

## 代码生成

### 生成 Protobuf API

本项目使用 [buf.build](https://buf.build/) 进行 Protobuf API 巷道生成。相关命令行工具和插件的安装方法参见 [Kratos 微服务框架 API 工程化指南](https://juejin.cn/post/7191095845096259641)。

#### 生成 Go 代码

```bash
cd backend/api
buf generate
```

#### 生成 TypeScript 客户端

```bash
cd backend/api
buf generate --template buf.admin.typescript.gen.yaml
```

#### 生成 Dart 客户端

```bash
cd backend/api
buf generate --template buf.app.dart.gen.yaml
```

#### 生成 OpenAPI v3 文档

```bash
cd backend/api
buf generate --template buf.admin.openapi.gen.yaml   # 管理后台
buf generate --template buf.app.openapi.gen.yaml     # 移动端
```

### 生成 Ent 代码

```bash
cd backend/app/{core|admin|app}/service
make ent
```

> `make ent` 必须带 `--feature privacy` 参数，是 `TenantPrivacy` 策略生效的前提。

### 生成 Wire 代码

```bash
cd backend/app/{core|admin|app}/service
make wire
```

## 构建与运行

```bash
cd backend/app/{core|admin|app}/service
make build    # 构建二进制
make run      # 调试运行
```

### 构建顺序约束

core-service 依赖 ent 仓库与 wire 依赖注入，admin/app-service 依赖 core 的 gRPC 桩与自身的 HTTP 桩。完整构建顺序：

```bash
# 1. 先生成 proto 桩（core + admin + app 三端）
cd backend/api && buf generate

# 2. core 生成 ent 与 wire（admin/app 不持有 ent 仓库）
cd backend/app/core/service && make ent wire

# 3. admin / app 生成 wire
cd backend/app/admin/service && make wire
cd backend/app/app/service && make wire

# 4. 构建各服务
cd backend/app/core/service && make build
cd backend/app/admin/service && make build
cd backend/app/app/service && make build
```

### OpenAPI 文档生成

`make openapi` 脚本按 `SERVICE_NAME` 选择 `buf.{admin|app}.openapi.gen.yaml`（core-service 为纯 gRPC，跳过）：

```bash
cd backend/app/admin/service && make openapi   # 生成 admin OpenAPI
cd backend/app/app/service && make openapi     # 生成 app OpenAPI
```

## Docker 部署

```bash
make docker    # 构建 Docker 镜像
```

## 相关文档

- [协同办公系统设计](../docs/oa-workflow-design.md)
- [移动端设计](../docs/oa-mobile-design.md)
- [项目 README](../README.md)
