package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-oa/app/core/service/internal/data/ent"
	"go-wind-oa/app/core/service/internal/data/ent/leavetype"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

type LeaveTypeRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewLeaveTypeRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *LeaveTypeRepo {
	return &LeaveTypeRepo{
		log:       ctx.NewLoggerHelper("leave-type/repo/core-service"),
		entClient: entClient,
	}
}

func leaveTypeToDTO(e *ent.LeaveType) *oaV1.LeaveType {
	if e == nil {
		return nil
	}
	dto := &oaV1.LeaveType{
		Id:       trans.Ptr(e.ID),
		Code:     trans.Ptr(e.Code),
		Name:     trans.Ptr(e.Name),
		TenantId: e.TenantID,
	}
	if e.CreatedBy != nil {
		dto.CreatedBy = e.CreatedBy
	}
	dto.CreatedAt = timeutil.TimeToTimestamppb(e.CreatedAt)
	return dto
}

func (r *LeaveTypeRepo) Create(ctx context.Context, tid, uid uint32, code, name string) (*oaV1.LeaveType, error) {
	entity, err := r.entClient.Client().LeaveType.Create().
		SetCode(code).
		SetName(name).
		SetTenantID(tid).
		SetCreatedBy(uid).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		r.log.Errorf("insert leave type failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("insert leave type failed")
	}
	return leaveTypeToDTO(entity), nil
}

func (r *LeaveTypeRepo) GetByID(ctx context.Context, tid, id uint32) (*oaV1.LeaveType, error) {
	entity, err := r.entClient.Client().LeaveType.Query().
		Where(
			leavetype.IDEQ(id),
			leavetype.TenantIDEQ(tid),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, oaV1.ErrorNotFound("leave type not found")
		}
		r.log.Errorf("query leave type failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query leave type failed")
	}
	return leaveTypeToDTO(entity), nil
}

func (r *LeaveTypeRepo) List(ctx context.Context, tid uint32, paging *paginationV1.PagingRequest) (*oaV1.ListLeaveTypesResponse, error) {
	query := r.entClient.Client().LeaveType.Query().
		Where(leavetype.TenantIDEQ(tid))

	total, err := query.Clone().Count(ctx)
	if err != nil {
		r.log.Errorf("count leave types failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("count leave types failed")
	}
	entities, err := query.All(ctx)
	if err != nil {
		r.log.Errorf("list leave types failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("list leave types failed")
	}

	items := make([]*oaV1.LeaveType, 0, len(entities))
	for _, e := range entities {
		items = append(items, leaveTypeToDTO(e))
	}
	return &oaV1.ListLeaveTypesResponse{Items: items, Total: uint64(total)}, nil
}
