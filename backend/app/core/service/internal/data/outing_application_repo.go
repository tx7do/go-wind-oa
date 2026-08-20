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
	"go-wind-oa/app/core/service/internal/data/ent/outingapplication"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

type OutingApplicationRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewOutingApplicationRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *OutingApplicationRepo {
	return &OutingApplicationRepo{
		log:       ctx.NewLoggerHelper("outing-application/repo/core-service"),
		entClient: entClient,
	}
}

func outingStatusToProto(s outingapplication.OutingStatus) oaV1.OutingApplication_OutingStatus {
	switch s {
	case outingapplication.OutingStatusApproved:
		return oaV1.OutingApplication_APPROVED
	case outingapplication.OutingStatusRejected:
		return oaV1.OutingApplication_REJECTED
	case outingapplication.OutingStatusWithdrawn:
		return oaV1.OutingApplication_WITHDRAWN
	default:
		return oaV1.OutingApplication_PENDING
	}
}

func outingStatusToEntity(s oaV1.OutingApplication_OutingStatus) outingapplication.OutingStatus {
	switch s {
	case oaV1.OutingApplication_APPROVED:
		return outingapplication.OutingStatusApproved
	case oaV1.OutingApplication_REJECTED:
		return outingapplication.OutingStatusRejected
		case oaV1.OutingApplication_WITHDRAWN:
			return outingapplication.OutingStatusWithdrawn
	default:
		return outingapplication.OutingStatusPending
	}
}

func outingApplicationToDTO(e *ent.OutingApplication) *oaV1.OutingApplication {
	if e == nil {
		return nil
	}
	status := oaV1.OutingApplication_PENDING
	if e.OutingStatus != nil {
		status = outingStatusToProto(*e.OutingStatus)
	}
	dto := &oaV1.OutingApplication{
		Id:          trans.Ptr(e.ID),
		Reason:      trans.Ptr(e.Reason),
		Destination: trans.Ptr(e.Destination),
		StartTime:   timeutil.TimeToTimestamppb(&e.StartTime),
		EndTime:     timeutil.TimeToTimestamppb(&e.EndTime),
		OutingStatus: status.Enum(),
		TenantId:    e.TenantID,
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

func (r *OutingApplicationRepo) Create(
	ctx context.Context,
	tid, uid uint32,
	reason, destination string,
	startTime, endTime time.Time,
) (uint32, error) {
	entity, err := r.entClient.Client().OutingApplication.Create().
		SetReason(reason).
		SetDestination(destination).
		SetStartTime(startTime).
		SetEndTime(endTime).
		SetOutingStatus(outingapplication.OutingStatusPending).
		SetTenantID(tid).
		SetCreatedBy(uid).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		r.log.Errorf("insert outing application failed: %s", err.Error())
		return 0, oaV1.ErrorInternalServerError("insert outing application failed")
	}
	return entity.ID, nil
}

func (r *OutingApplicationRepo) Get(ctx context.Context, tid, id uint32) (*oaV1.OutingApplication, error) {
	entity, err := r.entClient.Client().OutingApplication.Query().
		Where(
			outingapplication.IDEQ(id),
			outingapplication.TenantIDEQ(tid),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, oaV1.ErrorNotFound("outing application not found")
		}
		r.log.Errorf("query outing application failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query outing application failed")
	}
	return outingApplicationToDTO(entity), nil
}

func (r *OutingApplicationRepo) UpdateStatus(ctx context.Context, tid, id uint32, status oaV1.OutingApplication_OutingStatus) error {
	if _, err := r.entClient.Client().OutingApplication.Update().
		Where(
			outingapplication.IDEQ(id),
			outingapplication.TenantIDEQ(tid),
		).
		SetOutingStatus(outingStatusToEntity(status)).
		SetUpdatedAt(time.Now()).
		Save(ctx); err != nil {
		r.log.Errorf("update outing application status failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("update outing application status failed")
	}
	return nil
}

func (r *OutingApplicationRepo) SetInstanceID(ctx context.Context, tid, id, instanceID uint32) error {
	if _, err := r.entClient.Client().OutingApplication.Update().
		Where(
			outingapplication.IDEQ(id),
			outingapplication.TenantIDEQ(tid),
		).
		SetInstanceID(instanceID).
		SetUpdatedAt(time.Now()).
		Save(ctx); err != nil {
		r.log.Errorf("link outing application instance failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("link outing application instance failed")
	}
	return nil
}

func (r *OutingApplicationRepo) List(
	ctx context.Context,
	tid, userID uint32,
	status oaV1.OutingApplication_OutingStatus,
	page, pageSize int32,
) ([]*oaV1.OutingApplication, int, error) {
	query := r.entClient.Client().OutingApplication.Query().
		Where(outingapplication.TenantIDEQ(tid))
	if userID != 0 {
		query = query.Where(outingapplication.CreatedByEQ(userID))
	}
	if status != 0 {
		query = query.Where(outingapplication.OutingStatusEQ(outingStatusToEntity(status)))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		r.log.Errorf("count outing applications failed: %s", err.Error())
		return nil, 0, oaV1.ErrorInternalServerError("count outing applications failed")
	}
	if pageSize > 0 {
		if page < 1 {
			page = 1
		}
		query = query.Offset((int(page) - 1) * int(pageSize)).Limit(int(pageSize))
	}
	entities, err := query.Order(ent.Desc(outingapplication.FieldID)).All(ctx)
	if err != nil {
		r.log.Errorf("list outing applications failed: %s", err.Error())
		return nil, 0, oaV1.ErrorInternalServerError("list outing applications failed")
	}

	items := make([]*oaV1.OutingApplication, 0, len(entities))
	for _, e := range entities {
		items = append(items, outingApplicationToDTO(e))
	}
	return items, total, nil
}

func (r *OutingApplicationRepo) GetEntity(ctx context.Context, tid, id uint32) (*ent.OutingApplication, error) {
	entity, err := r.entClient.Client().OutingApplication.Query().
		Where(
			outingapplication.IDEQ(id),
			outingapplication.TenantIDEQ(tid),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf("query outing application entity failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query outing application failed")
	}
	return entity, nil
}
