package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-oa/app/core/service/internal/data"

	contentV1 "go-wind-oa/api/gen/go/content/service/v1"
)

type TagService struct {
	contentV1.UnimplementedTagServiceServer

	tagRepo *data.TagRepo
	log     *log.Helper
}

func NewTagService(ctx *bootstrap.Context, uc *data.TagRepo) *TagService {
	return &TagService{
		log:     ctx.NewLoggerHelper("tag/service/core-service"),
		tagRepo: uc,
	}
}

func (s *TagService) List(ctx context.Context, req *paginationV1.PagingRequest) (*contentV1.ListTagResponse, error) {
	return s.tagRepo.List(ctx, req)
}

func (s *TagService) Get(ctx context.Context, req *contentV1.GetTagRequest) (*contentV1.Tag, error) {
	return s.tagRepo.Get(ctx, req)
}

func (s *TagService) Create(ctx context.Context, req *contentV1.CreateTagRequest) (*contentV1.Tag, error) {
	return s.tagRepo.Create(ctx, req)
}

func (s *TagService) Update(ctx context.Context, req *contentV1.UpdateTagRequest) (*contentV1.Tag, error) {
	return s.tagRepo.Update(ctx, req)
}

func (s *TagService) Delete(ctx context.Context, req *contentV1.DeleteTagRequest) (*emptypb.Empty, error) {
	err := s.tagRepo.Delete(ctx, req)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *TagService) TranslationExists(ctx context.Context, req *contentV1.TagTranslationExistsRequest) (*contentV1.TagTranslationExistsResponse, error) {
	exists, err := s.tagRepo.TranslationExists(ctx, req.GetTagId(), req.GetLanguageCode())
	if err != nil {
		return nil, err
	}

	return &contentV1.TagTranslationExistsResponse{
		Exists: exists,
	}, nil
}

func (s *TagService) GetTranslation(ctx context.Context, req *contentV1.GetTagRequest) (*contentV1.TagTranslation, error) {
	return s.tagRepo.GetTranslation(ctx, req)
}

func (s *TagService) CreateTranslation(ctx context.Context, req *contentV1.CreateTagTranslationRequest) (*contentV1.TagTranslation, error) {
	return s.tagRepo.CreateTranslation(ctx, req)
}

func (s *TagService) UpdateTranslation(ctx context.Context, req *contentV1.UpdateTagTranslationRequest) (*contentV1.TagTranslation, error) {
	return s.tagRepo.UpdateTranslation(ctx, req)
}

func (s *TagService) DeleteTranslation(ctx context.Context, req *contentV1.DeleteTagTranslationRequest) (*emptypb.Empty, error) {
	err := s.tagRepo.DeleteTranslation(ctx, req)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
