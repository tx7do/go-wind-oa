# GoWind OA 快速上手教程

本教程带你从零把整套系统跑起来：中间件 → 三个后端服务 → 导入演示数据 → 管理后台与移动端登录 → 用演示数据走一遍核心功能。全程约 15~30 分钟。

## 你将部署什么

| 组件 | 说明 | 地址 |
|------|------|------|
| 中间件容器 | PostgreSQL / Redis / MinIO / etcd / Jaeger，一条命令全部启动 | 见下文端口表 |
| core-service | 纯 gRPC 后端，全部业务逻辑在这里（端口随机，其他服务通过 etcd 自动发现它） | — |
| admin-service | 管理后台的 HTTP 后端 | <http://localhost:6600>（API）/ <http://localhost:6600/docs/>（Swagger） |
| app-service | 移动端的 HTTP 后端 | <http://localhost:6700>（API）/ <http://localhost:6700/docs/>（Swagger） |
| 管理后台前端 | Vue3 网页 | <http://localhost:5777> |
| 移动端 App | Flutter | 连接 `localhost:6700` |

其他端口：admin gRPC 与 SSE 事件流 `6601`（SSE 路径 `/events`），app gRPC 与 SSE `6701`，PostgreSQL `5432`，Redis `6379`，MinIO `9000`（API）/ `9001`（控制台），etcd `2379`（服务发现），Jaeger UI `16686`。

## 演示账号

| 账号 | 密码 | 身份 | 怎么登录 |
|------|------|------|----------|
| `admin` | `admin` | 平台超级管理员（租户 0） | 网页端「租户编号」**留空**；移动端直接输账号密码 |
| `tenant_admin` | `admin` | 测试租户「super」的管理员 | 网页端「租户编号」填 `super`；移动端暂不能登录（见「已知限制」） |
| `zhangsan` | `admin` | 测试租户里的普通员工 | 同上，仅网页端 |

说明：

- 网页端登录需要输入图片验证码，看不清点图片刷新。
- `admin` 是服务启动时自动种子的，不在演示 SQL 里；`tenant_admin` / `zhangsan` 和全部 OA 业务演示数据来自演示 SQL。
- 移动端当前版本的登录请求固定走平台租户，所以只有 `admin` 能登录 App。

## 1. 准备环境

