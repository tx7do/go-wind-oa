package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	adminV1 "go-wind-oa/api/gen/go/admin/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// ExpenseService 是 admin 边端的报销管理转发层（申请查询）。
type ExpenseService struct {
	adminV1.ExpenseServiceHTTPServer

	log *log.Helper

	expenseServiceClient oaV1.ExpenseServiceClient
}

func NewExpenseService(
	ctx *bootstrap.Context,
	expenseServiceClient oaV1.ExpenseServiceClient,
) *ExpenseService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "expense/service/admin-service"))
	return &ExpenseService{
		log:                  l,
		expenseServiceClient: expenseServiceClient,
	}
}

func (s *ExpenseService) ListExpenseApplications(ctx context.Context, req *oaV1.ListExpenseApplicationsRequest) (*oaV1.ListExpenseApplicationsResponse, error) {
	return s.expenseServiceClient.ListExpenseApplications(ctx, req)
}

func (s *ExpenseService) GetExpenseApplication(ctx context.Context, req *oaV1.GetExpenseApplicationRequest) (*oaV1.ExpenseApplication, error) {
	return s.expenseServiceClient.GetExpenseApplication(ctx, req)
}
