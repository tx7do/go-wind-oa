package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-oa/app/core/service/internal/data/ent"
	"go-wind-oa/app/core/service/internal/data/ent/leaveapplication"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

type LeaveApplicationRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewLeaveApplicationRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *LeaveApplicationRepo {
	return &LeaveApplicationRepo{
		log:       ctx.NewLoggerHelper("leave-application/repo/core-service"),
		entClient: entClient,
	}
}

func leaveStatusToProto(s leaveapplication.LeaveStatus) oaV1.LeaveApplication_LeaveStatus {
	switch s {
	case leaveapplication.LeaveStatusApproved:
		return oaV1.LeaveApplication_APPROVED
	case leaveapplication.LeaveStatusRejected:
		return oaV1.LeaveApplication_REJECTED
	case leaveapplication.LeaveStatusWithdrawn:
		return oaV1.LeaveApplication_WITHDRAWN
	default:
		return oaV1.LeaveApplication_PENDING
	}
}

func leaveStatusToEntity(s oaV1.LeaveApplication_LeaveStatus) leaveapplication.LeaveStatus {
	switch s {
	case oaV1.LeaveApplication_APPROVED:
		return leaveapplication.LeaveStatusApproved
	case oaV1.LeaveApplication_REJECTED:
		return leaveapplication.LeaveStatusRejected
	case oaV1.LeaveApplication_WITHDRAWN:
		return leaveapplication.LeaveStatusWithdrawn
	default:
		return leaveapplication.LeaveStatusPending
	}
}

func leaveApplicationToDTO(e *ent.LeaveApplication, typeName string) *oaV1.LeaveApplication {
	if e == nil {
		return nil
	}
	status := oaV1.LeaveApplication_PENDING
	if e.LeaveStatus != nil {
		status = leaveStatusToProto(*e.LeaveStatus)
	}
	startHalf := oaV1.HalfOfDay_AM
	if e.StartHalf == 1 {
		startHalf = oaV1.HalfOfDay_PM
	}
	endHalf := oaV1.HalfOfDay_PM
	if e.EndHalf == 0 {
		endHalf = oaV1.HalfOfDay_AM
	}
	dto := &oaV1.LeaveApplication{
		Id:            trans.Ptr(e.ID),
		LeaveTypeId:   trans.Ptr(e.LeaveTypeID),
		LeaveTypeName: trans.Ptr(typeName),
		StartDate:     timeutil.TimeToTimestamppb(&e.StartDate),
		EndDate:       timeutil.TimeToTimestamppb(&e.EndDate),
		Days:          trans.Ptr(e.Days),
		Reason:        trans.Ptr(e.Reason),
		LeaveStatus:   status.Enum(),
		StartHalf:     startHalf.Enum(),
		EndHalf:       endHalf.Enum(),
		TenantId:      e.TenantID,
	}
	if e.InstanceID != nil {
		dto.InstanceId = trans.Ptr(*e.InstanceID)
	}
	if e.CreatedBy != nil {
		dto.CreatedBy = e.CreatedBy
	}
	dto.CreatedAt = timeutil.TimeToTimestamppb(e.CreatedAt)
	return dto
}

func (r *LeaveApplicationRepo) Create(
	ctx context.Context,
	tid, uid uint32,
	typeID uint32,
	startDate, endDate time.Time,
	days float64,
	reason string,
	startHalf, endHalf uint8,
) (uint32, error) {
	entity, err := r.entClient.Client().LeaveApplication.Create().
		SetLeaveTypeID(typeID).
		SetStartDate(startDate).
		SetEndDate(endDate).
		SetDays(days).
		SetReason(reason).
		SetStartHalf(startHalf).
		SetEndHalf(endHalf).
		SetLeaveStatus(leaveapplication.LeaveStatusPending).
		SetTenantID(tid).
		SetCreatedBy(uid).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		r.log.Errorf("insert leave application failed: %s", err.Error())
		return 0, oaV1.ErrorInternalServerError("insert leave application failed")
	}
	return entity.ID, nil
}

