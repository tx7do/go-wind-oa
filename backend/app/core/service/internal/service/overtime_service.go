package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-oa/app/core/service/internal/data"
	"go-wind-oa/app/core/service/internal/data/ent/overtimeapplication"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

const (
	businessTypeOvertime       = "OVERTIME"
	overtimeDefinitionCode     = "OVERTIME"
	overtimeDefinitionVersion  = int32(1)
)

type OvertimeService struct {
	oaV1.UnimplementedOvertimeServiceServer

	log *log.Helper

	appRepo        *data.OvertimeApplicationRepo
	definitionRepo *data.WorkflowDefinitionRepo
	resolverRepo   *data.WorkflowResolverRepo

	workflowService *WorkflowService
}

func NewOvertimeService(
	ctx *bootstrap.Context,
	appRepo *data.OvertimeApplicationRepo,
	definitionRepo *data.WorkflowDefinitionRepo,
	resolverRepo *data.WorkflowResolverRepo,
	workflowService *WorkflowService,
	eventRegistry *WorkflowEventRegistry,
) *OvertimeService {
	svc := &OvertimeService{
		log:             ctx.NewLoggerHelper("overtime/service/core-service"),
		appRepo:         appRepo,
		definitionRepo:  definitionRepo,
		resolverRepo:    resolverRepo,
		workflowService: workflowService,
	}
	eventRegistry.Register(businessTypeOvertime, svc.onInstanceTerminal)
	return svc
}

func (s *OvertimeService) fillApplicantNames(ctx context.Context, tid uint32, items []*oaV1.OvertimeApplication) {
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

func (s *OvertimeService) SubmitOvertimeApplication(ctx context.Context, req *oaV1.SubmitOvertimeApplicationRequest) (*oaV1.SubmitOvertimeApplicationResponse, error) {
	tid, uid, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	if req.GetReason() == "" || req.GetStartTime() == nil || req.GetEndTime() == nil {
		return nil, oaV1.ErrorBadRequest("invalid parameter")
	}

	start := req.GetStartTime().AsTime().In(time.Local)
	end := req.GetEndTime().AsTime().In(time.Local)
	if end.Before(start) {
		return nil, oaV1.ErrorBadRequest("end time before start time")
	}

	appID, err := s.appRepo.Create(ctx, tid, uid, req.GetReason(), start, end, req.GetCompensationType())
	if err != nil {
		return nil, err
	}

	form, _ := json.Marshal(map[string]any{
		"business":         "加班申请",
		"reason":           req.GetReason(),
		"startTime":        start.Format("2006-01-02 15:04"),
		"endTime":          end.Format("2006-01-02 15:04"),
		"compensationType": req.GetCompensationType().String(),
	})
	s.ensureWorkflowDefinition(ctx, tid, uid)
	resp, err := s.workflowService.SubmitApply(ctx, &oaV1.SubmitApplyRequest{
		Code:         overtimeDefinitionCode,
		Version:      overtimeDefinitionVersion,
		FormData:     string(form),
		BusinessType: trans.Ptr(businessTypeOvertime),
		BusinessId:   trans.Ptr(appID),
	})
	if err != nil {
		_ = s.appRepo.UpdateStatus(ctx, tid, appID, oaV1.OvertimeApplication_REJECTED)
		return nil, err
	}
	if err := s.appRepo.SetInstanceID(ctx, tid, appID, resp.GetInstanceId()); err != nil {
		return nil, err
	}
	return &oaV1.SubmitOvertimeApplicationResponse{Id: appID, InstanceId: resp.GetInstanceId()}, nil
}

func (s *OvertimeService) ensureWorkflowDefinition(ctx context.Context, tid, uid uint32) {
	if _, err := s.definitionRepo.GetByCodeVersion(ctx, overtimeDefinitionCode, overtimeDefinitionVersion); err == nil {
		return
	}
	nodeConfig := `[{"approvers":[{"type":"LEADER"}],"strategy":"ALL"}]`
	_, err := s.definitionRepo.Create(ctx, &oaV1.CreateWorkflowDefinitionRequest{
		Data: &oaV1.WorkflowDefinition{
			Code:             trans.Ptr(overtimeDefinitionCode),
			Version:          trans.Ptr(overtimeDefinitionVersion),
			NodeConfig:       trans.Ptr(nodeConfig),
			DefinitionStatus: oaV1.WorkflowDefinition_ENABLED.Enum(),
			TenantId:         trans.Ptr(tid),
			CreatedBy:        trans.Ptr(uid),
		},
	})
	if err != nil {
		s.log.Warnf("bootstrap OVERTIME definition failed (may conflict): %s", err.Error())
	}
}

func (s *OvertimeService) ListOvertimeApplications(ctx context.Context, req *oaV1.ListOvertimeApplicationsRequest) (*oaV1.ListOvertimeApplicationsResponse, error) {
	tid, _, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	items, total, err := s.appRepo.List(ctx, tid, req.GetUserId(), req.GetStatus(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	s.fillApplicantNames(ctx, tid, items)
	return &oaV1.ListOvertimeApplicationsResponse{Items: items, Total: uint64(total)}, nil
}

func (s *OvertimeService) GetOvertimeApplication(ctx context.Context, req *oaV1.GetOvertimeApplicationRequest) (*oaV1.OvertimeApplication, error) {
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
	s.fillApplicantNames(ctx, tid, []*oaV1.OvertimeApplication{item})
	return item, nil
}

func (s *OvertimeService) onInstanceTerminal(ctx context.Context, tenantID, instanceID, businessID uint32, status oaV1.WorkflowInstance_InstanceStatus) {
	entity, err := s.appRepo.GetEntity(ctx, tenantID, businessID)
	if err != nil || entity == nil {
		s.log.Errorf("overtime hook load application %d failed: %v", businessID, err)
		return
	}
	if entity.InstanceID == nil || *entity.InstanceID != instanceID {
		return
	}
	if entity.OvertimeStatus == nil || *entity.OvertimeStatus != overtimeapplication.OvertimeStatusPending {
		return
	}

	var target oaV1.OvertimeApplication_OvertimeStatus
	switch status {
	case oaV1.WorkflowInstance_APPROVED:
		target = oaV1.OvertimeApplication_APPROVED
	case oaV1.WorkflowInstance_REJECTED:
		target = oaV1.OvertimeApplication_REJECTED
	case oaV1.WorkflowInstance_WITHDRAWN:
		target = oaV1.OvertimeApplication_WITHDRAWN
	default:
		return
	}
	if err := s.appRepo.UpdateStatus(ctx, tenantID, businessID, target); err != nil {
		s.log.Errorf("overtime hook sync application %d failed: %s", businessID, err.Error())
	}
}
