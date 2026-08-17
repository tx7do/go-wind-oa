//go:build wireinject
// +build wireinject

package providers

import (
	"github.com/google/wire"

	"go-wind-oa/app/oa/service/internal/server"
)

// ProviderSet OA 服务端依赖注入集合：仅含 rest server + 中间件构造。
var ProviderSet = wire.NewSet(
	server.NewRestServer,
	server.NewRestMiddleware,
)
