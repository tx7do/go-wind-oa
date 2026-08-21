<div align="center">

# GoWind OA｜コラボレーティブ・オフィスシステム

**そのまま使えるエンタープライズ級コラボレーティブ・オフィスシステム**

> **コラボレーティブ・オフィスを風のように自由に — GoWind OA**

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vuedotjs)](https://vuejs.org/)
[![Flutter](https://img.shields.io/badge/Flutter-3.x-02569B?logo=flutter)](https://flutter.dev/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)](https://www.docker.com/)

[English](./README.en-US.md) | [中文](./README.md) | **日本語**

</div>

---

## 概要

GoWind OA は、承認フロー、人事勤怠、休暇、経費精算といった日常オフィスシナリオをカバーするエンタープライズ級**コラボレーティブ・オフィスシステム**です。承認フロー基盤を提供する一つのサブシステムとして軽量ワークフロー エンジンを内蔵し、管理コンソールとモバイルクライアントを併せてオフィスプロセスのデジタル化を実現します。

バックエンドは [go-kratos](https://go-kratos.dev/) マイクロサービスフレームワーク上で core/admin/app の三サービス分離アーキテクチャを採用しています。管理コンソールは Vue3 + Element Plus、モバイルクライアントは Flutter です。

## 機能モジュール

| モジュール | 説明 | 状態 |
|------|------|------|
| ワークフロー エンジン | 線形状態機械、相議/分議、取下げ、リーダー/職位による審批者解決、業務フック | ✅ |
| 承認センター | 未処理/処理済/自申請リスト、詳細承認・転送、可視プロセス定義エディター | ✅ |
| 人事勤怠 | GPS/Wi-Fi 打刻、遅刻/早退/欠勤/休暇連動精算、祝祭日カレンダー、日次定時精算 | ✅ |
| 休暇管理 | 種別と残高、半日粒度、承認時の自動残好控除、自動ガイド付きプロセス定義 | ✅ |
| 経費精算 | 複数明細行、請求書写真の直接アップロード、自動ガイド付きプロセス定義 | ✅ |
| 出張 / 残業 / 印章 / 外出 | 4 種の同型承認書類。各々書類を作成し対応 v1 ワークフローに提出（定義は自動ガイド）。終端状態は書類状態の同期のみで、残高への副作用なし | ✅ |
| 社内メッセージ | 承認通知の DB 永続化、受信箱クエリ。SSE プッシュは admin 自身の SendMessage パスのみ有効であり、core 経由で書き込まれたワークフロー通知は SSE をトリガーしない | ✅ |
| お知らせ配信 | 社内メッセージの SendMessage ファンアウトを再利用（全員は target_all、部門別は ListUserIDsByOrgUnitIDs で target_user_ids に展開）。専用テーブルなし | ✅ |
| アドレス帳 | app 側の読み取り専用ラッパー（redact マスキング付き）+ admin/モバイル両エンドの組織ツリー閲覧とメンバーリスト | ✅ |
| フォーム エンジン | フィールド スキーマに基づく動的フォーム描画（モバイル生成、承認側はキー・バリュー表示） | ✅ |

> ワークフロー エンジンはコラボレーティブ・オフィスシステムのサブシステムであり、承認フローを駆動するものです。システム自体の位置づけではありません。

## 技術スタック

<table>
<tr><th>レイヤー</th><th>技術</th></tr>
<tr><td><strong>バックエンド フレームワーク</strong></td><td><code>Golang</code> · <code>go-kratos v2</code> · <code>Wire</code> · <code>Protobuf / Buf</code></td></tr>
<tr><td><strong>ORM</strong></td><td><code>Ent</code>（プライバシー層とマルチテナント分離を含む） · <code>PostgreSQL</code></td></tr>
<tr><td><strong>ミドルウェア</strong></td><td><code>Redis</code> · <code>MinIO</code>（S3 互換オブジェクトストレージ） · <code>Etcd</code>（サービスレジストリ/検出） · <code>Jaeger</code>（分散トレーシング）</td></tr>
<tr><td><strong>認証・認可</strong></td><td><code>JWT</code> · <code>RBAC</code> · <code>CAPTCHA</code> · マルチテナント データ分離</td></tr>
<tr><td><strong>管理コンソール</strong></td><td><code>Vue 3</code> · <code>TypeScript</code> · <code>Vite</code> · <code>Element Plus</code> · <code>Pinia</code> · <code>TanStack Query</code></td></tr>
<tr><td><strong>モバイルクライアント</strong></td><td><code>Flutter</code> · <code>Dart</code> · <code>Dio</code> · <code>GetIt</code></td></tr>
<tr><td><strong>コード生成</strong></td><td><code>Ent Schema → ORM</code> · <code>Protobuf → Go API / TypeScript / Dart クライアント / OpenAPI</code> · <code>Wire DI</code></td></tr>
<tr><td><strong>DevOps</strong></td><td><code>Docker</code> · <code>Docker Compose</code> · <code>Swagger UI</code></td></tr>
</table>

## リポジトリ構成

```text
go-wind-oa/
├── backend/                   # コラボレーティブ・オフィス バックエンド
│   ├── api/                   # Protobuf 定義とコード生成テンプレート
│   │   ├── protos/            # 各業務ドメイン proto（oa / internal_message / authentication / identity …）
│   │   └── gen/               # buf 生成 Go スタブ、OpenAPI、TS/Dart クライアント
│   ├── app/                   # 三サービス主ディレクトリ
│   │   ├── core/service/      # 純 gRPC：業務ロジックと永続化
│   │   ├── admin/service/     # HTTP エッジ：管理コンソール認証と業務転送
│   │   └── app/service/       # HTTP エッジ：モバイル認証と業務転送
│   ├── pkg/                   # 自己完結型基盤（middleware / crypto / viewer / oss …）
│   └── scripts/               # デプロイ・運用スクリプト
├── frontend/
│   ├── admin/                 # 管理コンソール（Vue3 + Element Plus）
│   └── mobile/                # モバイルクライアント（Flutter）
└── docs/                      # 設計ドキュメント
```

## アーキテクチャ概要

三サービス分離、各々が固有の責務を担います：

- **core-service**：純 gRPC バックエンド、ent リポジトリと業務ロジック実装を保持。各業務ドメイン（ワークフロー、勤怠、休暇、経費、社内メッセージ等）の gRPC サービスを登録。ミドルウェア `logging + ent`（プライバシー層は viewer コンテキストに基づきテナント分離を適用）。
- **admin-service**：HTTP エッジ、管理コンソール向けに認証と業務 HTTP エンドポイントを公開。各メソッドは転送層、gRPC クライアント経由で core-service を呼び出す。ミドルウェア `logging → auth + authz(ホワイトリスト) → ent`、auth は ent の前でなければならない（順序逆転で ent は SystemViewer にフォールバック、テナント分離が無効化）。
- **app-service**：HTTP エッジ、モバイルクライアント向けに認証と業務 HTTP エンドポイントを公開。同じ転送層とミドルウェア順序。

Proto ドメイン分離：core の `oa/service/v1/*.proto` は純 gRPC（HTTP アノテーション削除済み）；HTTP ルーティング アノテーションは `admin/service/v1/i_*.proto` と `app/service/v1/i_*.proto` ラッパー proto に定義され、`oa.service.v1` メッセージ型を参照する。

詳細は [docs/oa-workflow-design.md](./docs/oa-workflow-design.md) を参照。

## バックエンドビルド

各 service ディレクトリに Makefile（`include ../../../app.mk`）、`SERVICE_NAME` が buf openapi テンプレート選択を決定：

```bash
cd backend/app/core/service && make ent wire api build   # core に openapi なし（純 gRPC）
cd backend/app/admin/service && make openapi wire api build
cd backend/app/app/service && make openapi wire api build
```

`make ent` は ent ORM を生成（`--feature privacy` は省略不可、`TenantPrivacy` ポリシーが有効化される前提）。`make api` は Go proto スタブを生成。`make openapi` は service ごとに `buf.admin.openapi.gen.yaml` / `buf.app.openapi.gen.yaml` を選択（core はスキップ）。`make wire` は `wire_gen.go` を生成。

> バックエンドの環境準備、プロジェクト構成、デプロイ手順は [backend/README.md](./backend/README.md) を参照してください。

## 管理コンソール

Vue3 + Vite + TypeScript + Element Plus + Pinia + TanStack Query。OA モジュールコード：

- `src/api/composables/oa.ts` — TanStack Query hooks、生成された `apiClient` 各業務サービスをラップ。
- `src/api/generated/admin/service/v1/index.ts` — `backend/api/buf.admin.typescript.gen.yaml` で生成。
- `src/pages/app/oa/` — 承認センター、勤怠記録、祝祭日設定、休暇/経費管理、出張/残業/印章/外出などの同型書類管理、お知らせ配信、アドレス帳、プロセス定義エディター。
- `src/router/routes/modules/app/oa.ts` — フロントエンドルートモジュール（自動 glob 登録）。

```bash
cd backend/api && buf generate --template buf.admin.typescript.gen.yaml   # TS クライアント生成
cd frontend/admin && pnpm i && pnpm dev
```

## モバイルクライアント

Flutter + Dio + GetIt。OA feature：

- `lib/src/features/oa/services/` — 各業務サービス（ワークフロー、勤怠、休暇、経費、出張/残業/印章/外出などの同型書類、社内メッセージ通知、アドレス帳、ファイルアップロード）。完全な一覧は [docs/oa-mobile-design.md](./docs/oa-mobile-design.md) §2/§4 を参照。
- `lib/src/features/oa/pages/` — ログイン、承認タスクリストと詳細、一般申請提出（動的フォーム）、各書類の提出ページ、勤怠打刻、通知、アドレス帳。同上ドキュメント §3/§4 を参照。
- `lib/generated/api/app/service/v1/index.dart` — `backend/api/buf.app.dart.gen.yaml` で生成。

```bash
cd backend/api && buf generate --template buf.app.dart.gen.yaml   # Dart クライアント生成
cd frontend/mobile && flutter run
```

## 開発環境

必要な基盤ツールとミドルウェア、Docker Compose による一括起動については、[backend/README.md](./backend/README.md) の「前提要件」および「ミドルウェア」セクションを参照してください。

## 関連ドキュメント

- [バックエンド アーキテクチャ設計](./docs/oa-workflow-design.md)
- [モバイル設計](./docs/oa-mobile-design.md)
- [バックエンド README](./backend/README.md)

## License

[MIT](./LICENSE)
