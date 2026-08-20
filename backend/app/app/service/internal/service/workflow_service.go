package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	appV1 "go-wind-oa/api/gen/go/app/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// WorkflowService 是 app 边端的工作流转发层（移动端）。
// HTTP 请求经鉴权后透传至 core-service 的 gRPC WorkflowService。
type WorkflowService struct {
	appV1.WorkflowServiceHTTPServer

	log *log.Helper

	workflowServiceClient oaV1.WorkflowServiceClient
}

func NewWorkflowService(
	ctx *bootstrap.Context,
	workflowServiceClient oaV1.WorkflowServiceClient,
) *WorkflowService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "workflow/service/app-service"))
	return &WorkflowService{
		log:                   l,
		workflowServiceClient: workflowServiceClient,
	}
}

func (s *WorkflowService) SubmitApply(ctx context.Context, req *oaV1.SubmitApplyRequest) (*oaV1.SubmitApplyResponse, error) {
	resp, err := s.workflowServiceClient.SubmitApply(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *WorkflowService) AuditTask(ctx context.Context, req *oaV1.AuditTaskRequest) (*emptypb.Empty, error) {
	resp, err := s.workflowServiceClient.AuditTask(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *WorkflowService) WithdrawApply(ctx context.Context, req *oaV1.WithdrawApplyRequest) (*emptypb.Empty, error) {
	resp, err := s.workflowServiceClient.WithdrawApply(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *WorkflowService) GetMyTasks(ctx context.Context, req *oaV1.GetMyTasksRequest) (*oaV1.GetMyTasksResponse, error) {
	resp, err := s.workflowServiceClient.GetMyTasks(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *WorkflowService) GetTask(ctx context.Context, req *oaV1.GetTaskRequest) (*oaV1.GetTaskResponse, error) {
	resp, err := s.workflowServiceClient.GetTask(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
