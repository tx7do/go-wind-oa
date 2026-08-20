package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-oa/app/core/service/internal/data"

	siteV1 "go-wind-oa/api/gen/go/site/service/v1"
)

type NavigationItemService struct {
	siteV1.UnimplementedNavigationItemServiceServer

	navigationItemRepo *data.NavigationItemRepo
	log                *log.Helper
}

func NewNavigationItemService(ctx *bootstrap.Context, uc *data.NavigationItemRepo) *NavigationItemService {
	return &NavigationItemService{
		log:                ctx.NewLoggerHelper("navigation-item/service/core-service"),
		navigationItemRepo: uc,
	}
}

func (s *NavigationItemService) List(ctx context.Context, req *paginationV1.PagingRequest) (*siteV1.ListNavigationItemResponse, error) {
	return s.navigationItemRepo.List(ctx, req)
}

func (s *NavigationItemService) Get(ctx context.Context, req *siteV1.GetNavigationItemRequest) (*siteV1.NavigationItem, error) {
	return s.navigationItemRepo.Get(ctx, req)
}

func (s *NavigationItemService) Create(ctx context.Context, req *siteV1.CreateNavigationItemRequest) (*siteV1.NavigationItem, error) {
	return s.navigationItemRepo.Create(ctx, req.GetData())
}

func (s *NavigationItemService) Update(ctx context.Context, req *siteV1.UpdateNavigationItemRequest) (*siteV1.NavigationItem, error) {
	return s.navigationItemRepo.Update(ctx, req)
}

func (s *NavigationItemService) Delete(ctx context.Context, req *siteV1.DeleteNavigationItemRequest) (*emptypb.Empty, error) {
	err := s.navigationItemRepo.Delete(ctx, req)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
