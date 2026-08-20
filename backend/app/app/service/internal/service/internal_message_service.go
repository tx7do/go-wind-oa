package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	appV1 "go-wind-oa/api/gen/go/app/service/v1"
	internalMessageV1 "go-wind-oa/api/gen/go/internal_message/service/v1"
)

// InternalMessageService 是 app 边端的站内信只读转发层（移动端收件箱）。
// 收件人过滤由 core 侧按 viewer 上下文强制。
type InternalMessageService struct {
	appV1.InternalMessageServiceHTTPServer

	log *log.Helper

	internalMessageServiceClient internalMessageV1.InternalMessageServiceClient
}

func NewInternalMessageService(
	ctx *bootstrap.Context,
	internalMessageServiceClient internalMessageV1.InternalMessageServiceClient,
) *InternalMessageService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "internal-message/service/app-service"))
	return &InternalMessageService{
		log:                          l,
		internalMessageServiceClient: internalMessageServiceClient,
	}
}

func (s *InternalMessageService) ListMyMessages(ctx context.Context, req *internalMessageV1.ListMyMessagesRequest) (*internalMessageV1.ListInternalMessageResponse, error) {
	resp, err := s.internalMessageServiceClient.ListMyMessages(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
