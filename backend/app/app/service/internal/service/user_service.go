package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	appV1 "go-wind-oa/api/gen/go/app/service/v1"
	identityV1 "go-wind-oa/api/gen/go/identity/service/v1"
)

// UserService 是 app 边端的通讯录用户查询转发层（移动端通讯录用，只读）。
type UserService struct {
	appV1.UserServiceHTTPServer

	log *log.Helper

	userServiceClient identityV1.UserServiceClient
}

func NewUserService(
	ctx *bootstrap.Context,
	userServiceClient identityV1.UserServiceClient,
) *UserService {
	return &UserService{
		log:              ctx.NewLoggerHelper("user/service/app-service"),
		userServiceClient: userServiceClient,
	}
}

func (s *UserService) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListUserResponse, error) {
	resp, err := s.userServiceClient.List(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
