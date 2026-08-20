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
	"go-wind-oa/app/core/service/internal/data/ent/overtimeapplication"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

type OvertimeApplicationRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewOvertimeApplicationRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *OvertimeApplicationRepo {
	return &OvertimeApplicationRepo{
		log:       ctx.NewLoggerHelper("overtime-application/repo/core-service"),
		entClient: entClient,
	}
}

func overtimeStatusToProto(s overtimeapplication.OvertimeStatus) oaV1.OvertimeApplication_OvertimeStatus {
	switch s {
	case overtimeapplication.OvertimeStatusApproved:
		return oaV1.OvertimeApplication_APPROVED
	case overtimeapplication.OvertimeStatusRejected:
		return oaV1.OvertimeApplication_REJECTED
	case overtimeapplication.OvertimeStatusWithdrawn:
		return oaV1.OvertimeApplication_WITHDRAWN
	default:
		return oaV1.OvertimeApplication_PENDING
	}
}

func overtimeStatusToEntity(s oaV1.OvertimeApplication_OvertimeStatus) overtimeapplication.OvertimeStatus {
	switch s {
	case oaV1.OvertimeApplication_APPROVED:
		return overtimeapplication.OvertimeStatusApproved
	case oaV1.OvertimeApplication_REJECTED:
		return overtimeapplication.OvertimeStatusRejected
	case oaV1.OvertimeApplication_WITHDRAWN:
		return overtimeapplication.OvertimeStatusWithdrawn
	default:
		return overtimeapplication.OvertimeStatusPending
	}
}

func compensationTypeToProto(s overtimeapplication.CompensationType) oaV1.OvertimeApplication_CompensationType {
	switch s {
	case overtimeapplication.CompensationTypeOvertimePay:
		return oaV1.OvertimeApplication_OVERTIME_PAY
	default:
		return oaV1.OvertimeApplication_COMP_LEAVE
	}
}

func compensationTypeToEntity(s oaV1.OvertimeApplication_CompensationType) overtimeapplication.CompensationType {
	switch s {
	case oaV1.OvertimeApplication_OVERTIME_PAY:
		return overtimeapplication.CompensationTypeOvertimePay
	default:
		return overtimeapplication.CompensationTypeCompLeave
	}
}

func overtimeApplicationToDTO(e *ent.OvertimeApplication) *oaV1.OvertimeApplication {
	if e == nil {
		return nil
	}
	status := oaV1.OvertimeApplication_PENDING
	if e.OvertimeStatus != nil {
		status = overtimeStatusToProto(*e.OvertimeStatus)
	}
	comp := oaV1.OvertimeApplication_COMP_LEAVE
	if e.CompensationType != nil {
		comp = compensationTypeToProto(*e.CompensationType)
	}
	dto := &oaV1.OvertimeApplication{
		Id:                trans.Ptr(e.ID),
		Reason:            trans.Ptr(e.Reason),
		StartTime:         timeutil.TimeToTimestamppb(&e.StartTime),
		EndTime:           timeutil.TimeToTimestamppb(&e.EndTime),
		CompensationType:  comp.Enum(),
		OvertimeStatus:    status.Enum(),
		TenantId:          e.TenantID,
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

func (r *OvertimeApplicationRepo) Create(
	ctx context.Context,
	tid, uid uint32,
	reason string,
	startTime, endTime time.Time,
	compensationType oaV1.OvertimeApplication_CompensationType,
) (uint32, error) {
	entity, err := r.entClient.Client().OvertimeApplication.Create().
		SetReason(reason).
		SetStartTime(startTime).
		SetEndTime(endTime).
		SetCompensationType(compensationTypeToEntity(compensationType)).
		SetOvertimeStatus(overtimeapplication.OvertimeStatusPending).
		SetTenantID(tid).
		SetCreatedBy(uid).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		r.log.Errorf("insert overtime application failed: %s", err.Error())
		return 0, oaV1.ErrorInternalServerError("insert overtime application failed")
	}
	return entity.ID, nil
}

func (r *OvertimeApplicationRepo) Get(ctx context.Context, tid, id uint32) (*oaV1.OvertimeApplication, error) {
	entity, err := r.entClient.Client().OvertimeApplication.Query().
		Where(
			overtimeapplication.IDEQ(id),
			overtimeapplication.TenantIDEQ(tid),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, oaV1.ErrorNotFound("overtime application not found")
		}
		r.log.Errorf("query overtime application failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query overtime application failed")
	}
	return overtimeApplicationToDTO(entity), nil
}

func (r *OvertimeApplicationRepo) UpdateStatus(ctx context.Context, tid, id uint32, status oaV1.OvertimeApplication_OvertimeStatus) error {
	if _, err := r.entClient.Client().OvertimeApplication.Update().
		Where(
			overtimeapplication.IDEQ(id),
			overtimeapplication.TenantIDEQ(tid),
		).
		SetOvertimeStatus(overtimeStatusToEntity(status)).
		SetUpdatedAt(time.Now()).
		Save(ctx); err != nil {
		r.log.Errorf("update overtime application status failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("update overtime application status failed")
	}
	return nil
}

func (r *OvertimeApplicationRepo) SetInstanceID(ctx context.Context, tid, id, instanceID uint32) error {
	if _, err := r.entClient.Client().OvertimeApplication.Update().
		Where(
			overtimeapplication.IDEQ(id),
			overtimeapplication.TenantIDEQ(tid),
		).
		SetInstanceID(instanceID).
		SetUpdatedAt(time.Now()).
		Save(ctx); err != nil {
		r.log.Errorf("link overtime application instance failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("link overtime application instance failed")
	}
	return nil
}

func (r *OvertimeApplicationRepo) List(
	ctx context.Context,
	tid, userID uint32,
	status oaV1.OvertimeApplication_OvertimeStatus,
	page, pageSize int32,
) ([]*oaV1.OvertimeApplication, int, error) {
	query := r.entClient.Client().OvertimeApplication.Query().
		Where(overtimeapplication.TenantIDEQ(tid))
	if userID != 0 {
		query = query.Where(overtimeapplication.CreatedByEQ(userID))
	}
	if status != 0 {
		query = query.Where(overtimeapplication.OvertimeStatusEQ(overtimeStatusToEntity(status)))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		r.log.Errorf("count overtime applications failed: %s", err.Error())
		return nil, 0, oaV1.ErrorInternalServerError("count overtime applications failed")
	}
	if pageSize > 0 {
		if page < 1 {
			page = 1
		}
		query = query.Offset((int(page) - 1) * int(pageSize)).Limit(int(pageSize))
	}
	entities, err := query.Order(ent.Desc(overtimeapplication.FieldID)).All(ctx)
	if err != nil {
		r.log.Errorf("list overtime applications failed: %s", err.Error())
		return nil, 0, oaV1.ErrorInternalServerError("list overtime applications failed")
	}

	items := make([]*oaV1.OvertimeApplication, 0, len(entities))
	for _, e := range entities {
		items = append(items, overtimeApplicationToDTO(e))
	}
	return items, total, nil
}

func (r *OvertimeApplicationRepo) GetEntity(ctx context.Context, tid, id uint32) (*ent.OvertimeApplication, error) {
	entity, err := r.entClient.Client().OvertimeApplication.Query().
		Where(
			overtimeapplication.IDEQ(id),
			overtimeapplication.TenantIDEQ(tid),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf("query overtime application entity failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query overtime application failed")
	}
	return entity, nil
}
