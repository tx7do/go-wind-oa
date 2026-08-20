package service

import (
	"context"
	"encoding/json"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-oa/app/core/service/internal/data"
	"go-wind-oa/app/core/service/internal/data/ent/sealapplication"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

const (
	businessTypeSealApplication     = "SEAL_APPLICATION"
	sealDefinitionCode              = "SEAL_APPLICATION"
	sealDefinitionVersion           = int32(1)
)

type SealApplicationService struct {
	oaV1.UnimplementedSealApplicationServiceServer

	log *log.Helper

	appRepo        *data.SealApplicationRepo
	definitionRepo *data.WorkflowDefinitionRepo
	resolverRepo   *data.WorkflowResolverRepo

	workflowService *WorkflowService
}

func NewSealApplicationService(
	ctx *bootstrap.Context,
	appRepo *data.SealApplicationRepo,
	definitionRepo *data.WorkflowDefinitionRepo,
	resolverRepo *data.WorkflowResolverRepo,
	workflowService *WorkflowService,
	eventRegistry *WorkflowEventRegistry,
) *SealApplicationService {
	svc := &SealApplicationService{
		log:             ctx.NewLoggerHelper("seal-application/service/core-service"),
		appRepo:         appRepo,
		definitionRepo:  definitionRepo,
		resolverRepo:    resolverRepo,
		workflowService: workflowService,
	}
	eventRegistry.Register(businessTypeSealApplication, svc.onInstanceTerminal)
	return svc
}

func (s *SealApplicationService) fillApplicantNames(ctx context.Context, tid uint32, items []*oaV1.SealApplication) {
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

func (s *SealApplicationService) SubmitSealApplication(ctx context.Context, req *oaV1.SubmitSealApplicationRequest) (*oaV1.SubmitSealApplicationResponse, error) {
	tid, uid, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	if req.GetPurpose() == "" || req.GetRecipient() == "" {
		return nil, oaV1.ErrorBadRequest("invalid parameter")
	}

	appID, err := s.appRepo.Create(ctx, tid, uid, req.GetPurpose(), req.GetSealType(), req.GetFileCount(), req.GetRecipient())
	if err != nil {
		return nil, err
	}

	form, _ := json.Marshal(map[string]any{
		"business":  "用印申请",
		"purpose":   req.GetPurpose(),
		"sealType":  req.GetSealType().String(),
		"fileCount": req.GetFileCount(),
		"recipient": req.GetRecipient(),
	})
	s.ensureWorkflowDefinition(ctx, tid, uid)
	resp, err := s.workflowService.SubmitApply(ctx, &oaV1.SubmitApplyRequest{
		Code:         sealDefinitionCode,
		Version:      sealDefinitionVersion,
		FormData:     string(form),
		BusinessType: trans.Ptr(businessTypeSealApplication),
		BusinessId:   trans.Ptr(appID),
	})
	if err != nil {
		_ = s.appRepo.UpdateStatus(ctx, tid, appID, oaV1.SealApplication_REJECTED)
		return nil, err
	}
	if err := s.appRepo.SetInstanceID(ctx, tid, appID, resp.GetInstanceId()); err != nil {
		return nil, err
	}
	return &oaV1.SubmitSealApplicationResponse{Id: appID, InstanceId: resp.GetInstanceId()}, nil
}

func (s *SealApplicationService) ensureWorkflowDefinition(ctx context.Context, tid, uid uint32) {
	if _, err := s.definitionRepo.GetByCodeVersion(ctx, sealDefinitionCode, sealDefinitionVersion); err == nil {
		return
	}
	nodeConfig := `[{"approvers":[{"type":"LEADER"}],"strategy":"ALL"}]`
	_, err := s.definitionRepo.Create(ctx, &oaV1.CreateWorkflowDefinitionRequest{
		Data: &oaV1.WorkflowDefinition{
			Code:             trans.Ptr(sealDefinitionCode),
			Version:          trans.Ptr(sealDefinitionVersion),
			NodeConfig:       trans.Ptr(nodeConfig),
			DefinitionStatus: oaV1.WorkflowDefinition_ENABLED.Enum(),
			TenantId:         trans.Ptr(tid),
			CreatedBy:        trans.Ptr(uid),
		},
	})
	if err != nil {
		s.log.Warnf("bootstrap SEAL_APPLICATION definition failed (may conflict): %s", err.Error())
	}
}

func (s *SealApplicationService) ListSealApplications(ctx context.Context, req *oaV1.ListSealApplicationsRequest) (*oaV1.ListSealApplicationsResponse, error) {
	tid, _, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	items, total, err := s.appRepo.List(ctx, tid, req.GetUserId(), req.GetStatus(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	s.fillApplicantNames(ctx, tid, items)
	return &oaV1.ListSealApplicationsResponse{Items: items, Total: uint64(total)}, nil
}

func (s *SealApplicationService) GetSealApplication(ctx context.Context, req *oaV1.GetSealApplicationRequest) (*oaV1.SealApplication, error) {
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
	s.fillApplicantNames(ctx, tid, []*oaV1.SealApplication{item})
	return item, nil
}

func (s *SealApplicationService) onInstanceTerminal(ctx context.Context, tenantID, instanceID, businessID uint32, status oaV1.WorkflowInstance_InstanceStatus) {
	entity, err := s.appRepo.GetEntity(ctx, tenantID, businessID)
	if err != nil || entity == nil {
		s.log.Errorf("seal hook load application %d failed: %v", businessID, err)
		return
	}
	if entity.InstanceID == nil || *entity.InstanceID != instanceID {
		return
	}
	if entity.SealStatus == nil || *entity.SealStatus != sealapplication.SealStatusPending {
		return
	}

	var target oaV1.SealApplication_SealStatus
	switch status {
	case oaV1.WorkflowInstance_APPROVED:
		target = oaV1.SealApplication_APPROVED
	case oaV1.WorkflowInstance_REJECTED:
		target = oaV1.SealApplication_REJECTED
	case oaV1.WorkflowInstance_WITHDRAWN:
		target = oaV1.SealApplication_WITHDRAWN
	default:
		return
	}
	if err := s.appRepo.UpdateStatus(ctx, tenantID, businessID, target); err != nil {
		s.log.Errorf("seal hook sync application %d failed: %s", businessID, err.Error())
	}
}
