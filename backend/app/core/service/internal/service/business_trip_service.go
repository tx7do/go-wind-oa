package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-oa/app/core/service/internal/data"
	"go-wind-oa/app/core/service/internal/data/ent/businesstripapplication"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// 业务单据类型与内置流程定义代码。
const (
	businessTypeBusinessTrip = "BUSINESS_TRIP"

	businessTripDefinitionCode    = "BUSINESS_TRIP"
	businessTripDefinitionVersion = int32(1)
)

type BusinessTripService struct {
	oaV1.UnimplementedBusinessTripServiceServer

	log *log.Helper

	appRepo        *data.BusinessTripApplicationRepo
	definitionRepo *data.WorkflowDefinitionRepo
	resolverRepo   *data.WorkflowResolverRepo

	workflowService *WorkflowService
}

func NewBusinessTripService(
	ctx *bootstrap.Context,
	appRepo *data.BusinessTripApplicationRepo,
	definitionRepo *data.WorkflowDefinitionRepo,
	resolverRepo *data.WorkflowResolverRepo,
	workflowService *WorkflowService,
	eventRegistry *WorkflowEventRegistry,
) *BusinessTripService {
	svc := &BusinessTripService{
		log:             ctx.NewLoggerHelper("business-trip/service/core-service"),
		appRepo:         appRepo,
		definitionRepo:  definitionRepo,
		resolverRepo:    resolverRepo,
		workflowService: workflowService,
	}
	// 注册工作流终结回调：出差申请无额度副作用，三态仅同步单据状态。
	eventRegistry.Register(businessTypeBusinessTrip, svc.onInstanceTerminal)
	return svc
}

