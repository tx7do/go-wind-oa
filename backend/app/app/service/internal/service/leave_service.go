package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	appV1 "go-wind-oa/api/gen/go/app/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"

	"go-wind-oa/pkg/middleware/auth"
)

// LeaveService 是 app 边端的请假转发层（移动端）。
// 列表/额度强制按当前操作者过滤（覆盖 user_id 参数，防越权）。
type LeaveService struct {
	appV1.LeaveServiceHTTPServer

	log *log.Helper

	leaveServiceClient oaV1.LeaveServiceClient
}

func NewLeaveService(
	ctx *bootstrap.Context,
	leaveServiceClient oaV1.LeaveServiceClient,
) *LeaveService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "leave/service/app-service"))
	return &LeaveService{
		log:                l,
		leaveServiceClient: leaveServiceClient,
	}
}

func (s *LeaveService) SubmitLeaveApplication(ctx context.Context, req *oaV1.SubmitLeaveApplicationRequest) (*oaV1.SubmitLeaveApplicationResponse, error) {
	return s.leaveServiceClient.SubmitLeaveApplication(ctx, req)
}

func (s *LeaveService) ListLeaveApplications(ctx context.Context, req *oaV1.ListLeaveApplicationsRequest) (*oaV1.ListLeaveApplicationsResponse, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	// app 端强制按本人查，忽略/覆盖客户端传入的 user_id。
	req.UserId = operator.GetUserId()
	return s.leaveServiceClient.ListLeaveApplications(ctx, req)
}

func (s *LeaveService) GetLeaveApplication(ctx context.Context, req *oaV1.GetLeaveApplicationRequest) (*oaV1.LeaveApplication, error) {
	return s.leaveServiceClient.GetLeaveApplication(ctx, req)
}

func (s *LeaveService) ListLeaveTypes(ctx context.Context, req *paginationV1.PagingRequest) (*oaV1.ListLeaveTypesResponse, error) {
	return s.leaveServiceClient.ListLeaveTypes(ctx, req)
}

func (s *LeaveService) ListLeaveBalances(ctx context.Context, req *oaV1.ListLeaveBalancesRequest) (*oaV1.ListLeaveBalancesResponse, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	// app 端强制查本人额度。
	req.UserId = operator.GetUserId()
	return s.leaveServiceClient.ListLeaveBalances(ctx, req)
}
