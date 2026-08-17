//go:build wireinject
// +build wireinject

// Package providers 暴露 OA 数据层的 wire provider 集合。
//
// 与 go-wind-cms/app/core/service/internal/data/providers/wire_set.go 同构：
// 仅罗列构造函数引用（wire.NewSet），不含任何 wire.Bind；具体的注入由顶层
// wire.go 的 wire.Build 完成并经 wire 代码生成得到 wire_gen.go。
package providers

import (
	"github.com/google/wire"

	client "go-wind-oa/app/oa/service/internal/data/client"
	"go-wind-oa/app/oa/service/internal/data"
)

// ProviderSet OA 数据层的依赖注入集合。
//
// 包含两块：
//  1. ent 客户端（client.NewEntClient，砖块复用 cms core 同名构造）—— 为仓库提供 ORM 会话；
//  2. 四个仓库 + 两个 cms gRPC 客户端：
//     - data.NewNotificationServiceClient（core-service 站内信通知组件，审批流转通知）；
//     - data.NewAuthenticationServiceClient（admin-service 鉴权组件，登录/登出/验证码/令牌刷新转发）。
//
// 不在此处出现：鉴权 / ent viewer 中间件、HTTP server —— 它们由 server/providers 提供。
var ProviderSet = wire.NewSet(
	client.NewEntClient,
	data.NewDiscovery,
	data.NewWorkflowDefinitionRepo,
	data.NewWorkflowInstanceRepo,
	data.NewWorkflowTaskRepo,
	data.NewWorkflowLogRepo,
	data.NewNotificationServiceClient,
	data.NewAuthenticationServiceClient,
)
