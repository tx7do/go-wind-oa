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

// PlanModuleService 套餐功能模块管理服务（HTTP BFF，转发 core PlanModuleService）。
type PlanModuleService struct {
	adminV1.PlanModuleServiceHTTPServer

	log *log.Helper

	planModuleServiceClient identityV1.PlanModuleServiceClient
}

func NewPlanModuleService(ctx *bootstrap.Context, planModuleServiceClient identityV1.PlanModuleServiceClient) *PlanModuleService {
	return &PlanModuleService{
		log:                    ctx.NewLoggerHelper("plan-module/service/admin-service"),
		planModuleServiceClient: planModuleServiceClient,
	}
}

func (s *PlanModuleService) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListPlanModuleResponse, error) {
	return s.planModuleServiceClient.List(ctx, req)
}

func (s *PlanModuleService) Get(ctx context.Context, req *identityV1.GetPlanModuleRequest) (*identityV1.PlanModule, error) {
	return s.planModuleServiceClient.Get(ctx, req)
}

func (s *PlanModuleService) Create(ctx context.Context, req *identityV1.CreatePlanModuleRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	return s.planModuleServiceClient.Create(ctx, req)
}

func (s *PlanModuleService) Update(ctx context.Context, req *identityV1.UpdatePlanModuleRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	id := req.GetId()
	req.Data.Id = &id
	return s.planModuleServiceClient.Update(ctx, req)
}

func (s *PlanModuleService) Delete(ctx context.Context, req *identityV1.DeletePlanModuleRequest) (*emptypb.Empty, error) {
	return s.planModuleServiceClient.Delete(ctx, req)
}
