package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-oa/app/core/service/internal/data"

	redisCacheV1 "go-wind-oa/api/gen/go/redis_cache/service/v1"
)

// RedisCacheMonitorService 提供 Redis 运行时只读监控视图的 gRPC 接口（admin BFF 转发）。
// 纯透传至 RedisCacheMonitorRepo，不做任何业务加工——对应审计日志 Get 的极简形态。
type RedisCacheMonitorService struct {
	redisCacheV1.UnimplementedRedisCacheMonitorServiceServer

	log  *log.Helper
	repo *data.RedisCacheMonitorRepo
}

// NewRedisCacheMonitorService 构造 Redis 缓存监控服务。
func NewRedisCacheMonitorService(
	ctx *bootstrap.Context,
	repo *data.RedisCacheMonitorRepo,
) *RedisCacheMonitorService {
	return &RedisCacheMonitorService{
		log:  ctx.NewLoggerHelper("redis-cache-monitor/service/core-service"),
		repo: repo,
	}
}

// Get 返回 Redis 缓存监控信息（INFO/DBSIZE/SLOWLOG 聚合视图）。
func (s *RedisCacheMonitorService) Get(ctx context.Context, _ *redisCacheV1.GetRedisCacheMonitorRequest) (*redisCacheV1.RedisCacheMonitorInfo, error) {
	return s.repo.GetInfo(ctx)
}
