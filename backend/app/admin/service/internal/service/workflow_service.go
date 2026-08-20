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

// WorkflowService 是 admin 边端的工作流转发层。
// HTTP 请求经鉴权后透传至 core-service 的 gRPC WorkflowService。
type WorkflowService struct {
	adminV1.WorkflowServiceHTTPServer

	log *log.Helper

	workflowServiceClient oaV1.WorkflowServiceClient
}

func NewWorkflowService(
	ctx *bootstrap.Context,
	workflowServiceClient oaV1.WorkflowServiceClient,
) *WorkflowService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "workflow/service/admin-service"))
	return &WorkflowService{
		log:                   l,
		workflowServiceClient: workflowServiceClient,
	}
}

func (s *WorkflowService) ListWorkflowDefinition(ctx context.Context, req *paginationV1.PagingRequest) (*oaV1.ListWorkflowDefinitionResponse, error) {
	resp, err := s.workflowServiceClient.ListWorkflowDefinition(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *WorkflowService) GetWorkflowDefinition(ctx context.Context, req *oaV1.GetWorkflowDefinitionRequest) (*oaV1.WorkflowDefinition, error) {
	resp, err := s.workflowServiceClient.GetWorkflowDefinition(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *WorkflowService) CreateWorkflowDefinition(ctx context.Context, req *oaV1.CreateWorkflowDefinitionRequest) (*oaV1.WorkflowDefinition, error) {
	resp, err := s.workflowServiceClient.CreateWorkflowDefinition(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *WorkflowService) UpdateWorkflowDefinition(ctx context.Context, req *oaV1.UpdateWorkflowDefinitionRequest) (*emptypb.Empty, error) {
	resp, err := s.workflowServiceClient.UpdateWorkflowDefinition(ctx, req)
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

func (s *WorkflowService) AuditTask(ctx context.Context, req *oaV1.AuditTaskRequest) (*emptypb.Empty, error) {
	resp, err := s.workflowServiceClient.AuditTask(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
