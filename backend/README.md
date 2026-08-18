# go-wind-oa Backend

go-wind-oa 后端 — OA 工作流审批系统，基于 [go-kratos](https://go-kratos.dev/) 微服务框架，采用 core/admin/app 三 service 分離架構。

- **core-service**：纯 gRPC，持有工作流引擎与站內信实现（无 HTTP 端点）。
- **admin-service** / **app-service**：HTTP 边端转发层，分别为 admin 前端与移动端暴露 HTTP 端点，经 gRPC 客户端转发至 core-service。

后端架构详见根目录 [`../README.md`](../README.md) 与 [`../docs/oa-workflow-design.md`](../docs/oa-workflow-design.md)。

## 技术栈

- [Kratos](https://go-kratos.dev/) -- 微服务框架
- [Consul](https://www.consul.io/) / [Etcd](https://etcd.io/) -- 服务发现和配置管理
- [OpenTelemetry](https://opentelemetry.io/) -- 分布式可观察系统
- [Wire](https://github.com/google/wire) -- 依赖注入框架
- [OpenAPI](https://www.openapis.org/) -- RESTful API 文档
- [Redis](https://redis.io/) -- 非关系型数据库
- [PostgreSQL](https://www.postgresql.org/) / [MySQL](https://www.mysql.com/) -- 关系型数据库
- [Ent](https://entgo.io/) -- Golang ORM 框架

## API文档

### Swagger UI

- [Admin Swagger UI](http://localhost:9700/docs/)
- [App Swagger UI](http://localhost:9800/docs/)

> core-service 无 HTTP 端点，无 Swagger UI。

### openapi.yaml

- [Admin openapi.yaml](http://localhost:9700/docs/openapi.yaml)
- [App openapi.yaml](http://localhost:9800/docs/openapi.yaml)

## 生成Protobuf API

本项目使用 [buf.build](https://buf.build/) 进行 Protobuf API 构建。

### 生成GO代码

```bash
cd `{项目根目录}/backend/app/{服务名}/service`
make api
```

### 生成OpenAPI v3文档

```bash
cd `{项目根目录}/backend/app/{服务名}/service`
make openapi
```

> core-service 无 HTTP 端点，`make openapi` 跳过。

### 生成 Admin TypeScript 客户端

```bash
cd `{项目根目录}/backend/api`
buf generate --template buf.admin.typescript.gen.yaml
```

### 生成 Mobile Dart 客户端

```bash
cd `{项目根目录}/backend/api`
buf generate --template buf.flutter.oa.dart.gen.yaml
```

## 其他代码生成

### 生成ent代码

```bash
cd `{项目根目录}/backend/app/{服务名}/service`
make ent
```

### 生成wire代码

```bash
cd `{项目根目录}/backend/app/{服务名}/service`
make wire
```

## 构建程序

```bash
cd `{项目根目录}/backend/app/{服务名}/service`
make build
```

### 调试运行

```bash
cd `{项目根目录}/backend/app/{服务名}/service`
make run
```

## Docker部署

### 构建Docker镜像

```bash
cd `{项目根目录}/backend/app/{服务名}/service`
make docker
```
