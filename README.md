# go-wind-oa

办公自动化（OA）系统：轻量级工作流审批引擎 + 管理后台 + 移动端。

## 仓结构

```
go-wind-oa/
├── backend/            # OA 后端（Kratos + Ent + Wire，三 service 架构）
│   ├── api/            # proto + buf 生成模板（Go 桩 / openapi / admin TS / mobile Dart）
│   │   ├── protos/     # 6 个保留域：oa / internal_message / authentication / identity / admin / app
│   │   └── gen/go/     # buf 生成 Go 桩（按域分目录）
│   ├── app/
│   │   ├── core/service/   # 纯 gRPC：工作流引擎 + 站內信（无 HTTP 端點）
│   │   ├── admin/service/  # HTTP 边端：admin 前端鑑權 + 站內信 + 工作流轉發
│   │   └── app/service/    # HTTP 边端：移動端鑑權 + 工作流轉發
│   └── pkg/            # 自包含基座（middleware / serviceid / crypto / eventbus 等，無 cms 依賴）
├── frontend/
│   ├── admin/          # Admin 管理前端（Vue3 + Element-Plus）
│   └── mobile/         # 移動端（Flutter）
└── docs/
    ├── oa-workflow-design.md   # 後端架構（注：部分章節描述舊單 service 架構，待更新）
    └── oa-mobile-design.md     # 移動端架構（注：同上）
```

## 後端架構

三 service 分離，仿 `go-wind-cms` 的 core/admin/app 模式：

- **core-service**：純 gRPC，持有 ent 倉庫與工作流引擎實現。註冊 `WorkflowService`（`oa.service.v1`）+ `InternalMessageService`（`internal_message.service.v1`）+ Category/Recipient。中間件 `logging + ent`（core 是被 admin/app 調用的後端，無 auth）。
- **admin-service**：HTTP 邊端，為 admin 前端暴露鑑權 + 站內信 + 工作流的 HTTP 端點。各 service 方法為轉發層，經 gRPC 客戶端打 core-service。中間件 `logging → auth+authz(白名單) → ent`，auth 必須在 ent 之前（順序顛倒則 ent 兜底 SystemViewer，租戶隔離失效）。
- **app-service**：HTTP 邊端，為移動端暴露鑑權 + 工作流的 HTTP 端點。同款轉發層與中間件順序。

Proto 域分離：core 的 `oa/service/v1/workflow.proto` 為純 gRPC（HTTP 注解剝離）；HTTP 路由注解定義在 `admin/service/v1/i_workflow.proto` 與 `app/service/v1/i_workflow.proto` wrapper proto，引用 `oa.service.v1` 消息類型。鑑權 HTTP 端點定義在 `admin/service/v1/i_authentication.proto` 與 `app/service/v1/i_authentication.proto`，引用 cms `authentication.service.v1` 消息類型。

## 後端構建

每個 service 目錄都有 Makefile（`include ../../../app.mk`），`SERVICE_NAME` 決定 buf openapi 模板選擇：

```bash
cd backend/app/core/service && make ent wire api build   # core 無 openapi（純 gRPC）
cd backend/app/admin/service && make openapi wire api build
cd backend/app/app/service && make openapi wire api build
```

`make ent` 生成 ent ORM（`--feature privacy` 不可省，是 `TenantPrivacy` 策略生效前提）。`make api` 生成 Go proto 樁。`make openapi` 按 service 選 `buf.admin.openapi.gen.yaml` / `buf.app.openapi.gen.yaml`（core 跳過）。`make wire` 生成 `wire_gen.go`。

> 注：`docs/oa-workflow-design.md` 與 `docs/oa-mobile-design.md` 撰寫於舊的單 service 架構時期（`app/oa/service`），其中目錄結構、proto 路徑（`oa/v1`）、生成模板名（`buf.vue-element.oa.typescript.gen.yaml`）、`swagger_parser` 鏈等描述均與現狀不符。以本 README 與代碼為准。

## Admin 前端

Vue3 + Vite + TypeScript + Element-Plus + Pinia。OA 模塊代碼：

- `src/api/composables/oa.ts` — Vue Query hooks 封裝生成的 `apiClient.workflowService`。
- `src/api/generated/admin/service/v1/index.ts` — 由 `backend/api/buf.admin.typescript.gen.yaml` 生成。
- `src/pages/app/oa/definition/` — 流程定義列表 + Drawer 表單。
- `src/router/routes/modules/app/oa.ts` — 前端路由模塊（accessMode=frontend，自動 glob 註冊）。

```bash
cd backend/api && buf generate --template buf.admin.typescript.gen.yaml   # 生成 TS 客戶端
cd frontend/admin && pnpm i && pnpm dev
```

## 移動端

Flutter + bloc + cached_query + dio。OA feature：

- `lib/src/features/oa/services/workflow_service.dart` — `BaseService` 子類，調生成的 `apiClient.workflowService`。
- `lib/src/features/oa/pages/{task_list,task_detail,submit_apply,notifications,attendance,shell}/` — 各頁面。
- `lib/generated/api/app/service/v1/index.dart` — 由 `backend/api/buf.flutter.oa.dart.gen.yaml` 生成（移動端 Dart 客戶端，含 `ApiClient.workflowService` + `authenticationService`）。

```bash
cd backend/api && buf generate --template buf.flutter.oa.dart.gen.yaml   # 生成 Dart 客戶端
cd frontend/mobile && flutter run
```

> 當前開發機未裝 Flutter SDK（僅 fvm 殼）。用戶本地 `fvm use <version>` 後執行。

## 範圍與後端待辦

四功能實現狀態：
- ✅ 工作流審批（移動端 + admin，完整）
- ✅ 流程定義管理（admin，完整；啟用/禁用接口為後端待辦）
- 🟡 即時消息推送（移動端骨架，後端 SSE/FCM/JPush 待對接）
- 🟡 移動考勤打卡（移動端骨架，後端考勤服務/圍欌/ Wi-Fi 指紋庫待建）

