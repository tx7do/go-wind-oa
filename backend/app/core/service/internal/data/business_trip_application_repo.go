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
	"go-wind-oa/app/core/service/internal/data/ent/businesstripapplication"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

type BusinessTripApplicationRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewBusinessTripApplicationRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *BusinessTripApplicationRepo {
	return &BusinessTripApplicationRepo{
		log:       ctx.NewLoggerHelper("business-trip-application/repo/core-service"),
		entClient: entClient,
	}
}

func tripStatusToProto(s businesstripapplication.TripStatus) oaV1.BusinessTripApplication_BusinessTripStatus {
	switch s {
	case businesstripapplication.TripStatusApproved:
		return oaV1.BusinessTripApplication_APPROVED
	case businesstripapplication.TripStatusRejected:
		return oaV1.BusinessTripApplication_REJECTED
	case businesstripapplication.TripStatusWithdrawn:
		return oaV1.BusinessTripApplication_WITHDRAWN
	default:
		return oaV1.BusinessTripApplication_PENDING
	}
}

func tripStatusToEntity(s oaV1.BusinessTripApplication_BusinessTripStatus) businesstripapplication.TripStatus {
	switch s {
	case oaV1.BusinessTripApplication_APPROVED:
		return businesstripapplication.TripStatusApproved
	case oaV1.BusinessTripApplication_REJECTED:
		return businesstripapplication.TripStatusRejected
	case oaV1.BusinessTripApplication_WITHDRAWN:
		return businesstripapplication.TripStatusWithdrawn
	default:
		return businesstripapplication.TripStatusPending
	}
}

func businessTripApplicationToDTO(e *ent.BusinessTripApplication) *oaV1.BusinessTripApplication {
	if e == nil {
		return nil
	}
	status := oaV1.BusinessTripApplication_PENDING
	if e.TripStatus != nil {
		status = tripStatusToProto(*e.TripStatus)
	}
	dto := &oaV1.BusinessTripApplication{
		Id:          trans.Ptr(e.ID),
		Title:       trans.Ptr(e.Title),
		Destination: trans.Ptr(e.Destination),
		StartDate:   timeutil.TimeToTimestamppb(&e.StartDate),
		EndDate:     timeutil.TimeToTimestamppb(&e.EndDate),
		Itinerary:   trans.Ptr(e.Itinerary),
		TripStatus:  status.Enum(),
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

func (r *BusinessTripApplicationRepo) Create(
	ctx context.Context,
	tid, uid uint32,
	title, destination string,
	startDate, endDate time.Time,
	itinerary string,
) (uint32, error) {
	entity, err := r.entClient.Client().BusinessTripApplication.Create().
		SetTitle(title).
		SetDestination(destination).
		SetStartDate(startDate).
		SetEndDate(endDate).
		SetItinerary(itinerary).
		SetTripStatus(businesstripapplication.TripStatusPending).
		SetTenantID(tid).
		SetCreatedBy(uid).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		r.log.Errorf("insert business trip application failed: %s", err.Error())
		return 0, oaV1.ErrorInternalServerError("insert business trip application failed")
	}
	return entity.ID, nil
}

// Get 按主键读取。
func (r *BusinessTripApplicationRepo) Get(ctx context.Context, tid, id uint32) (*oaV1.BusinessTripApplication, error) {
	entity, err := r.entClient.Client().BusinessTripApplication.Query().
		Where(
			businesstripapplication.IDEQ(id),
			businesstripapplication.TenantIDEQ(tid),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, oaV1.ErrorNotFound("business trip application not found")
		}
		r.log.Errorf("query business trip application failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query business trip application failed")
	}
	return businessTripApplicationToDTO(entity), nil
}

func (r *BusinessTripApplicationRepo) UpdateStatus(ctx context.Context, tid, id uint32, status oaV1.BusinessTripApplication_BusinessTripStatus) error {
	if _, err := r.entClient.Client().BusinessTripApplication.Update().
		Where(
			businesstripapplication.IDEQ(id),
			businesstripapplication.TenantIDEQ(tid),
		).
		SetTripStatus(tripStatusToEntity(status)).
		SetUpdatedAt(time.Now()).
		Save(ctx); err != nil {
		r.log.Errorf("update business trip application status failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("update business trip application status failed")
	}
	return nil
}

func (r *BusinessTripApplicationRepo) SetInstanceID(ctx context.Context, tid, id, instanceID uint32) error {
	if _, err := r.entClient.Client().BusinessTripApplication.Update().
		Where(
			businesstripapplication.IDEQ(id),
			businesstripapplication.TenantIDEQ(tid),
		).
		SetInstanceID(instanceID).
		SetUpdatedAt(time.Now()).
		Save(ctx); err != nil {
		r.log.Errorf("link business trip application instance failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("link business trip application instance failed")
	}
	return nil
}

// List 查询申请单。userID==0 时查全部（admin）；status 非零值时按状态过滤。
// pageSize>0 时分页（page 从 1 起）并返回真实总数，否则全量（total=条数）。
func (r *BusinessTripApplicationRepo) List(
	ctx context.Context,
	tid, userID uint32,
	status oaV1.BusinessTripApplication_BusinessTripStatus,
	page, pageSize int32,
) ([]*oaV1.BusinessTripApplication, int, error) {
	query := r.entClient.Client().BusinessTripApplication.Query().
		Where(businesstripapplication.TenantIDEQ(tid))
	if userID != 0 {
		query = query.Where(businesstripapplication.CreatedByEQ(userID))
	}
	if status != 0 {
		query = query.Where(businesstripapplication.TripStatusEQ(tripStatusToEntity(status)))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		r.log.Errorf("count business trip applications failed: %s", err.Error())
		return nil, 0, oaV1.ErrorInternalServerError("count business trip applications failed")
	}
	if pageSize > 0 {
		if page < 1 {
			page = 1
		}
		query = query.Offset((int(page) - 1) * int(pageSize)).Limit(int(pageSize))
	}
	entities, err := query.Order(ent.Desc(businesstripapplication.FieldID)).All(ctx)
	if err != nil {
		r.log.Errorf("list business trip applications failed: %s", err.Error())
		return nil, 0, oaV1.ErrorInternalServerError("list business trip applications failed")
	}

	items := make([]*oaV1.BusinessTripApplication, 0, len(entities))
	for _, e := range entities {
		items = append(items, businessTripApplicationToDTO(e))
	}
	return items, total, nil
}

// GetEntity 供工作流终结回调读取原始实体（校验关联与状态同步）。不存在返回 nil。
func (r *BusinessTripApplicationRepo) GetEntity(ctx context.Context, tid, id uint32) (*ent.BusinessTripApplication, error) {
	entity, err := r.entClient.Client().BusinessTripApplication.Query().
		Where(
			businesstripapplication.IDEQ(id),
			businesstripapplication.TenantIDEQ(tid),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf("query business trip application entity failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query business trip application failed")
	}
	return entity, nil
}
