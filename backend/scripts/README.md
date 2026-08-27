# Script 脚本指南

本目录是后端（`backend/`）的环境准备、Docker 编排与宿主机部署脚本。所有脚本均以 `backend/` 为工作目录，除直接运行外，也可通过 `backend/Makefile` 的封装目标调用（见文末表格）。

## 目录结构

```
scripts/
├── env/                            # 环境准备脚本
│   ├── install_unix_prod.sh        # Unix/Linux 生产环境：基础工具 + Node.js + pm2 + Docker + Go
│   ├── install_unix_dev.sh         # Unix/Linux 开发环境：生产内容 + protoc 代码生成插件
│   ├── install_windows_dev.ps1     # Windows 开发环境：Scoop + Docker Desktop + Go（-SkipDocker / -AutoConfirm 参数）
│   └── lib/                        # 共享函数库（Unix .sh 与 PowerShell .ps1 各一套：日志/hosts/系统检测等）
├── docker/                         # Docker Compose 编排脚本（.sh / .ps1 双版本）
│   ├── full_deploy.sh|ps1          # 全量启动：中间件 + core/admin/app 三个应用服务
│   ├── libs_only.sh|ps1            # 仅启动中间件（本地运行三个服务时配合使用）
│   └── opensearch/Dockerfile       # 自定义 OpenSearch 镜像（安装 smartcn 中文分词插件）；当前两个 compose 文件均未编排 opensearch 服务
└── deploy/
    └── pm2_service.sh              # PM2 托管三个服务的宿主机部署
```

## 环境准备（env/）

| 脚本 | 平台 | 安装内容 |
|------|------|----------|
| `install_unix_prod.sh` | macOS / Ubuntu / Debian / CentOS / RHEL / Rocky / Alma / Fedora | git、wget、jq 等基础工具，Node.js 22.x + pm2，Docker（含 Compose），Golang |
| `install_unix_dev.sh` | 同上 | 生产脚本全部内容 + protoc-gen-go / protoc-gen-go-grpc 等代码生成插件 |
| `install_windows_dev.ps1` | Windows（管理员 PowerShell） | Scoop 包管理器、Docker Desktop、Go 环境与插件 |

脚本幂等，可重复运行。Unix 脚本支持可选的 hosts 初始化（默认关闭）：

```bash
AUTO_INIT_HOSTS=true HOSTS_IP=127.0.0.1 HOSTS_DOMAIN_SUFFIX=.local \
HOSTS_SERVICES="postgres redis" ./scripts/env/install_unix_dev.sh
```

## Docker 编排（docker/）

两个脚本对应 `backend/` 下的两个 compose 文件：

| 脚本 | Compose 文件 | 启动内容 |
|------|--------------|----------|
| `full_deploy.sh` / `full_deploy.ps1` | `docker-compose.yaml` | PostgreSQL、Redis、MinIO、Etcd、Jaeger + admin/app/core 三个应用服务容器 |
| `libs_only.sh` / `libs_only.ps1` | `docker-compose.libs.yaml`（默认，可经 `COMPOSE_FILE` 覆盖） | 仅 PostgreSQL、Redis、MinIO、Etcd、Jaeger |

**full_deploy 前置条件**：compose 中的应用服务引用 `go-wind-oa/{admin,app,core}-service` 镜像，需先在 `backend/` 执行 `make docker`（逐服务构建镜像）再运行。

**可配置项**：

- `APP_ROOT`（Bash）/ `-AppRoot`（PowerShell）— 中间件数据卷根目录，默认 `/root/app` 或 `C:\app`
- `COMPOSE_FILE` / `-ComposeFile` — compose 文件路径，`libs_only` 默认 `docker-compose.libs.yaml`

```bash
# 本地开发（推荐）：Docker 跑中间件，三个服务本地 make run
./scripts/docker/libs_only.sh
cd app/core/service && make run   # admin / app 同理

# 一键全量
make docker && ./scripts/docker/full_deploy.sh
```

## PM2 宿主机部署（deploy/pm2_service.sh）

不使用 Docker 时的部署方式，流程：

1. 加载 `backend/.env`（存在时）
2. `make build_only` 编译三个服务二进制
3. 将 `app/{core,admin,app}/service` 下的 `bin/server` 与 `configs/*.yaml` 安装到 `~/app/go_wind_oa/<service>/`
4. PM2 以 `go_wind_oa` 命名空间托管，进程名 `go_wind_oa-core` / `go_wind_oa-admin` / `go_wind_oa-app`

`PROJECT_NAME` 环境变量可覆盖安装目录与命名空间名。

## Makefile 封装入口

在 `backend/` 目录执行（Windows 下自动调用 .ps1 版本）：

| 命令 | 等效脚本 |
|------|----------|
| `make install-dev` / `make install-prod` | `env/install_unix_dev.sh` / `env/install_unix_prod.sh` |
| `make docker-up` | `docker/full_deploy.*`（全量启动） |
| `make docker-libs` | `docker/libs_only.*`（仅中间件） |
| `make docker-down` | `docker compose down` |
| `make pm2-deploy` | `deploy/pm2_service.sh` |
| `make docker` | 逐服务构建三个应用镜像 |
| `make compose-up` / `compose-up-libs` / `compose-down` | 直接调用 `docker compose`（不经过脚本） |
