package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-oa/app/core/service/internal/data"
	"go-wind-oa/app/core/service/internal/data/ent/leaveapplication"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// 业务单据类型与内置流程定义代码。
const (
	businessTypeLeave = "LEAVE"

	leaveDefinitionCode    = "LEAVE"
	leaveDefinitionVersion = int32(1)
)

type LeaveService struct {
	oaV1.UnimplementedLeaveServiceServer

	log *log.Helper

	typeRepo       *data.LeaveTypeRepo
	balanceRepo    *data.LeaveBalanceRepo
	appRepo        *data.LeaveApplicationRepo
	definitionRepo *data.WorkflowDefinitionRepo
	resolverRepo   *data.WorkflowResolverRepo

	workflowService *WorkflowService
}

func NewLeaveService(
	ctx *bootstrap.Context,
	typeRepo *data.LeaveTypeRepo,
	balanceRepo *data.LeaveBalanceRepo,
	appRepo *data.LeaveApplicationRepo,
	definitionRepo *data.WorkflowDefinitionRepo,
	resolverRepo *data.WorkflowResolverRepo,
	workflowService *WorkflowService,
	eventRegistry *WorkflowEventRegistry,
) *LeaveService {
	svc := &LeaveService{
		log:             ctx.NewLoggerHelper("leave/service/core-service"),
		typeRepo:        typeRepo,
		balanceRepo:     balanceRepo,
		appRepo:         appRepo,
		definitionRepo:  definitionRepo,
		resolverRepo:    resolverRepo,
		workflowService: workflowService,
	}
	// 注册工作流终结回调：审批通过扣减额度，驳回/撤回同步状态。
	eventRegistry.Register(businessTypeLeave, svc.onInstanceTerminal)
	return svc
}

