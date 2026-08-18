package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	adminV1 "go-wind-oa/api/gen/go/admin/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// WorkflowService admin-service 側的工作流 HTTP 邊端轉發層。
//
// 對齊 cms admin-service 的 i_*.proto HTTP 邊端轉發模式：admin-service
// 持有由 admin/service/v1/i_workflow.proto 生成的 HTTP server interface，
// 收到請求後原樣轉發到 core-service 的 WorkflowService gRPC 實現（狀態機
// 落庫）。本層不做業務邏輯，僅做協議邊界轉換——與 cms admin-service 的
// InternalMessageService / ApiService 等轉發服務同構。
//
// 鑑權 / 租戶隔離由 admin-service 的 REST 中間件鏈
// （auth.Server + ent.Server）在請求進入本層前完成：auth 注入
// OperatorMetadata，ent 據此構建帶租戶作用域的 viewer，core-service
// 的 TenantPrivacy 策略再按 viewer 做行級隔離。
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

func (s *WorkflowService) CreateWorkflowDefinition(ctx context.Context, req *oaV1.CreateWorkflowDefinitionRequest) (*oaV1.WorkflowDefinition, error) {
	return s.workflowServiceClient.CreateWorkflowDefinition(ctx, req)
}

func (s *WorkflowService) ListWorkflowDefinition(ctx context.Context, req *oaV1.ListWorkflowDefinitionRequest) (*oaV1.ListWorkflowDefinitionResponse, error) {
	return s.workflowServiceClient.ListWorkflowDefinition(ctx, req)
}

func (s *WorkflowService) UpdateWorkflowDefinition(ctx context.Context, req *oaV1.UpdateWorkflowDefinitionRequest) (*oaV1.WorkflowDefinition, error) {
	return s.workflowServiceClient.UpdateWorkflowDefinition(ctx, req)
}

func (s *WorkflowService) GetWorkflowDefinition(ctx context.Context, req *oaV1.GetWorkflowDefinitionRequest) (*oaV1.WorkflowDefinition, error) {
	return s.workflowServiceClient.GetWorkflowDefinition(ctx, req)
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
