package data

import (
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/kratos-bootstrap/rpc"

	internalMessageV1 "go-wind-cms/api/gen/go/internal_message/service/v1"
	cmsServiceID "go-wind-cms/pkg/serviceid"
)

// NewNotificationServiceClient 创建对 go-wind-cms 站内信通知组件的 gRPC 客户端。
//
// 砖块式复用：OA 工作流审批流转时不自建通知通道，而是经此客户端调用
// internal_message 服务的 SendMessage，将“下一审批人 / 申请人”的待办/结果
// 通知投递进 cms 既有的站内信通道（落库 + SSE 推送）。
//
// 构造方式与 go-wind-cms/app/admin/service/internal/data/data.go 中的
// NewInternalMessageServiceClient 同构：经服务发现定位 core-service，
// 由 rpc.CreateGrpcClient 取底层 grpc.ClientConn，再由生成客户端包装。
func NewNotificationServiceClient(ctx *bootstrap.Context, r registry.Discovery) internalMessageV1.InternalMessageServiceClient {
	cli, err := rpc.CreateGrpcClient(ctx.Context(), r, cmsServiceID.NewDiscoveryName(cmsServiceID.CoreService), ctx.GetConfig())
	if err != nil {
		return nil
	}
	return internalMessageV1.NewInternalMessageServiceClient(cli)
}
