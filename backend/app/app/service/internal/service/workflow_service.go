package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
	appV1 "go-wind-oa/api/gen/go/app/service/v1"
)

// WorkflowService app-service 側的工作流 HTTP 邊端轉發層。
//
// 與 admin-service 的 WorkflowService 同構：持有 HTTP 邊端（由 buf 生成的
// appV1.WorkflowServiceHTTPServer），收到請求後原樣轉發到 core-service 的
// WorkflowService gRPC 實現。本層不做業務邏輯，僅做協議邊界轉換。
//
// 鑑權 / 租戶隔離由 app-service 的 REST 中間件鏈在請求進入本層前完成。
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
	return s.workflowServiceClient.SubmitApply(ctx, req)
}

func (s *WorkflowService) AuditTask(ctx context.Context, req *oaV1.AuditTaskRequest) (*emptypb.Empty, error) {
	return s.workflowServiceClient.AuditTask(ctx, req)
}

func (s *WorkflowService) GetMyTasks(ctx context.Context, req *oaV1.GetMyTasksRequest) (*oaV1.GetMyTasksResponse, error) {
	return s.workflowServiceClient.GetMyTasks(ctx, req)
}

func (s *WorkflowService) GetTask(ctx context.Context, req *oaV1.GetTaskRequest) (*oaV1.GetTaskResponse, error) {
	return s.workflowServiceClient.GetTask(ctx, req)
}
