package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	appV1 "go-wind-oa/api/gen/go/app/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"

	"go-wind-oa/pkg/middleware/auth"
)

// ExpenseService 是 app 边端的报销转发层（移动端）。列表强制按当前操作者过滤。
type ExpenseService struct {
	appV1.ExpenseServiceHTTPServer

	log *log.Helper

	expenseServiceClient oaV1.ExpenseServiceClient
}

func NewExpenseService(
	ctx *bootstrap.Context,
	expenseServiceClient oaV1.ExpenseServiceClient,
) *ExpenseService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "expense/service/app-service"))
	return &ExpenseService{
		log:                  l,
		expenseServiceClient: expenseServiceClient,
	}
}

func (s *ExpenseService) SubmitExpenseApplication(ctx context.Context, req *oaV1.SubmitExpenseApplicationRequest) (*oaV1.SubmitExpenseApplicationResponse, error) {
	return s.expenseServiceClient.SubmitExpenseApplication(ctx, req)
}

func (s *ExpenseService) ListExpenseApplications(ctx context.Context, req *oaV1.ListExpenseApplicationsRequest) (*oaV1.ListExpenseApplicationsResponse, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	// app 端强制按本人查，忽略/覆盖客户端传入的 user_id。
	req.UserId = operator.GetUserId()
	return s.expenseServiceClient.ListExpenseApplications(ctx, req)
}

func (s *ExpenseService) GetExpenseApplication(ctx context.Context, req *oaV1.GetExpenseApplicationRequest) (*oaV1.ExpenseApplication, error) {
	return s.expenseServiceClient.GetExpenseApplication(ctx, req)
}
