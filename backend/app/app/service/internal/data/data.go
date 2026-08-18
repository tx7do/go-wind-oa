package data

import (
	"github.com/redis/go-redis/v9"

	authzEngine "github.com/tx7do/kratos-authz/engine"
	"github.com/tx7do/kratos-authz/engine/noop"

	"github.com/go-kratos/kratos/v2/registry"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	redisClient "github.com/tx7do/kratos-bootstrap/cache/redis"
	bRegistry "github.com/tx7do/kratos-bootstrap/registry"
	"github.com/tx7do/kratos-bootstrap/rpc"

	authenticationV1 "go-wind-oa/api/gen/go/authentication/service/v1"
	internalMessageV1 "go-wind-oa/api/gen/go/internal_message/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"

	"go-wind-oa/pkg/serviceid"
)

func NewClientType() authenticationV1.ClientType {
	return authenticationV1.ClientType_app
}

// NewRedisClient 创建Redis客户端
func NewRedisClient(ctx *bootstrap.Context) (*redis.Client, func(), error) {
	cfg := ctx.GetConfig()
	if cfg == nil {
		return nil, func() {}, nil
	}

	l := ctx.NewLoggerHelper("redis/data/app-service")

	cli := redisClient.NewClient(cfg.Data, l)

	return cli, func() {
		if err := cli.Close(); err != nil {
			l.Error(err)
		}
	}, nil
}

// NewDiscovery 创建服务发现客户端
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

// NewAuthorizer 创建权鉴器
func NewAuthorizer() authzEngine.Engine {
	return noop.State{}
}

func NewAuthenticationServiceClient(ctx *bootstrap.Context, r registry.Discovery) authenticationV1.AuthenticationServiceClient {
	cli, err := rpc.CreateGrpcClient(ctx.Context(), r, serviceid.NewDiscoveryName(serviceid.CoreService), ctx.GetConfig())
	if err != nil {
		return nil
	}

	return authenticationV1.NewAuthenticationServiceClient(cli)
}

// NewWorkflowServiceClient 创建对 OA core-service 工作流引擎的 gRPC 客户端。
//
// app-service 的 HTTP 邊端收到工作流請求後，經此客戶端轉發到 core-service
// 的 WorkflowService gRPC 實現（狀態機落庫）。構造方式與 cms app-service 的
// NewXxxServiceClient 同構。
func NewWorkflowServiceClient(ctx *bootstrap.Context, r registry.Discovery) oaV1.WorkflowServiceClient {
	cli, err := rpc.CreateGrpcClient(ctx.Context(), r, serviceid.NewDiscoveryName(serviceid.CoreService), ctx.GetConfig())
	if err != nil {
		return nil
	}

	return oaV1.NewWorkflowServiceClient(cli)
}

// NewInternalMessageServiceClient 创建对 OA core-service 站内信的 gRPC 客户端。
//
// app-service 的 HTTP 邊端收到站內信查詢請求後，經此客戶端轉發到 core-service
// 的 InternalMessageService gRPC 實現。構造方式與 NewWorkflowServiceClient 同構。
func NewInternalMessageServiceClient(ctx *bootstrap.Context, r registry.Discovery) internalMessageV1.InternalMessageServiceClient {
	cli, err := rpc.CreateGrpcClient(ctx.Context(), r, serviceid.NewDiscoveryName(serviceid.CoreService), ctx.GetConfig())
	if err != nil {
		return nil
	}

	return internalMessageV1.NewInternalMessageServiceClient(cli)
}
