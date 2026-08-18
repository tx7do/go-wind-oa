package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	internalMessageV1 "go-wind-oa/api/gen/go/internal_message/service/v1"

	appV1 "go-wind-oa/api/gen/go/app/service/v1"
)

// InternalMessageService 站内信转发层（app HTTP 边端）。
//
// 仅转发 ListMessage / GetMessage 至 core-service 的 gRPC 实现。
// 写操作（Send/Update/Delete/Revoke）不经 app 边端。
type InternalMessageService struct {
	appV1.InternalMessageServiceHTTPServer

	log *log.Helper

	internalMessageServiceClient internalMessageV1.InternalMessageServiceClient
}

func NewInternalMessageService(
	ctx *bootstrap.Context,
	internalMessageServiceClient internalMessageV1.InternalMessageServiceClient,
) *InternalMessageService {
	l := ctx.NewLoggerHelper("internal-message/service/app-service")
	return &InternalMessageService{
		log:                           l,
		internalMessageServiceClient: internalMessageServiceClient,
	}
}

func (s *InternalMessageService) ListMessage(ctx context.Context, req *paginationV1.PagingRequest) (*internalMessageV1.ListInternalMessageResponse, error) {
	return s.internalMessageServiceClient.ListMessage(ctx, req)
}

func (s *InternalMessageService) GetMessage(ctx context.Context, req *internalMessageV1.GetInternalMessageRequest) (*internalMessageV1.InternalMessage, error) {
	return s.internalMessageServiceClient.GetMessage(ctx, req)
}
