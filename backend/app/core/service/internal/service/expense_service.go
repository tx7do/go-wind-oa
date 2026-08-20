package service

import (
	"context"
	"encoding/json"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-oa/app/core/service/internal/data"
	"go-wind-oa/app/core/service/internal/data/ent/expenseapplication"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

const (
	businessTypeExpense = "EXPENSE"

	expenseDefinitionCode    = "EXPENSE"
	expenseDefinitionVersion = int32(1)
)

type ExpenseService struct {
	oaV1.UnimplementedExpenseServiceServer

	log *log.Helper

	appRepo *data.ExpenseApplicationRepo

	definitionRepo *data.WorkflowDefinitionRepo
	resolverRepo   *data.WorkflowResolverRepo

	workflowService *WorkflowService
}

func NewExpenseService(
	ctx *bootstrap.Context,
	appRepo *data.ExpenseApplicationRepo,
	definitionRepo *data.WorkflowDefinitionRepo,
	resolverRepo *data.WorkflowResolverRepo,
	workflowService *WorkflowService,
	eventRegistry *WorkflowEventRegistry,
) *ExpenseService {
	svc := &ExpenseService{
		log:             ctx.NewLoggerHelper("expense/service/core-service"),
		appRepo:         appRepo,
		definitionRepo:  definitionRepo,
		resolverRepo:    resolverRepo,
		workflowService: workflowService,
	}
	// 注册工作流终结回调：报销审批通过后仅同步状态（ERP 应付联动为未来扩展点）。
	eventRegistry.Register(businessTypeExpense, svc.onInstanceTerminal)
	return svc
}

// fillApplicantNames 批量回填申请人姓名（失败仅缺姓名，不影响列表）。
func (s *ExpenseService) fillApplicantNames(ctx context.Context, tid uint32, items []*oaV1.ExpenseApplication) {
	ids := make([]uint32, 0, len(items))
	for _, it := range items {
		if it.GetCreatedBy() != 0 {
			ids = append(ids, it.GetCreatedBy())
		}
	}
	names, err := s.resolverRepo.ResolveUsernames(ctx, tid, ids)
	if err != nil {
		return
	}
	for _, it := range items {
		if name, ok := names[it.GetCreatedBy()]; ok {
			it.ApplicantName = trans.Ptr(name)
		}
	}
}

// ensureWorkflowDefinition EXPENSE 流程定义兜底：租户内不存在时按默认模板创建并启用
// （提交给申请人主管 LEADER，会签）。并发首提撞唯一索引时忽略，回读校验。
func (s *ExpenseService) ensureWorkflowDefinition(ctx context.Context, tid, uid uint32) {
	if _, err := s.definitionRepo.GetByCodeVersion(ctx, expenseDefinitionCode, expenseDefinitionVersion); err == nil {
		return
	}
	nodeConfig := `[{"approvers":[{"type":"LEADER"}],"strategy":"ALL"}]`
	_, err := s.definitionRepo.Create(ctx, &oaV1.CreateWorkflowDefinitionRequest{
		Data: &oaV1.WorkflowDefinition{
			Code:             trans.Ptr(expenseDefinitionCode),
			Version:          trans.Ptr(expenseDefinitionVersion),
			NodeConfig:       trans.Ptr(nodeConfig),
			DefinitionStatus: oaV1.WorkflowDefinition_ENABLED.Enum(),
			TenantId:         trans.Ptr(tid),
			CreatedBy:        trans.Ptr(uid),
		},
	})
	if err != nil {
		s.log.Warnf("bootstrap EXPENSE definition failed (may conflict): %s", err.Error())
	}
}

// SubmitExpenseApplication 提交报销申请：建单（含明细与发票文件引用）→ 进程内提交
// EXPENSE 工作流。EXPENSE 流程定义不存在时按默认模板自动创建并启用。
func (s *ExpenseService) SubmitExpenseApplication(ctx context.Context, req *oaV1.SubmitExpenseApplicationRequest) (*oaV1.SubmitExpenseApplicationResponse, error) {
	tid, uid, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	if req.GetTitle() == "" || len(req.GetItems()) == 0 {
		return nil, oaV1.ErrorBadRequest("invalid parameter")
	}
	for _, item := range req.GetItems() {
		if item.GetAmount() <= 0 {
			return nil, oaV1.ErrorBadRequest("invalid item amount")
		}
	}

	appID, total, err := s.appRepo.CreateWithItems(ctx, tid, uid, req.GetTitle(), req.GetItems())
	if err != nil {
		return nil, err
	}

	form, _ := json.Marshal(map[string]any{
		"business":    "费用报销",
		"title":       req.GetTitle(),
		"totalAmount": total,
		"itemCount":   len(req.GetItems()),
	})
	s.ensureWorkflowDefinition(ctx, tid, uid)
	resp, err := s.workflowService.SubmitApply(ctx, &oaV1.SubmitApplyRequest{
		Code:         expenseDefinitionCode,
		Version:      expenseDefinitionVersion,
		FormData:     string(form),
		BusinessType: trans.Ptr(businessTypeExpense),
		BusinessId:   trans.Ptr(appID),
	})
	if err != nil {
		_ = s.appRepo.UpdateStatus(ctx, tid, appID, oaV1.ExpenseApplication_REJECTED)
		return nil, err
	}
	if err := s.appRepo.SetInstanceID(ctx, tid, appID, resp.GetInstanceId()); err != nil {
		return nil, err
	}
	return &oaV1.SubmitExpenseApplicationResponse{Id: appID, InstanceId: resp.GetInstanceId()}, nil
}

func (s *ExpenseService) ListExpenseApplications(ctx context.Context, req *oaV1.ListExpenseApplicationsRequest) (*oaV1.ListExpenseApplicationsResponse, error) {
	tid, _, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	items, total, err := s.appRepo.List(ctx, tid, req.GetUserId(), req.GetStatus(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	s.fillApplicantNames(ctx, tid, items)
	return &oaV1.ListExpenseApplicationsResponse{Items: items, Total: uint64(total)}, nil
}

func (s *ExpenseService) GetExpenseApplication(ctx context.Context, req *oaV1.GetExpenseApplicationRequest) (*oaV1.ExpenseApplication, error) {
	tid, _, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	if req.GetId() == 0 {
		return nil, oaV1.ErrorBadRequest("invalid parameter")
	}
	item, err := s.appRepo.Get(ctx, tid, req.GetId())
	if err != nil {
		return nil, err
	}
	s.fillApplicantNames(ctx, tid, []*oaV1.ExpenseApplication{item})
	return item, nil
}

// onInstanceTerminal 工作流终结回调。校验关联+幂等；ERP 应付款联动是未来扩展点，
// 当前仅同步申请单状态。
func (s *ExpenseService) onInstanceTerminal(ctx context.Context, tenantID, instanceID, businessID uint32, status oaV1.WorkflowInstance_InstanceStatus) {
	entity, err := s.appRepo.GetEntity(ctx, tenantID, businessID)
	if err != nil || entity == nil {
		s.log.Errorf("expense hook load application %d failed: %v", businessID, err)
		return
	}
	if entity.InstanceID == nil || *entity.InstanceID != instanceID {
		return
	}
	if entity.ExpenseStatus == nil || *entity.ExpenseStatus != expenseapplication.ExpenseStatusPending {
		return
	}

	switch status {
	case oaV1.WorkflowInstance_APPROVED:
		if err := s.appRepo.UpdateStatus(ctx, tenantID, businessID, oaV1.ExpenseApplication_APPROVED); err != nil {
			s.log.Errorf("expense hook approve application %d failed: %s", businessID, err.Error())
		}
	case oaV1.WorkflowInstance_REJECTED:
		if err := s.appRepo.UpdateStatus(ctx, tenantID, businessID, oaV1.ExpenseApplication_REJECTED); err != nil {
			s.log.Errorf("expense hook reject application %d failed: %s", businessID, err.Error())
		}
	case oaV1.WorkflowInstance_WITHDRAWN:
		if err := s.appRepo.UpdateStatus(ctx, tenantID, businessID, oaV1.ExpenseApplication_WITHDRAWN); err != nil {
			s.log.Errorf("expense hook withdraw application %d failed: %s", businessID, err.Error())
		}
	}
}
