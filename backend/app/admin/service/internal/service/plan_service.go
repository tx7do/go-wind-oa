package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	adminV1 "go-wind-oa/api/gen/go/admin/service/v1"
	identityV1 "go-wind-oa/api/gen/go/identity/service/v1"
)

// PlanService 套餐管理服务（HTTP BFF，转发 core PlanService）。
type PlanService struct {
	adminV1.PlanServiceHTTPServer

	log *log.Helper

	planServiceClient identityV1.PlanServiceClient
}

func NewPlanService(ctx *bootstrap.Context, planServiceClient identityV1.PlanServiceClient) *PlanService {
	return &PlanService{
		log:               ctx.NewLoggerHelper("plan/service/admin-service"),
		planServiceClient: planServiceClient,
	}
}

func (s *PlanService) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListPlanResponse, error) {
	return s.planServiceClient.List(ctx, req)
}

func (s *PlanService) Get(ctx context.Context, req *identityV1.GetPlanRequest) (*identityV1.Plan, error) {
	return s.planServiceClient.Get(ctx, req)
}

func (s *PlanService) Create(ctx context.Context, req *identityV1.CreatePlanRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	return s.planServiceClient.Create(ctx, req)
}

func (s *PlanService) Update(ctx context.Context, req *identityV1.UpdatePlanRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	id := req.GetId()
	req.Data.Id = &id
	return s.planServiceClient.Update(ctx, req)
}

func (s *PlanService) Delete(ctx context.Context, req *identityV1.DeletePlanRequest) (*emptypb.Empty, error) {
	return s.planServiceClient.Delete(ctx, req)
}
