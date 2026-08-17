# go-wind-oa

办公自动化（OA）系统：轻量级工作流审批引擎 + 管理后台 + 移动端。

## 仓结构

```
go-wind-oa/
├── backend/            # OA 工作流后端（Kratos + Ent + Wire，详见 docs/oa-workflow-design.md）
│   ├── api/            # proto + buf 生成模板（Go 桩 / admin TS 客户端 / mobile openapi）
│   └── app/oa/service/ # 服务实现（Makefile 驱动 ent/api/openapi/wire）
├── frontend/
│   ├── admin/               # Admin 管理前端（Vue3+Element-Plus，拷自 go-wind-admin）
│   └── mobile/             # 移动端（Flutter，拷自 go-wind-cms/flutter_app）
└── docs/
    ├── oa-workflow-design.md   # 后端架构
    └── oa-mobile-design.md     # 移动端架构
```

## 后端构建

```bash
cd backend/app/oa/service
make ent          # 生成 ent ORM（--feature privacy 不可省）
make api          # 生成 Go proto 桩
make openapi      # 生成 openapi.yaml（供 mobile swagger_parser）
make wire         # 生成 wire_gen.go
make build        # 编译
```

详见 `docs/oa-workflow-design.md`。

## Admin 前端

拷自 `go-wind-admin/frontend/admin/vue-element`（Vue3 + Vite + TypeScript + Element-Plus + Pinia + vue-router5）。OA 模块代码：

- `src/api/composables/oa.ts` — Vue Query hooks 封装生成的 `apiClient.workflowService`。
- `src/api/generated/oa/v1/index.ts` — 由 `backend/api/buf.vue-element.oa.typescript.gen.yaml` 生成。
- `src/pages/app/oa/definition/` — 流程定义列表 + Drawer 表单（`ProPage` + `ProModal`，对齐 cms internal_message 模式）。
- `src/router/routes/modules/app/oa.ts` — 前端路由模块（accessMode=frontend，自动 glob 注册）。

```bash
cd backend/api && buf generate --template buf.vue-element.oa.typescript.gen.yaml   # 生成 TS 客户端
cd frontend/admin && pnpm i && pnpm dev
```

## 移动端

拷自 `go-wind-cms/frontend/app/flutter_app`（Flutter + bloc + cached_query + dio）。OA feature：

- `lib/src/features/oa/services/workflow_service.dart` — `BaseService` 子类，调生成的 `apiClient.workflowService`。
- `lib/src/features/oa/pages/{task_list,task_detail,submit_apply,notifications,attendance}/` — 五页面（三完整 + 两骨架）。
- `lib/src/features/oa/pages/shell/oa_shell_page.dart` — 底部导航 Shell。

```bash
cd backend/app/oa/service && make openapi          # 生成 openapi.yaml
cd frontend/mobile && dart run bin/clean_and_gen.dart   # 生成 Dart 客户端（需 Flutter SDK）
flutter run
```

> 当前开发机未装 Flutter SDK（仅 fvm 壳）。用户本地 `fvm use <version>` 后执行后两步。

## 范围与后端待办

四功能实现状态：
- ✅ 工作流审批（移动端，完整）
- ✅ 流程定义管理（admin，完整；启用/禁用接口为后端待办）
- 🟡 即時消息推送（移动端骨架，后端 SSE/FCM/JPush 待对接）
- 🟡 移動考勤打卡（移动端骨架，后端考勤服务/围栏/Wi-Fi 指纹库待建）

后端待办清单见 `docs/oa-mobile-design.md §5`，包括：
- **鉴权缺口**（阻塞移动端登录）：OA 后端无 `authenticationService`，需转发 cms 或前端直连 cms。
- **单任务详情端点**：后端缺 `GetTask`，详情页信息量受限。
- 推送通道、考勤服务、定义启用/禁用接口。