// fillApplicantNames 批量回填申请人姓名（失败仅缺姓名，不影响列表）。
func (s *LeaveService) fillApplicantNames(ctx context.Context, tid uint32, items []*oaV1.LeaveApplication) {
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

func (s *LeaveService) CreateLeaveType(ctx context.Context, req *oaV1.CreateLeaveTypeRequest) (*oaV1.LeaveType, error) {
	tid, uid, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	if req.GetData() == nil || req.GetData().GetCode() == "" || req.GetData().GetName() == "" {
		return nil, oaV1.ErrorBadRequest("invalid parameter")
	}
	return s.typeRepo.Create(ctx, tid, uid, req.GetData().GetCode(), req.GetData().GetName())
}

func (s *LeaveService) ListLeaveTypes(ctx context.Context, req *paginationV1.PagingRequest) (*oaV1.ListLeaveTypesResponse, error) {
	tid, _, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	return s.typeRepo.List(ctx, tid, req)
}

func (s *LeaveService) GrantLeaveBalance(ctx context.Context, req *oaV1.GrantLeaveBalanceRequest) (*emptypb.Empty, error) {
	tid, uid, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	if req.GetUserId() == 0 || req.GetLeaveTypeId() == 0 || req.GetYear() == 0 || req.GetTotalDays() < 0 {
		return nil, oaV1.ErrorBadRequest("invalid parameter")
	}
	if err := s.balanceRepo.Grant(ctx, tid, uid, req.GetUserId(), req.GetLeaveTypeId(), int(req.GetYear()), req.GetTotalDays()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *LeaveService) ListLeaveBalances(ctx context.Context, req *oaV1.ListLeaveBalancesRequest) (*oaV1.ListLeaveBalancesResponse, error) {
	tid, _, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	year := int(req.GetYear())
	if year == 0 {
		year = time.Now().Year()
	}
	items, err := s.balanceRepo.List(ctx, tid, req.GetUserId(), year)
	if err != nil {
		return nil, err
	}
	return &oaV1.ListLeaveBalancesResponse{Items: items, Total: uint64(len(items))}, nil
}

// SubmitLeaveApplication 提交请假申请：校验额度 → 建申请单 → 进程内提交 LEAVE 工作流。
// LEAVE 流程定义不存在时按默认模板（提交给申请人主管，会签）自动创建并启用。
func (s *LeaveService) SubmitLeaveApplication(ctx context.Context, req *oaV1.SubmitLeaveApplicationRequest) (*oaV1.SubmitLeaveApplicationResponse, error) {
	tid, uid, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	if req.GetLeaveTypeId() == 0 || req.GetStartDate() == nil || req.GetEndDate() == nil {
		return nil, oaV1.ErrorBadRequest("invalid parameter")
	}

	lt, err := s.typeRepo.GetByID(ctx, tid, req.GetLeaveTypeId())
	if err != nil {
		return nil, err
	}

	// 日期统一截断到服务器本地零点：客户端可能带 UTC 午夜或带时分，考勤结算
	//（HasApprovedLeaveCovering）按本地午夜比较，不截断会出现跨时区错位。
	// Timestamp.AsTime() 固定返回 UTC 位置（与 JSON 时区偏移无关），先转到
	// 服务器本地时区再截断，保证与考勤结算（time.Now() 本地）同一日期轴。
	start := truncateDate(req.GetStartDate().AsTime().In(time.Local))
	end := truncateDate(req.GetEndDate().AsTime().In(time.Local))
	if end.Before(start) {
		return nil, oaV1.ErrorBadRequest("end date before start date")
	}
	// 半日缺省：开始 AM、结束 PM（proto3 枚举零值无法区分未传，故用 optional）。
	startHalf := req.GetStartHalf()
	if req.StartHalf == nil {
		startHalf = oaV1.HalfOfDay_AM
	}
	endHalf := req.GetEndHalf()
	if req.EndHalf == nil {
		endHalf = oaV1.HalfOfDay_PM
	}
	days := computeLeaveDays(start, end, startHalf, endHalf)
	if days <= 0 {
		return nil, oaV1.ErrorBadRequest("invalid half-day range")
	}

	balance, err := s.balanceRepo.Get(ctx, tid, uid, req.GetLeaveTypeId(), start.Year())
	if err != nil {
		return nil, err
	}
	if balance == nil || balance.GetTotalDays()-balance.GetUsedDays() < days {
		return nil, oaV1.ErrorBadRequest("insufficient leave balance")
	}

	startHalfVal, endHalfVal := halfDayValues(startHalf, endHalf)
	appID, err := s.appRepo.Create(ctx, tid, uid, req.GetLeaveTypeId(), start, end, days, req.GetReason(), startHalfVal, endHalfVal)
	if err != nil {
		return nil, err
	}

	form, _ := json.Marshal(map[string]any{
		"business":      "请假申请",
		"leaveTypeName": lt.GetName(),
		"startDate":     start.Format("2006-01-02"),
		"endDate":       end.Format("2006-01-02"),
		"days":          days,
		"reason":        req.GetReason(),
	})
	s.ensureWorkflowDefinition(ctx, tid, uid)
	resp, err := s.workflowService.SubmitApply(ctx, &oaV1.SubmitApplyRequest{
		Code:         leaveDefinitionCode,
		Version:      leaveDefinitionVersion,
		FormData:     string(form),
		BusinessType: trans.Ptr(businessTypeLeave),
		BusinessId:   trans.Ptr(appID),
	})
	if err != nil {
		_ = s.appRepo.UpdateStatus(ctx, tid, appID, oaV1.LeaveApplication_REJECTED)
		return nil, err
	}
	if err := s.appRepo.SetInstanceID(ctx, tid, appID, resp.GetInstanceId()); err != nil {
		return nil, err
	}
	return &oaV1.SubmitLeaveApplicationResponse{Id: appID, InstanceId: resp.GetInstanceId()}, nil
}

// computeLeaveDays 半日粒度天数：start_half=PM 表示首日只请下午，end_half=AM 表示
// 末日只请上午。同日 start=PM 且 end=AM 视为非法（返回 0）。
func computeLeaveDays(start, end time.Time, startHalf, endHalf oaV1.HalfOfDay) float64 {
	diff := int(end.Sub(start).Hours() / 24)
	if diff < 0 {
		return 0
	}
	if diff == 0 {
		switch {
		case startHalf == oaV1.HalfOfDay_PM && endHalf == oaV1.HalfOfDay_AM:
			return 0
		case startHalf == oaV1.HalfOfDay_PM || endHalf == oaV1.HalfOfDay_AM:
			return 0.5
		default:
			return 1
		}
	}
	startHalves := 2
	if startHalf == oaV1.HalfOfDay_PM {
		startHalves = 1
	}
	endHalves := 2
	if endHalf == oaV1.HalfOfDay_AM {
		endHalves = 1
	}
	return float64((diff-1)*2+startHalves+endHalves) / 2
}

func halfDayValues(startHalf, endHalf oaV1.HalfOfDay) (uint8, uint8) {
	sh := uint8(0)
	if startHalf == oaV1.HalfOfDay_PM {
		sh = 1
	}
	eh := uint8(1)
	if endHalf == oaV1.HalfOfDay_AM {
		eh = 0
	}
	return sh, eh
}

// ensureWorkflowDefinition LEAVE 流程定义兜底：租户内不存在时按默认模板创建并启用
// （提交给申请人主管 LEADER，会签）。并发首提撞唯一索引时忽略，回读校验。
func (s *LeaveService) ensureWorkflowDefinition(ctx context.Context, tid, uid uint32) {
	if _, err := s.definitionRepo.GetByCodeVersion(ctx, leaveDefinitionCode, leaveDefinitionVersion); err == nil {
		return
	}
	nodeConfig := `[{"approvers":[{"type":"LEADER"}],"strategy":"ALL"}]`
	_, err := s.definitionRepo.Create(ctx, &oaV1.CreateWorkflowDefinitionRequest{
		Data: &oaV1.WorkflowDefinition{
			Code:             trans.Ptr(leaveDefinitionCode),
			Version:          trans.Ptr(leaveDefinitionVersion),
			NodeConfig:       trans.Ptr(nodeConfig),
			DefinitionStatus: oaV1.WorkflowDefinition_ENABLED.Enum(),
			TenantId:         trans.Ptr(tid),
			CreatedBy:        trans.Ptr(uid),
		},
	})
	if err != nil {
		// 并发创建冲突或瞬时错误：后续 SubmitApply 的 GetByCodeVersion 会给出准确结果。
		s.log.Warnf("bootstrap LEAVE definition failed (may conflict): %s", err.Error())
	}
}

func (s *LeaveService) ListLeaveApplications(ctx context.Context, req *oaV1.ListLeaveApplicationsRequest) (*oaV1.ListLeaveApplicationsResponse, error) {
	tid, _, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	items, err := s.appRepo.List(ctx, tid, req.GetUserId(), req.GetStatus(), s.typeRepo)
	if err != nil {
		return nil, err
	}
	s.fillApplicantNames(ctx, tid, items)
	return &oaV1.ListLeaveApplicationsResponse{Items: items, Total: uint64(len(items))}, nil
}

func (s *LeaveService) GetLeaveApplication(ctx context.Context, req *oaV1.GetLeaveApplicationRequest) (*oaV1.LeaveApplication, error) {
	tid, _, ok := callerFromContext(ctx)
	if !ok {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}
	if req.GetId() == 0 {
		return nil, oaV1.ErrorBadRequest("invalid parameter")
	}
	item, err := s.appRepo.Get(ctx, tid, req.GetId(), s.typeRepo)
	if err != nil {
		return nil, err
	}
	s.fillApplicantNames(ctx, tid, []*oaV1.LeaveApplication{item})
	return item, nil
}

// onInstanceTerminal 工作流终结回调。校验单据与实例关联（防伪造 business 字段），
// 幂等（仅 PENDING 单据处理）：APPROVED 扣额度，REJECTED/WITHDRAWN 仅同步状态。
func (s *LeaveService) onInstanceTerminal(ctx context.Context, tenantID, instanceID, businessID uint32, status oaV1.WorkflowInstance_InstanceStatus) {
	entity, err := s.appRepo.GetEntity(ctx, tenantID, businessID)
	if err != nil || entity == nil {
		s.log.Errorf("leave hook load application %d failed: %v", businessID, err)
		return
	}
	if entity.InstanceID == nil || *entity.InstanceID != instanceID {
		return
	}
	if entity.LeaveStatus == nil || *entity.LeaveStatus != leaveapplication.LeaveStatusPending {
		return
	}

	switch status {
	case oaV1.WorkflowInstance_APPROVED:
		if err := s.appRepo.UpdateStatus(ctx, tenantID, businessID, oaV1.LeaveApplication_APPROVED); err != nil {
			s.log.Errorf("leave hook approve application %d failed: %s", businessID, err.Error())
			return
		}
		applicant := uint32(0)
		if entity.CreatedBy != nil {
			applicant = *entity.CreatedBy
		}
		if err := s.balanceRepo.AddUsedDays(ctx, tenantID, applicant, entity.LeaveTypeID, entity.StartDate.Year(), entity.Days); err != nil {
			s.log.Errorf("leave hook deduct balance application %d failed: %s", businessID, err.Error())
		}
	case oaV1.WorkflowInstance_REJECTED:
		if err := s.appRepo.UpdateStatus(ctx, tenantID, businessID, oaV1.LeaveApplication_REJECTED); err != nil {
			s.log.Errorf("leave hook reject application %d failed: %s", businessID, err.Error())
		}
	case oaV1.WorkflowInstance_WITHDRAWN:
		if err := s.appRepo.UpdateStatus(ctx, tenantID, businessID, oaV1.LeaveApplication_WITHDRAWN); err != nil {
			s.log.Errorf("leave hook withdraw application %d failed: %s", businessID, err.Error())
		}
	}
}
