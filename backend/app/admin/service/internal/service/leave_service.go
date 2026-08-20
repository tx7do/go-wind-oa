package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	adminV1 "go-wind-oa/api/gen/go/admin/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// LeaveService 是 admin 边端的请假管理转发层（类型/额度管理 + 申请查询）。
type LeaveService struct {
	adminV1.LeaveServiceHTTPServer

	log *log.Helper

	leaveServiceClient oaV1.LeaveServiceClient
}

func NewLeaveService(
	ctx *bootstrap.Context,
	leaveServiceClient oaV1.LeaveServiceClient,
) *LeaveService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "leave/service/admin-service"))
	return &LeaveService{
		log:                l,
		leaveServiceClient: leaveServiceClient,
	}
}

func (s *LeaveService) CreateLeaveType(ctx context.Context, req *oaV1.CreateLeaveTypeRequest) (*oaV1.LeaveType, error) {
	return s.leaveServiceClient.CreateLeaveType(ctx, req)
}

func (s *LeaveService) ListLeaveTypes(ctx context.Context, req *paginationV1.PagingRequest) (*oaV1.ListLeaveTypesResponse, error) {
	return s.leaveServiceClient.ListLeaveTypes(ctx, req)
}

func (s *LeaveService) GrantLeaveBalance(ctx context.Context, req *oaV1.GrantLeaveBalanceRequest) (*emptypb.Empty, error) {
	return s.leaveServiceClient.GrantLeaveBalance(ctx, req)
}

func (s *LeaveService) ListLeaveBalances(ctx context.Context, req *oaV1.ListLeaveBalancesRequest) (*oaV1.ListLeaveBalancesResponse, error) {
	return s.leaveServiceClient.ListLeaveBalances(ctx, req)
}

func (s *LeaveService) ListLeaveApplications(ctx context.Context, req *oaV1.ListLeaveApplicationsRequest) (*oaV1.ListLeaveApplicationsResponse, error) {
	return s.leaveServiceClient.ListLeaveApplications(ctx, req)
}

func (s *LeaveService) GetLeaveApplication(ctx context.Context, req *oaV1.GetLeaveApplicationRequest) (*oaV1.LeaveApplication, error) {
	return s.leaveServiceClient.GetLeaveApplication(ctx, req)
}
