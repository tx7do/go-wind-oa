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

	"go-wind-oa/app/app/service/internal/data"

	"go-wind-oa/pkg/middleware/auth"
)

// ProviderSet is the Wire provider set for data layer.
//
// OA app-service 只持有鑑權轉發與工作流轉發兩類下游客戶端，以及 redis /
// discovery / authorizer 等基座 infra。不再持有 DTM、minio、tenant resolver、
// 或任何 cms 業務域的 gRPC 客戶端。
var ProviderSet = wire.NewSet(
	data.NewRedisClient,
	data.NewDiscovery,

	data.NewClientType,
	data.NewAuthorizer,

	auth.NewTokenChecker,

	data.NewAuthenticationServiceClient,

	data.NewWorkflowServiceClient,
)
