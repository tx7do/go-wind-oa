<div align="center">

# GoWind OA｜Collaborative Office System

**An out-of-the-box enterprise collaborative office system**

> **Let collaborative office work flow as freely as the wind — GoWind OA**

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vuedotjs)](https://vuejs.org/)
[![Flutter](https://img.shields.io/badge/Flutter-3.x-02569B?logo=flutter)](https://flutter.dev/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)](https://www.docker.com/)

**English** | [中文](./README.md) | [日本語](./README.ja-JP.md)

</div>

---

## Overview

GoWind OA is an enterprise-grade **collaborative office system** covering everyday office scenarios such as approval workflows, HR attendance, leave, and expense reports. A lightweight workflow engine ships as one subsystem providing the approval-flow foundation, accompanied by an admin console and a mobile client to digitize office processes.

The backend is built on the [go-kratos](https://go-kratos.dev/) microservice framework with a core/admin/app three-service separation. The admin frontend uses Vue3 + Element Plus; the mobile client uses Flutter.

## Modules

| Module | Description | Status |
|------|------|------|
| Workflow Engine | Linear state machine, countersign/or-sign, withdrawal, leader/position resolver, business hook | ✅ |
| Approval Center | Pending/done/initiated lists, detail approval & forwarding, visual process-definition editor | ✅ |
| HR Attendance | GPS/Wi-Fi check-in, late/early-leave/absence/leave-aware settlement, holiday calendar, daily scheduled settlement | ✅ |
| Leave Management | Types & quotas, half-day granularity, auto quota deduction on approval, auto-guided process definition | ✅ |
| Expense Management | Multi-line items, invoice photo direct upload, auto-guided process definition | ✅ |
| Business Trip / Overtime / Seal / Outing | Four same-shape approval documents; each creates a document and submits the matching v1 workflow (auto-guided definition). Terminal state only syncs document status — no quota side effects | ✅ |
| Internal Messaging | Approval notifications persisted, inbox query. SSE push is effective only for admin's own SendMessage path; workflow notifications written via core do not trigger SSE | ✅ |
| Announcement Publishing | Reuses internal_message SendMessage fan-out (target_all for broadcast, ListUserIDsByOrgUnitIDs to expand org units into target_user_ids). No dedicated table | ✅ |
| Directory | App-side read-only wrapper (with redact masking) + admin/mobile dual-end org-tree browsing and member lists | ✅ |
| Form Engine | Field-schema-based dynamic form rendering (mobile generation, approval-side key-value display) | ✅ |

> The workflow engine is a subsystem of the collaborative office system, driving approval flows; it is not the system's identity.

## Tech Stack

<table>
<tr><th>Layer</th><th>Technology</th></tr>
<tr><td><strong>Backend Framework</strong></td><td><code>Golang</code> · <code>go-kratos v2</code> · <code>Wire</code> · <code>Protobuf / Buf</code></td></tr>
<tr><td><strong>ORM</strong></td><td><code>Ent</code> (with privacy layer and multi-tenant isolation) · <code>PostgreSQL</code></td></tr>
<tr><td><strong>Middleware</strong></td><td><code>Redis</code> · <code>MinIO</code> (S3-compatible object storage) · <code>Etcd</code> (service registry/discovery) · <code>Jaeger</code> (tracing)</td></tr>
<tr><td><strong>Auth</strong></td><td><code>JWT</code> · <code>RBAC</code> · <code>CAPTCHA</code> · multi-tenant data isolation</td></tr>
<tr><td><strong>Admin Frontend</strong></td><td><code>Vue 3</code> · <code>TypeScript</code> · <code>Vite</code> · <code>Element Plus</code> · <code>Pinia</code> · <code>TanStack Query</code></td></tr>
<tr><td><strong>Mobile Client</strong></td><td><code>Flutter</code> · <code>Dart</code> · <code>Dio</code> · <code>GetIt</code></td></tr>
<tr><td><strong>Code Generation</strong></td><td><code>Ent Schema → ORM</code> · <code>Protobuf → Go API / TypeScript / Dart client / OpenAPI</code> · <code>Wire DI</code></td></tr>
<tr><td><strong>DevOps</strong></td><td><code>Docker</code> · <code>Docker Compose</code> · <code>Swagger UI</code></td></tr>
</table>

## Repository Structure

```text
go-wind-oa/
├── backend/                   # Collaborative office backend
│   ├── api/                   # Protobuf definitions and code-gen templates
│   │   ├── protos/            # Business-domain protos (oa / internal_message / authentication / identity …)
│   │   └── gen/               # buf-generated Go stubs, OpenAPI, TS/Dart clients
│   ├── app/                   # Three-service main directories
│   │   ├── core/service/      # Pure gRPC: business logic and persistence
│   │   ├── admin/service/     # HTTP edge: admin auth and business forwarding
│   │   └── app/service/       # HTTP edge: mobile auth and business forwarding
│   ├── pkg/                   # Self-contained base (middleware / crypto / viewer / oss …)
│   └── scripts/               # Deployment and ops scripts
├── frontend/
│   ├── admin/                 # Admin console (Vue3 + Element Plus)
│   └── mobile/                # Mobile client (Flutter)
└── docs/                      # Design documents
```

## Architecture Overview

Three services, each with distinct responsibilities:

- **core-service**: Pure-gRPC backend holding ent repositories and business-logic implementations. Registers gRPC services for each business domain (workflow, attendance, leave, expense, internal messaging, …). Middleware `logging + ent` (privacy layer applies tenant isolation based on the viewer context).
- **admin-service**: HTTP edge exposing auth and business HTTP endpoints for the admin console; each method is a forwarder calling core-service via gRPC clients. Middleware `logging → auth + authz(whitelist) → ent`; auth must precede ent (inverted order makes ent fall back to SystemViewer, breaking tenant isolation).
- **app-service**: HTTP edge exposing auth and business HTTP endpoints for the mobile client; same forwarder pattern and middleware order.

Proto domain separation: core's `oa/service/v1/*.proto` are pure gRPC (HTTP annotations stripped); HTTP routing annotations live in `admin/service/v1/i_*.proto` and `app/service/v1/i_*.proto` wrapper protos, referencing `oa.service.v1` message types.

See [docs/oa-workflow-design.md](./docs/oa-workflow-design.md) for details.

## Backend Build

Each service directory has a Makefile (`include ../../../app.mk`); `SERVICE_NAME` selects the buf openapi template:

```bash
cd backend/app/core/service && make ent wire api build   # core has no openapi (pure gRPC)
cd backend/app/admin/service && make openapi wire api build
cd backend/app/app/service && make openapi wire api build
```

`make ent` generates the ent ORM (`--feature privacy` must not be omitted; it is the prerequisite for the `TenantPrivacy` policy). `make api` generates Go proto stubs. `make openapi` selects `buf.admin.openapi.gen.yaml` / `buf.app.openapi.gen.yaml` per service (core skipped). `make wire` generates `wire_gen.go`.

> For backend environment setup, project structure, and deployment, see [backend/README.md](./backend/README.md).

## Admin Frontend

Vue3 + Vite + TypeScript + Element Plus + Pinia + TanStack Query. OA module code:

- `src/api/composables/oa.ts` — TanStack Query hooks wrapping the generated `apiClient` services.
- `src/api/generated/admin/service/v1/index.ts` — generated by `backend/api/buf.admin.typescript.gen.yaml`.
- `src/pages/app/oa/` — approval center, attendance records, holiday setup, leave/expense management, same-shape documents business-trip/overtime/seal/outing management, announcement publishing, directory, process-definition editor.
- `src/router/routes/modules/app/oa.ts` — frontend route module (auto glob registration).

```bash
cd backend/api && buf generate --template buf.admin.typescript.gen.yaml   # generate TS client
cd frontend/admin && pnpm i && pnpm dev
```

## Mobile Client

Flutter + Dio + GetIt. OA feature:

- `lib/src/features/oa/services/` — business services (workflow, attendance, leave, expense, same-shape documents business-trip/overtime/seal/outing, internal-message notifications, directory, file upload). Full list in [docs/oa-mobile-design.md](./docs/oa-mobile-design.md) §2/§4.
- `lib/src/features/oa/pages/` — login, approval task list & detail, generic submit-application (dynamic form), per-document submission pages, attendance check-in, notifications, directory. See same doc §3/§4.
- `lib/generated/api/app/service/v1/index.dart` — generated by `backend/api/buf.app.dart.gen.yaml`.

```bash
cd backend/api && buf generate --template buf.app.dart.gen.yaml   # generate Dart client
cd frontend/mobile && flutter run
```

## Development Environment

Required base tools, middleware, and Docker Compose one-click startup are documented in the “Prerequisites” and “Middleware” sections of [backend/README.md](./backend/README.md).

## Related Documentation

- [Backend Architecture Design](./docs/oa-workflow-design.md)
- [Mobile Design](./docs/oa-mobile-design.md)
- [Backend README](./backend/README.md)

## License

[MIT](./LICENSE)