// Get 按主键读取（含类型名称回填）。
func (r *LeaveApplicationRepo) Get(ctx context.Context, tid, id uint32, leaveTypeRepo *LeaveTypeRepo) (*oaV1.LeaveApplication, error) {
	entity, err := r.entClient.Client().LeaveApplication.Query().
		Where(
			leaveapplication.IDEQ(id),
			leaveapplication.TenantIDEQ(tid),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, oaV1.ErrorNotFound("leave application not found")
		}
		r.log.Errorf("query leave application failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query leave application failed")
	}
	typeName := ""
	if lt, err := leaveTypeRepo.GetByID(ctx, tid, entity.LeaveTypeID); err == nil && lt != nil {
		typeName = lt.GetName()
	}
	return leaveApplicationToDTO(entity, typeName), nil
}

func (r *LeaveApplicationRepo) UpdateStatus(ctx context.Context, tid, id uint32, status oaV1.LeaveApplication_LeaveStatus) error {
	if _, err := r.entClient.Client().LeaveApplication.Update().
		Where(
			leaveapplication.IDEQ(id),
			leaveapplication.TenantIDEQ(tid),
		).
		SetLeaveStatus(leaveStatusToEntity(status)).
		SetUpdatedAt(time.Now()).
		Save(ctx); err != nil {
		r.log.Errorf("update leave application status failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("update leave application status failed")
	}
	return nil
}

func (r *LeaveApplicationRepo) SetInstanceID(ctx context.Context, tid, id, instanceID uint32) error {
	if _, err := r.entClient.Client().LeaveApplication.Update().
		Where(
			leaveapplication.IDEQ(id),
			leaveapplication.TenantIDEQ(tid),
		).
		SetInstanceID(instanceID).
		SetUpdatedAt(time.Now()).
		Save(ctx); err != nil {
		r.log.Errorf("link leave application instance failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("link leave application instance failed")
	}
	return nil
}

// List 查询申请单。userID==0 时查全部（admin）；status 非零值时按状态过滤。
func (r *LeaveApplicationRepo) List(
	ctx context.Context,
	tid, userID uint32,
	status oaV1.LeaveApplication_LeaveStatus,
	leaveTypeRepo *LeaveTypeRepo,
) ([]*oaV1.LeaveApplication, error) {
	query := r.entClient.Client().LeaveApplication.Query().
		Where(leaveapplication.TenantIDEQ(tid))
	if userID != 0 {
		query = query.Where(leaveapplication.CreatedByEQ(userID))
	}
	if status != 0 {
		query = query.Where(leaveapplication.LeaveStatusEQ(leaveStatusToEntity(status)))
	}
	entities, err := query.All(ctx)
	if err != nil {
		r.log.Errorf("list leave applications failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("list leave applications failed")
	}

	typeNames := make(map[uint32]string)
	items := make([]*oaV1.LeaveApplication, 0, len(entities))
	for _, e := range entities {
		name, ok := typeNames[e.LeaveTypeID]
		if !ok {
			name = ""
			if lt, err := leaveTypeRepo.GetByID(ctx, tid, e.LeaveTypeID); err == nil && lt != nil {
				name = lt.GetName()
			}
			typeNames[e.LeaveTypeID] = name
		}
		items = append(items, leaveApplicationToDTO(e, name))
	}
	return items, nil
}

// GetEntity 供工作流终结回调读取原始实体（校验关联与状态同步）。不存在返回 nil。
func (r *LeaveApplicationRepo) GetEntity(ctx context.Context, tid, id uint32) (*ent.LeaveApplication, error) {
	entity, err := r.entClient.Client().LeaveApplication.Query().
		Where(
			leaveapplication.IDEQ(id),
			leaveapplication.TenantIDEQ(tid),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf("query leave application entity failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query leave application failed")
	}
	return entity, nil
}

// HasApprovedLeaveCovering 判断用户在指定日期是否被已通过的请假单覆盖（考勤结算用）。
func (r *LeaveApplicationRepo) HasApprovedLeaveCovering(ctx context.Context, tid, userID uint32, date time.Time) (bool, error) {
	count, err := r.entClient.Client().LeaveApplication.Query().
		Where(
			leaveapplication.TenantIDEQ(tid),
			leaveapplication.CreatedByEQ(userID),
			leaveapplication.LeaveStatusEQ(leaveapplication.LeaveStatusApproved),
			leaveapplication.StartDateLTE(date),
			leaveapplication.EndDateGTE(date),
		).
		Count(ctx)
	if err != nil {
		r.log.Errorf("query leave coverage failed: %s", err.Error())
		return false, oaV1.ErrorInternalServerError("query leave coverage failed")
	}
	return count > 0, nil
}
