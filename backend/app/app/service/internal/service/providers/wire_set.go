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

	"go-wind-oa/app/app/service/internal/service"
)

// ProviderSet is the Wire provider set for service layer.
//
// OA app-service 只暴露鑑權轉發與工作流轉發兩類 HTTP 邊端服務，均為轉發層。
var ProviderSet = wire.NewSet(
	service.NewAuthenticationService,
	service.NewWorkflowService,
)
