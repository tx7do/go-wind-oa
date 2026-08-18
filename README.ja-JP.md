# go-wind-oa｜OA ワークフロー承認システム

> **軽量・リニア・状態機械駆動のワークフロー承認エンジン — go-wind-oa**

go-wind-oa は、軽量なワークフロー承認エンジン、管理コンソール、モバイルクライアントからなるオフィスオートメーション（OA）ワークフロー承認システムです。

バックエンドは Go マイクロサービスフレームワーク [go-kratos](https://go-kratos.dev/) を基盤とし、core/admin/app の三サービス分離アーキテクチャを採用しています：core-service はワークフロー エンジンと内部メッセージングを保持する純 gRPC バックエンド、admin-service / app-service は HTTP エッジ転送層です。

フロントエンドの admin は Vue3 + Element-Plus、mobile は Flutter です。

[English](./README.en-US.md) | [中文](./README.md) | **日本語**

## リポジトリ構成

```
go-wind-oa/
├── backend/            # OA バックエンド（Kratos + Ent + Wire、三サービスアーキテクチャ）
│   ├── api/            # proto + buf 生成テンプレート（Go スタブ / openapi / admin TS / mobile Dart）
│   │   ├── protos/     # 6 保持ドメイン：oa / internal_message / authentication / identity / admin / app
│   │   └── gen/go/     # buf 生成 Go スタブ（ドメイン別ディレクトリ）
│   ├── app/
│   │   ├── core/service/   # 純 gRPC：ワークフロー エンジン + 内部メッセージング（HTTP エンドポイントなし）
│   │   ├── admin/service/  # HTTP エッジ：admin 認証 + 内部メッセージング + ワークフロー転送
│   │   └── app/service/    # HTTP エッジ：モバイル認証 + ワークフロー転送
│   └── pkg/            # 自己完結型基盤（middleware / serviceid / crypto / eventbus、cms 依存なし）
├── frontend/
│   ├── admin/          # 管理コンソール（Vue3 + Element-Plus）
│   └── mobile/         # モバイルクライアント（Flutter）
└── docs/
    ├── oa-workflow-design.md   # バックエンド アーキテクチャ
    └── oa-mobile-design.md     # モバイル アーキテクチャ
```

## バックエンドアーキテクチャ

三サービス分離、`go-wind-cms` の core/admin/app パターンを踏襲：

- **core-service**：純 gRPC、ent リポジトリとワークフロー エンジン実装を保持。`WorkflowService`（`oa.service.v1`）+ `InternalMessageService`（`internal_message.service.v1`）+ Category/Recipient を登録。ミドルウェア チェーン `logging + ent`（core は admin/app から呼ばれるバックエンド、auth なし）。
- **admin-service**：HTTP エッジ、admin フロントエンド向けに認証 + 内部メッセージング + ワークフローの HTTP エンドポイントを公開。各 service メソッドは転送層、gRPC クライアント経由で core-service を呼び出す。ミドルウェア チェーン `logging → auth+authz（ホワイトリスト）→ ent`、auth は ent の前でなければならない（順序逆転で ent は SystemViewer にフォールバック、テナント分離が無効化）。
- **app-service**：HTTP エッジ、モバイルクライアント向けに認証 + ワークフローの HTTP エンドポイントを公開。同じ転送層とミドルウェア順序。

Proto ドメイン分離：core の `oa/service/v1/workflow.proto` は純 gRPC（HTTP アノテーション削除済み）；HTTP ルーティング アノテーションは `admin/service/v1/i_workflow.proto` と `app/service/v1/i_workflow.proto` ラッパー proto に定義され、`oa.service.v1` メッセージ型を参照する。認証 HTTP エンドポイントは `admin/service/v1/i_authentication.proto` と `app/service/v1/i_authentication.proto` に定義され、cms `authentication.service.v1` メッセージ型を参照する。

詳細は `docs/oa-workflow-design.md` を参照。

## バックエンドビルド

各 service ディレクトリに Makefile（`include ../../../app.mk`）、`SERVICE_NAME` が buf openapi テンプレート選択を決定：

```bash
cd backend/app/core/service && make ent wire api build   # core に openapi なし（純 gRPC）
cd backend/app/admin/service && make openapi wire api build
cd backend/app/app/service && make openapi wire api build
```

`make ent` は ent ORM を生成（`--feature privacy` は省略不可、`TenantPrivacy` ポリシーが有効化される前提）。`make api` は Go proto スタブを生成。`make openapi` は service ごとに `buf.admin.openapi.gen.yaml` / `buf.app.openapi.gen.yaml` を選択（core はスキップ）。`make wire` は `wire_gen.go` を生成。

## Admin フロントエンド

Vue3 + Vite + TypeScript + Element-Plus + Pinia。OA モジュールコード：

- `src/api/composables/oa.ts` — Vue Query hooks で生成された `apiClient.workflowService` をラップ。
- `src/api/generated/admin/service/v1/index.ts` — `backend/api/buf.admin.typescript.gen.yaml` で生成。
- `src/pages/app/oa/definition/` — ワークフロー定義リスト + Drawer フォーム。
- `src/router/routes/modules/app/oa.ts` — フロントエンドルートモジュール（accessMode=frontend、自動 glob 登録）。

```bash
cd backend/api && buf generate --template buf.admin.typescript.gen.yaml   # TS クライアント生成
cd frontend/admin && pnpm i && pnpm dev
```

## モバイルクライアント

Flutter + bloc + cached_query + dio。OA feature：

- `lib/src/features/oa/services/workflow_service.dart` — `BaseService` サブクラス、生成された `apiClient.workflowService` を呼び出す。
- `lib/src/features/oa/pages/{task_list,task_detail,submit_apply,notifications,attendance,shell}/` — 各ページ。
- `lib/generated/api/app/service/v1/index.dart` — `backend/api/buf.flutter.oa.dart.gen.yaml` で生成（モバイル Dart クライアント、`ApiClient.workflowService` + `authenticationService` を含む）。

```bash
cd backend/api && buf generate --template buf.flutter.oa.dart.gen.yaml   # Dart クライアント生成
cd frontend/mobile && flutter run
```

> 現在の開発機には Flutter SDK が未導入（fvm シェルのみ）。ユーザーはローカルで `fvm use <version>` を実行後に上記を実行してください。

## スコープとバックエンド TODO

四機能の実装状態：
- ✅ ワークフロー承認（モバイル + admin、完全）
- ✅ ワークフロー定義管理（admin、完全；有効化/無効化インターフェースはバックエンド TODO）
- 🟡 インスタントメッセージプッシュ（モバイル骨架、バックエンド SSE/FCM/JPush 未対応）
- 🟡 モバイル勤怠打刻（モバイル骨架、バックエンド勤怠サービス/ジオフェンス/Wi-Fi 指紋ライブラリ未構築）
