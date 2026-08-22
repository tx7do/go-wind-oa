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

// PlanQuotaService 套餐配额管理服务（HTTP BFF，转发 core PlanQuotaService）。
type PlanQuotaService struct {
	adminV1.PlanQuotaServiceHTTPServer

	log *log.Helper

	planQuotaServiceClient identityV1.PlanQuotaServiceClient
}

func NewPlanQuotaService(ctx *bootstrap.Context, planQuotaServiceClient identityV1.PlanQuotaServiceClient) *PlanQuotaService {
	return &PlanQuotaService{
		log:                    ctx.NewLoggerHelper("plan-quota/service/admin-service"),
		planQuotaServiceClient: planQuotaServiceClient,
	}
}

func (s *PlanQuotaService) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListPlanQuotaResponse, error) {
	return s.planQuotaServiceClient.List(ctx, req)
}

func (s *PlanQuotaService) Get(ctx context.Context, req *identityV1.GetPlanQuotaRequest) (*identityV1.PlanQuota, error) {
	return s.planQuotaServiceClient.Get(ctx, req)
}

func (s *PlanQuotaService) Create(ctx context.Context, req *identityV1.CreatePlanQuotaRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	return s.planQuotaServiceClient.Create(ctx, req)
}

func (s *PlanQuotaService) Update(ctx context.Context, req *identityV1.UpdatePlanQuotaRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	id := req.GetId()
	req.Data.Id = &id
	return s.planQuotaServiceClient.Update(ctx, req)
}

func (s *PlanQuotaService) Delete(ctx context.Context, req *identityV1.DeletePlanQuotaRequest) (*emptypb.Empty, error) {
	return s.planQuotaServiceClient.Delete(ctx, req)
}
