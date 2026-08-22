package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-oa/app/core/service/internal/data"

	identityV1 "go-wind-oa/api/gen/go/identity/service/v1"

	"go-wind-oa/pkg/metadata"
)

type PlanModuleService struct {
	identityV1.UnimplementedPlanModuleServiceServer

	log           *log.Helper
	planModuleRepo *data.PlanModuleRepo
}

func NewPlanModuleService(
	ctx *bootstrap.Context,
	planModuleRepo *data.PlanModuleRepo,
) *PlanModuleService {
	return &PlanModuleService{
		log:            ctx.NewLoggerHelper("plan-module/service/core-service"),
		planModuleRepo: planModuleRepo,
	}
}

func (s *PlanModuleService) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListPlanModuleResponse, error) {
	return s.planModuleRepo.List(ctx, req)
}

func (s *PlanModuleService) Get(ctx context.Context, req *identityV1.GetPlanModuleRequest) (*identityV1.PlanModule, error) {
	return s.planModuleRepo.Get(ctx, req)
}

func (s *PlanModuleService) Create(ctx context.Context, req *identityV1.CreatePlanModuleRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息（gRPC 元数据中的 OperatorMetadata）
	operator, err := metadata.FromServerContext(ctx)
	if err != nil {
		return nil, identityV1.ErrorUnauthorized("operator context missing")
	}

	req.Data.CreatedBy = trans.Ptr(uint32(operator.GetUserId()))

	if err = s.planModuleRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PlanModuleService) Update(ctx context.Context, req *identityV1.UpdatePlanModuleRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息（gRPC 元数据中的 OperatorMetadata）
	operator, err := metadata.FromServerContext(ctx)
	if err != nil {
		return nil, identityV1.ErrorUnauthorized("operator context missing")
	}

	req.Data.Id = trans.Ptr(req.GetId())

	req.Data.UpdatedBy = trans.Ptr(uint32(operator.GetUserId()))
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	if err = s.planModuleRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PlanModuleService) Delete(ctx context.Context, req *identityV1.DeletePlanModuleRequest) (*emptypb.Empty, error) {
	if err := s.planModuleRepo.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
