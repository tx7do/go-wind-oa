# Makefile for building the go-wind-oa OA micro service application.
#
# 该 Makefile 仅服务于 oa/service 子模块，与 go-wind-cms/backend/app.mk 的目标
# 子集同构，剔除 OA MVP 不需要的功能（openapi / docker / wire —— OA 无 SSE、
# 不打镜像、用 go run 替代 wire 生成产物由后续接入）。保留的目标与 cms 对齐：
#   - ent : 由 ent CLI 按本模块 internal/data/ent/schema 生成 ORM 代码
#           （与 cms 同样的五个 feature 标志，privacy 为 TenantPrivacy 策略所必需）
#   - api : 由 buf 按 ../../api/buf.gen.yaml 生成 proto 桩代码
#   - build / run / clean / help : 通用构建与清理
#
# 注意：ent 目标的 ifneq 守卫与 cms 完全一致 —— 只在 internal/data/ent 存在时触发，
# 因此即便某次提交删除了 schema 目录，make ent 也会静默 no-op 而非报错。

GOPATH ?= $(shell go env GOPATH)
GO_CMD  := GO111MODULE=on go

.PHONY: build clean gen ent api openapi wire run help

# build golang application
build: api ent
	@$(GO_CMD) build -o ./bin/ ./...

# build golang application only (no codegen)
build_only:
	@$(GO_CMD) build -o ./bin/ ./...

# run application
run: api ent
	-@$(GO_CMD) run ./cmd/server -c ./configs

# generate code (proto + ent)
gen: ent api wire

# clean build files
clean:
	@$(GO_CMD) clean

# generate ent code, if ent schema exist in the project's internal/data/ent folder.
# Feature flags MUST match go-wind-cms: privacy (TenantPrivacy policy), entql,
# sql/modifier, sql/upsert, sql/lock. Dropping privacy disables tenant isolation.
ent:
ifneq ("$(wildcard ./internal/data/ent)","")
	@ent generate \
				--feature privacy \
				--feature entql \
				--feature sql/modifier \
				--feature sql/upsert \
				--feature sql/lock \
				./internal/data/ent/schema
endif

# generate protobuf api go code (oa/v1 -> api/gen/go/oa/v1)
api:
	@cd ../../../api && \
	buf generate

# generate openapi v3 yaml doc (oa/v1 -> app/oa/service/cmd/server/assets/openapi.yaml)
# 供 frontend/mobile（拷自 cms flutter_app）的 swagger_parser 消费，生成 Dart 客户端。
# 与 go-wind-cms/backend/app.mk openapi 目标同构。
openapi:
	@cd ../../../api && \
	buf generate --template buf.oa.openapi.gen.yaml

# generate wire code (与 go-wind-cms/backend/app.mk wire 目标同构)。
# 在 cmd/server 下生成 wire_gen.go，提供 main.go 引用的 initApp。
wire:
	@go run -mod=mod github.com/google/wire/cmd/wire ./cmd/server

# show help
help:
	@echo ""
	@echo "Usage:"
	@echo " make [target]"
	@echo ""
	@echo "Targets:"
	@echo "   api     - generate protobuf Go stubs via buf"
	@echo "   ent     - generate ent ORM code (privacy feature REQUIRED for tenant isolation)"
	@echo "   openapi - generate OpenAPI v3 yaml (consumed by mobile swagger_parser)"
	@echo "   wire    - generate wire dependency-injection code (wire_gen.go)"
	@echo "   build   - build the OA service binary"
	@echo "   run     - run the OA service locally"
	@echo "   clean   - clean build artifacts"
	@echo ""

.DEFAULT_GOAL := help
