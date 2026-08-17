//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/google/wire"

	"github.com/go-kratos/kratos/v2"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	dataProviders "go-wind-oa/app/oa/service/internal/data/providers"
	serverProviders "go-wind-oa/app/oa/service/internal/server/providers"
	serviceProviders "go-wind-oa/app/oa/service/internal/service/providers"
)

// initApp init kratos application.
//
// 与 go-wind-cms/app/admin/service/cmd/server/wire.go 同构：仅 wire.Build
// 三个 provider set + newApp，无额外绑定。OA 仅起 HTTP server，故 newApp 只接收
// *http.Server（无 gRPC / SSE server）。
func initApp(*bootstrap.Context) (*kratos.App, func(), error) {
	panic(
		wire.Build(
			serverProviders.ProviderSet,
			serviceProviders.ProviderSet,
			dataProviders.ProviderSet,
			newApp,
		),
	)
}
