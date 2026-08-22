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

type PlanQuotaService struct {
	identityV1.UnimplementedPlanQuotaServiceServer

	log          *log.Helper
	planQuotaRepo *data.PlanQuotaRepo
}

func NewPlanQuotaService(
	ctx *bootstrap.Context,
	planQuotaRepo *data.PlanQuotaRepo,
) *PlanQuotaService {
	return &PlanQuotaService{
		log:           ctx.NewLoggerHelper("plan-quota/service/core-service"),
		planQuotaRepo: planQuotaRepo,
	}
}

func (s *PlanQuotaService) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListPlanQuotaResponse, error) {
	return s.planQuotaRepo.List(ctx, req)
}

func (s *PlanQuotaService) Get(ctx context.Context, req *identityV1.GetPlanQuotaRequest) (*identityV1.PlanQuota, error) {
	return s.planQuotaRepo.Get(ctx, req)
}

func (s *PlanQuotaService) Create(ctx context.Context, req *identityV1.CreatePlanQuotaRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息（gRPC 元数据中的 OperatorMetadata）
	operator, err := metadata.FromServerContext(ctx)
	if err != nil {
		return nil, identityV1.ErrorUnauthorized("operator context missing")
	}

	req.Data.CreatedBy = trans.Ptr(uint32(operator.GetUserId()))

	if err = s.planQuotaRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PlanQuotaService) Update(ctx context.Context, req *identityV1.UpdatePlanQuotaRequest) (*emptypb.Empty, error) {
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

	if err = s.planQuotaRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PlanQuotaService) Delete(ctx context.Context, req *identityV1.DeletePlanQuotaRequest) (*emptypb.Empty, error) {
	if err := s.planQuotaRepo.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
