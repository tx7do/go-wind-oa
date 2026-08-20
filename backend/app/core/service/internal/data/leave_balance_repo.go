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
	"go-wind-oa/app/core/service/internal/data/ent/leavebalance"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

type LeaveBalanceRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewLeaveBalanceRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *LeaveBalanceRepo {
	return &LeaveBalanceRepo{
		log:       ctx.NewLoggerHelper("leave-balance/repo/core-service"),
		entClient: entClient,
	}
}

func leaveBalanceToDTO(e *ent.LeaveBalance) *oaV1.LeaveBalance {
	if e == nil {
		return nil
	}
	dto := &oaV1.LeaveBalance{
		Id:          trans.Ptr(e.ID),
		UserId:      trans.Ptr(e.UserID),
		LeaveTypeId: trans.Ptr(e.LeaveTypeID),
		Year:        trans.Ptr(int32(e.Year)),
		TotalDays:   trans.Ptr(e.TotalDays),
		UsedDays:    trans.Ptr(e.UsedDays),
		TenantId:    e.TenantID,
	}
	dto.CreatedAt = timeutil.TimeToTimestamppb(e.CreatedAt)
	return dto
}

// Get 读取指定用户/类型/年度的额度，不存在返回 nil（无错误）。
func (r *LeaveBalanceRepo) Get(ctx context.Context, tid, userID, typeID uint32, year int) (*oaV1.LeaveBalance, error) {
	entity, err := r.entClient.Client().LeaveBalance.Query().
		Where(
			leavebalance.TenantIDEQ(tid),
			leavebalance.UserIDEQ(userID),
			leavebalance.LeaveTypeIDEQ(typeID),
			leavebalance.YearEQ(year),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf("query leave balance failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query leave balance failed")
	}
	return leaveBalanceToDTO(entity), nil
}

// Grant 授予/调整额度：存在则覆盖 total_days，不存在则创建。
func (r *LeaveBalanceRepo) Grant(ctx context.Context, tid, operatorID, userID, typeID uint32, year int, totalDays float64) error {
	_, err := r.entClient.Client().LeaveBalance.Query().
		Where(
			leavebalance.TenantIDEQ(tid),
			leavebalance.UserIDEQ(userID),
			leavebalance.LeaveTypeIDEQ(typeID),
			leavebalance.YearEQ(year),
		).
		Only(ctx)
	if ent.IsNotFound(err) {
		if _, err := r.entClient.Client().LeaveBalance.Create().
			SetUserID(userID).
			SetLeaveTypeID(typeID).
			SetYear(year).
			SetTotalDays(totalDays).
			SetUsedDays(0).
			SetTenantID(tid).
			SetCreatedBy(operatorID).
			SetCreatedAt(time.Now()).
			Save(ctx); err != nil {
			r.log.Errorf("insert leave balance failed: %s", err.Error())
			return oaV1.ErrorInternalServerError("insert leave balance failed")
		}
		return nil
	}
	if err != nil {
		r.log.Errorf("query leave balance failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("query leave balance failed")
	}

	if _, err := r.entClient.Client().LeaveBalance.Update().
		Where(
			leavebalance.TenantIDEQ(tid),
			leavebalance.UserIDEQ(userID),
			leavebalance.LeaveTypeIDEQ(typeID),
			leavebalance.YearEQ(year),
		).
		SetTotalDays(totalDays).
		SetUpdatedAt(time.Now()).
		SetUpdatedBy(operatorID).
		Save(ctx); err != nil {
		r.log.Errorf("update leave balance failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("update leave balance failed")
	}
	return nil
}

// AddUsedDays 审批通过后扣减额度。额度不足时不做钳制（提交时已校验，此处容忍并发窗口）。
func (r *LeaveBalanceRepo) AddUsedDays(ctx context.Context, tid, userID, typeID uint32, year int, days float64) error {
	if _, err := r.entClient.Client().LeaveBalance.Update().
		Where(
			leavebalance.TenantIDEQ(tid),
			leavebalance.UserIDEQ(userID),
			leavebalance.LeaveTypeIDEQ(typeID),
			leavebalance.YearEQ(year),
		).
		AddUsedDays(days).
		SetUpdatedAt(time.Now()).
		Save(ctx); err != nil {
		r.log.Errorf("deduct leave balance failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("deduct leave balance failed")
	}
	return nil
}

// List 查询额度。userID==0 时查全部（admin）。
func (r *LeaveBalanceRepo) List(ctx context.Context, tid, userID uint32, year int) ([]*oaV1.LeaveBalance, error) {
	query := r.entClient.Client().LeaveBalance.Query().
		Where(
			leavebalance.TenantIDEQ(tid),
			leavebalance.YearEQ(year),
		)
	if userID != 0 {
		query = query.Where(leavebalance.UserIDEQ(userID))
	}
	entities, err := query.All(ctx)
	if err != nil {
		r.log.Errorf("list leave balances failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("list leave balances failed")
	}
	items := make([]*oaV1.LeaveBalance, 0, len(entities))
	for _, e := range entities {
		items = append(items, leaveBalanceToDTO(e))
	}
	return items, nil
}