| 工具 | 版本要求 | 用途 |
|------|----------|------|
| [Docker](https://www.docker.com/) | 较新版本 | 启动中间件 |
| [Go](https://go.dev/) | 1.22+ | 运行三个后端服务 |
| [Node.js](https://nodejs.org/) | 18+ | 运行管理后台前端 |
| [pnpm](https://pnpm.io/) | 最新 | 前端依赖管理 |
| [Flutter](https://flutter.dev/) | 3.x | 运行移动端（可跳过，先体验网页端） |

只在改代码、重新生成客户端时才需要 [Buf](https://buf.build/) 和 GNU Make，本教程跑通用不到。

## 2. 启动中间件

```bash
cd backend
docker compose -f docker-compose.libs.yaml up -d
docker compose -f docker-compose.libs.yaml ps   # 确认 5 个容器都是 running
```

这一步会启动：PostgreSQL（数据库 `go_wind_oa`，账号 `postgres`，密码 `*Abcd123456`）、Redis、MinIO（发票等文件存储，控制台 <http://localhost:9001>，账号 `root`）、etcd（服务发现，三个后端服务靠它互相找到对方）、Jaeger（链路追踪，UI <http://localhost:16686>）。

## 3. 启动三个后端服务

### 为什么需要配置副本

配置文件里的数据库和 Redis 地址写的是 Docker 网络内的主机名（`postgres`、`redis`），在宿主机上直接跑解析不了。所以给每个服务拷一份配置，把主机名改成 `localhost`，启动时指向副本——仓库里的原始配置保持不动。

### 准备三份配置副本

```bash
# core
cd backend/app/core/service
cp -r configs configs.local
sed -i 's/host=postgres/host=localhost/; s/"redis:6379"/"localhost:6379"/' configs.local/data.yaml

# admin
cd ../../admin/service
cp -r configs configs.local
sed -i 's/"redis:6379"/"localhost:6379"/' configs.local/data.yaml

# app
cd ../../app/service
cp -r configs configs.local
sed -i 's/"redis:6379"/"localhost:6379"/' configs.local/data.yaml
```

MinIO 和 etcd 的地址配置本来就是 `127.0.0.1`，不用改。`configs.local/` 已加入 `.gitignore`，不会被误提交。

### 按顺序启动

```bash
# 1. core 先启动：首次启动会自动建全部表（migrate: true），并种子平台超管 admin/admin
cd backend/app/core/service
go run ./cmd/server --conf ./configs.local

# 2. 再启动 admin（新开一个终端）
cd backend/app/admin/service
go run ./cmd/server --conf ./configs.local

# 3. 最后启动 app（再开一个终端）
cd backend/app/app/service
go run ./cmd/server --conf ./configs.local
```

两个注意点：

- **必须是 `--conf`（双横线）**。写成 `-conf` 会被命令行解析器拆成 `-c onf`，服务读到错误的配置目录。
- 启动顺序有讲究：admin / app 要通过 etcd 发现 core 的 gRPC 地址，core 没起来它们会连不上业务接口。

### 验证

浏览器打开 <http://localhost:6600/docs/>，能看到管理后台的 Swagger 文档即成功。日志里出现 fatal / connection refused 时，优先检查第 2 步的容器是否都在运行。

## 4. 导入演示数据

等 core 至少完整启动过一次（表已自动建好）之后，导入演示数据：

```bash
cd backend
docker compose -f docker-compose.libs.yaml exec -T postgres \
  psql -U postgres -d go_wind_oa < sql/postgresql-demo-data.sql
```

演示数据包含：测试租户「super」（编码 `super`）+ 两个用户（`tenant_admin`、`zhangsan`）+ 完整组织架构树 + OA 全部业务表的示例数据（流程定义、进行中的流程实例与待办任务、考勤记录与设置、请假类型与额度、报销、出差/加班/用印/外出单据等）。

> 演示 SQL 会**先清空全部业务表再插入**，只能用于开发/演示库，切勿对生产库执行。

## 5. 启动管理后台

```bash
cd frontend/admin
pnpm i
pnpm dev
```

浏览器打开 <http://localhost:5777>，登录：

- 平台超管：租户编号**留空**，账号 `admin`，密码 `admin`
- 租户管理员：租户编号填 `super`，账号 `tenant_admin`，密码 `admin`

具体每个页面怎么用，见 [管理后台使用教程](./oa-admin-user-guide.md)。

> 端口与联动关系：前端开发服务器跑在 `5777`，这个端口必须出现在 admin 服务的 CORS 白名单里（`backend/app/admin/service/configs/server.yaml` 的 `cors.origins`，仓库默认已配好 `5777`）。如果你改了前端端口，记得同步改 CORS 配置并重启 admin 服务。

## 6. 启动移动端

移动端通过 `frontend/mobile/.env` 里的 `API_BASE_URL` 找后端：

| 运行方式 | `API_BASE_URL` 应填 |
|----------|---------------------|
| Windows/macOS 直接跑 | `http://localhost:6700`（默认值） |
| Android 模拟器 | `http://10.0.2.2:6700`（模拟器里 localhost 指模拟器自己） |
| 手机真机 | `http://<电脑局域网IP>:6700`，并放行防火墙 |

```bash
cd frontend/mobile
flutter run
```

用 `admin` / `admin` 登录。界面操作说明见 [移动端使用教程](./oa-mobile-user-guide.md)。

## 7. 十分钟体验路径

导入演示数据后，按下面顺序能把主要功能走一遍：

1. **网页端审批**：以 `tenant_admin`（租户 `super`）登录 → 「审批中心」→「待我审批」里有一条演示请假申请 → 点「审批」→ 填意见、点「通过」→ 切到「已办」查看。
2. **流程定义可视化编辑**：「流程定义」→ 任一条点「查看」，或「新建」体验画布编辑：拖节点、连条件分支、配会签/或签。
3. **考勤**：「考勤记录」选日期查看打卡结果 → 点「执行当日结算」。
4. **公告**：「公告发布」→ 范围选「全员」→ 发布。
5. **移动端**：`admin` 登录 → 「工作流」Tab 的「待我审批」里有一条演示待办 → 进详情点「同意」→ 「考勤」Tab 打卡（当天第 1 次按 = 上班卡，第 2 次 = 下班卡并结算当日结果）→ 「通知」Tab 下拉刷新能看到审批通知。

## 已知限制

- **网页端没有「提交申请」入口**：请假、报销等单据的提交在移动端，管理后台对应页面是只读列表 + 审批配置。
- **移动端只能登录平台超管 `admin`**：登录请求当前不带租户编号，租户用户（如 `tenant_admin`）无法从 App 登录。
- **平台超管不能发起新申请**：超管（租户 0）调用「获取申请表单」等发起入口会返回 403，这是租户隔离的预期行为。所以「全新提交 → 审批」的完整闭环目前无法只靠界面演示（演示数据里已内置进行中的流程实例供体验审批动作）；完整链路的自动化验证可参考 `backend/app/core/service/cmd/smokeseed` 种子工具。
- 站内信的 SSE 实时推送仅在管理后台自身的发送路径生效，工作流产生的通知只落库不推送——各端看通知用列表刷新。

## 常见问题

| 现象 | 原因与处理 |
|------|------------|
| 服务启动 fatal：`connection refused` | PostgreSQL / Redis / etcd 容器没起来或没就绪，`docker compose -f docker-compose.libs.yaml ps` 检查后重试 |
| admin / app 日志报找不到 core 服务 | core 没启动或 etcd 不通；先起 core 再起边端 |
| 登录时验证码总是错 | 验证码区分大小写且有时效，看不清就点图片换一张 |
| 前端请求报 CORS 错误 | 前端端口不在 admin 服务 `server.yaml` 的 `cors.origins` 白名单里，加上后重启 admin 服务 |
| 前端登录后马上被踢回登录页 | 后端服务刚重启过、旧令牌全部失效，退出重新登录即可；持续复现则确认三个服务都跑的是最新代码（改过 Go 代码必须重新 `go run`） |
| 移动端登录报网络错误 | `API_BASE_URL` 与运行方式不匹配（模拟器用 `10.0.2.2`，真机用电脑局域网 IP），改完 `.env` 需重新 `flutter run` |
| 改了后端代码不生效 | `go run` 是一次性进程，改动后要停掉重启；改了 proto 或 ent schema 还要先重新生成（见 [开发者教程](./oa-developer-guide.md)） |
