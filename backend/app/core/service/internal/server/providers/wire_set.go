//go:build wireinject
// +build wireinject

//go:generate go run github.com/google/wire/cmd/wire

// This file defines the dependency injection ProviderSet for the data layer and contains no business logic.
// The build tag `wireinject` excludes this source from normal `go build` and final binaries.
// Run `go generate ./...` or `go run github.com/google/wire/cmd/wire` to regenerate the Wire output (e.g. `wire_gen.go`), which will be included in final builds.
// Keep provider constructors here only; avoid init-time side effects or runtime logic in this file.

package providers

import (
	"github.com/google/wire"

	"go-wind-oa/app/core/service/internal/server"
)

// ProviderSet is the Wire provider set for server layer.
//
// OA core-service 為 gRPC-only：僅註冊 grpc server 與其中間件鏈
// （logging + ent）。不再持有 asynq 任務隊列。
var ProviderSet = wire.NewSet(
	server.NewGrpcMiddleware,
	server.NewGrpcServer,
)
