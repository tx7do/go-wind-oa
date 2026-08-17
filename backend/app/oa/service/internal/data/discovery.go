package data

import (
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bRegistry "github.com/tx7do/kratos-bootstrap/registry"
)

// NewDiscovery 创建服务发现客户端。
//
// 砖块复用 go-wind-cms：构造与 cms/app/admin/service/internal/data/data.go
// 中的 NewDiscovery 同构（差异仅在不引入 DTM 驱动 —— OA 不参与跨服务分布式事务）。
// 仅用于经服务发现定位 core-service，为 NewNotificationServiceClient 提供 gRPC 连接。
func NewDiscovery(ctx *bootstrap.Context) registry.Discovery {
	cfg := ctx.GetConfig()
	if cfg == nil {
		return nil
	}

	discovery, err := bRegistry.NewDiscovery(cfg.Registry)
	if err != nil {
		return nil
	}

	return discovery
}