// fillApplicantNames 批量回填申请人姓名（失败仅缺姓名，不影响列表）。
func (s *BusinessTripService) fillApplicantNames(ctx context.Context, tid uint32, items []*oaV1.BusinessTripApplication) {
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

// SubmitBusinessTripApplication 提交出差申请：建申请单 → 进程内提交 BUSINESS_TRIP 工作流。
// BUSINESS_TRIP 流程定义不存在时按默认模板（提交给申请人主管，会签）自动创建并启用。
func (s *BusinessTripService) SubmitBusinessTripApplication(ctx context.Context, req *oaV1.SubmitBusinessTripApplicationRequest) (*oaV1.SubmitBusinessTripApplicationResponse, error) {
	tid, uid, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	if req.GetTitle() == "" || req.GetDestination() == "" || req.GetStartDate() == nil || req.GetEndDate() == nil {
		return nil, oaV1.ErrorBadRequest("invalid parameter")
	}

	// 日期统一截断到服务器本地零点（与请假一致，避免跨时区错位）。
	start := truncateDate(req.GetStartDate().AsTime().In(time.Local))
	end := truncateDate(req.GetEndDate().AsTime().In(time.Local))
	if end.Before(start) {
		return nil, oaV1.ErrorBadRequest("end date before start date")
	}

	appID, err := s.appRepo.Create(ctx, tid, uid, req.GetTitle(), req.GetDestination(), start, end, req.GetItinerary())
	if err != nil {
		return nil, err
	}

	form, _ := json.Marshal(map[string]any{
		"business":    "出差申请",
		"title":       req.GetTitle(),
		"destination": req.GetDestination(),
		"startDate":   start.Format("2006-01-02"),
		"endDate":     end.Format("2006-01-02"),
		"itinerary":   req.GetItinerary(),
	})
	s.ensureWorkflowDefinition(ctx, tid, uid)
	resp, err := s.workflowService.SubmitApply(ctx, &oaV1.SubmitApplyRequest{
		Code:         businessTripDefinitionCode,
		Version:      businessTripDefinitionVersion,
		FormData:     string(form),
		BusinessType: trans.Ptr(businessTypeBusinessTrip),
		BusinessId:   trans.Ptr(appID),
	})
	if err != nil {
		_ = s.appRepo.UpdateStatus(ctx, tid, appID, oaV1.BusinessTripApplication_REJECTED)
		return nil, err
	}
	if err := s.appRepo.SetInstanceID(ctx, tid, appID, resp.GetInstanceId()); err != nil {
		return nil, err
	}
	return &oaV1.SubmitBusinessTripApplicationResponse{Id: appID, InstanceId: resp.GetInstanceId()}, nil
}

// ensureWorkflowDefinition BUSINESS_TRIP 流程定义兜底：租户内不存在时按默认模板创建并启用
// （提交给申请人主管 LEADER，会签）。并发首提撞唯一索引时忽略，回读校验。
func (s *BusinessTripService) ensureWorkflowDefinition(ctx context.Context, tid, uid uint32) {
	if _, err := s.definitionRepo.GetByCodeVersion(ctx, businessTripDefinitionCode, businessTripDefinitionVersion); err == nil {
		return
	}
	nodeConfig := `[{"approvers":[{"type":"LEADER"}],"strategy":"ALL"}]`
	_, err := s.definitionRepo.Create(ctx, &oaV1.CreateWorkflowDefinitionRequest{
		Data: &oaV1.WorkflowDefinition{
			Code:             trans.Ptr(businessTripDefinitionCode),
			Version:          trans.Ptr(businessTripDefinitionVersion),
			NodeConfig:       trans.Ptr(nodeConfig),
			DefinitionStatus: oaV1.WorkflowDefinition_ENABLED.Enum(),
			TenantId:         trans.Ptr(tid),
			CreatedBy:        trans.Ptr(uid),
		},
	})
	if err != nil {
		s.log.Warnf("bootstrap BUSINESS_TRIP definition failed (may conflict): %s", err.Error())
	}
}

func (s *BusinessTripService) ListBusinessTripApplications(ctx context.Context, req *oaV1.ListBusinessTripApplicationsRequest) (*oaV1.ListBusinessTripApplicationsResponse, error) {
	tid, _, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	items, total, err := s.appRepo.List(ctx, tid, req.GetUserId(), req.GetStatus(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	s.fillApplicantNames(ctx, tid, items)
	return &oaV1.ListBusinessTripApplicationsResponse{Items: items, Total: uint64(total)}, nil
}

func (s *BusinessTripService) GetBusinessTripApplication(ctx context.Context, req *oaV1.GetBusinessTripApplicationRequest) (*oaV1.BusinessTripApplication, error) {
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
	s.fillApplicantNames(ctx, tid, []*oaV1.BusinessTripApplication{item})
	return item, nil
}

// onInstanceTerminal 工作流终结回调。校验单据与实例关联（防伪造 business 字段），
// 幂等（仅 PENDING 单据处理）：三态均仅同步单据状态，出差申请无额度副作用。
func (s *BusinessTripService) onInstanceTerminal(ctx context.Context, tenantID, instanceID, businessID uint32, status oaV1.WorkflowInstance_InstanceStatus) {
	entity, err := s.appRepo.GetEntity(ctx, tenantID, businessID)
	if err != nil || entity == nil {
		s.log.Errorf("business-trip hook load application %d failed: %v", businessID, err)
		return
	}
	if entity.InstanceID == nil || *entity.InstanceID != instanceID {
		return
	}
	if entity.TripStatus == nil || *entity.TripStatus != businesstripapplication.TripStatusPending {
		return
	}

	var target oaV1.BusinessTripApplication_BusinessTripStatus
	switch status {
	case oaV1.WorkflowInstance_APPROVED:
		target = oaV1.BusinessTripApplication_APPROVED
	case oaV1.WorkflowInstance_REJECTED:
		target = oaV1.BusinessTripApplication_REJECTED
	case oaV1.WorkflowInstance_WITHDRAWN:
		target = oaV1.BusinessTripApplication_WITHDRAWN
	default:
		return
	}
	if err := s.appRepo.UpdateStatus(ctx, tenantID, businessID, target); err != nil {
		s.log.Errorf("business-trip hook sync application %d failed: %s", businessID, err.Error())
	}
}
