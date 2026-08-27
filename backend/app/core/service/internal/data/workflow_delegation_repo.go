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
	"go-wind-oa/app/core/service/internal/data/ent/workflowdelegation"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

type WorkflowDelegationRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewWorkflowDelegationRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *WorkflowDelegationRepo {
	return &WorkflowDelegationRepo{
		log:       ctx.NewLoggerHelper("workflow-delegation/repo/core-service"),
		entClient: entClient,
	}
}

func delegationToDTO(e *ent.WorkflowDelegation) *oaV1.WorkflowDelegation {
	if e == nil {
		return nil
	}
	dto := &oaV1.WorkflowDelegation{}
	if e.DelegatorUserID != nil {
		dto.DelegatorUserId = e.DelegatorUserID
	}
	if e.DelegateUserID != nil {
		dto.DelegateUserId = e.DelegateUserID
	}
	dto.TenantId = e.TenantID
	dto.Id = trans.Ptr(e.ID)
	dto.CreatedAt = timeutil.TimeToTimestamppb(e.CreatedAt)
	return dto
}

// Upsert 创建或更新委托记录（按 delegator_user_id 租户内唯一）。
func (r *WorkflowDelegationRepo) Upsert(
	ctx context.Context, tid, uid, delegatorUserID, delegateUserID uint32,
) (*oaV1.WorkflowDelegation, error) {
	builder := r.entClient.Client().WorkflowDelegation.Create().
		SetTenantID(tid).
		SetCreatedBy(uid).
		SetCreatedAt(time.Now()).
		SetDelegatorUserID(delegatorUserID).
		SetDelegateUserID(delegateUserID)
	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("upsert workflow delegation failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("upsert workflow delegation failed")
	}
	return delegationToDTO(entity), nil
}

func (r *WorkflowDelegationRepo) List(
	ctx context.Context, tid uint32, paging *paginationV1.PagingRequest,
) (*oaV1.ListWorkflowDelegationResponse, error) {
	query := r.entClient.Client().WorkflowDelegation.Query().
		Where(workflowdelegation.TenantIDEQ(tid))

	total, err := query.Clone().Count(ctx)
	if err != nil {
		r.log.Errorf("count workflow delegations failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("count workflow delegations failed")
	}
	entities, err := query.All(ctx)
	if err != nil {
		r.log.Errorf("list workflow delegations failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("list workflow delegations failed")
	}

	items := make([]*oaV1.WorkflowDelegation, 0, len(entities))
	for _, e := range entities {
		items = append(items, delegationToDTO(e))
	}
	return &oaV1.ListWorkflowDelegationResponse{Items: items, Total: uint64(total)}, nil
}

func (r *WorkflowDelegationRepo) DeleteByID(
	ctx context.Context, tid, id uint32,
) error {
	n, err := r.entClient.Client().WorkflowDelegation.Delete().
		Where(
			workflowdelegation.IDEQ(id),
			workflowdelegation.TenantIDEQ(tid),
		).
		Exec(ctx)
	if err != nil {
		r.log.Errorf("delete workflow delegation failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("delete workflow delegation failed")
	}
	if n == 0 {
		return oaV1.ErrorNotFound("workflow delegation not found")
	}
	return nil
}

// ResolveDelegate 返回 delegatorUserID 当前生效的代理人 ID。
// 无委托记录时返回 (0, nil)。
func (r *WorkflowDelegationRepo) ResolveDelegate(
	ctx context.Context, tid, delegatorUserID uint32,
) (uint32, error) {
	entity, err := r.entClient.Client().WorkflowDelegation.Query().
		Where(
			workflowdelegation.TenantIDEQ(tid),
			workflowdelegation.DelegatorUserIDEQ(delegatorUserID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, nil
		}
		r.log.Errorf("query workflow delegation failed: %s", err.Error())
		return 0, oaV1.ErrorInternalServerError("query workflow delegation failed")
	}
	if entity.DelegateUserID == nil {
		return 0, nil
	}
	return *entity.DelegateUserID, nil
}
