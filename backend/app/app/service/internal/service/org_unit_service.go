package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	appV1 "go-wind-oa/api/gen/go/app/service/v1"
	identityV1 "go-wind-oa/api/gen/go/identity/service/v1"
)

// OrgUnitService 是 app 边端的组织单元查询转发层（移动端通讯录用，只读）。
type OrgUnitService struct {
	appV1.OrgUnitServiceHTTPServer

	log *log.Helper

	orgUnitServiceClient identityV1.OrgUnitServiceClient
}

func NewOrgUnitService(
	ctx *bootstrap.Context,
	orgUnitServiceClient identityV1.OrgUnitServiceClient,
) *OrgUnitService {
	return &OrgUnitService{
		log:                  ctx.NewLoggerHelper("org-unit/service/app-service"),
		orgUnitServiceClient: orgUnitServiceClient,
	}
}

func (s *OrgUnitService) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListOrgUnitResponse, error) {
	resp, err := s.orgUnitServiceClient.List(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *OrgUnitService) Get(ctx context.Context, req *identityV1.GetOrgUnitRequest) (*identityV1.OrgUnit, error) {
	resp, err := s.orgUnitServiceClient.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
