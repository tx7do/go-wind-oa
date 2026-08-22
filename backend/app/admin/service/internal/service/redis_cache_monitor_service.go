package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	adminV1 "go-wind-oa/api/gen/go/admin/service/v1"
	redisCacheV1 "go-wind-oa/api/gen/go/redis_cache/service/v1"
)

// RedisCacheMonitorService 提供 Redis 运行时只读监控视图的 HTTP 接口（转发 core）。
type RedisCacheMonitorService struct {
	adminV1.RedisCacheMonitorServiceHTTPServer

	log *log.Helper

	redisCacheMonitorServiceClient redisCacheV1.RedisCacheMonitorServiceClient
}

func NewRedisCacheMonitorService(
	ctx *bootstrap.Context,
	redisCacheMonitorServiceClient redisCacheV1.RedisCacheMonitorServiceClient,
) *RedisCacheMonitorService {
	return &RedisCacheMonitorService{
		log:                            ctx.NewLoggerHelper("redis-cache-monitor/service/admin-service"),
		redisCacheMonitorServiceClient: redisCacheMonitorServiceClient,
	}
}

// Get 返回 Redis 缓存监控信息（INFO/DBSIZE/SLOWLOG 聚合视图）。
func (s *RedisCacheMonitorService) Get(ctx context.Context, req *redisCacheV1.GetRedisCacheMonitorRequest) (*redisCacheV1.RedisCacheMonitorInfo, error) {
	return s.redisCacheMonitorServiceClient.Get(ctx, req)
}
