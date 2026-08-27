package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-oa/app/core/service/internal/data/ent"
	"go-wind-oa/app/core/service/internal/data/ent/workflowinstance"
	"go-wind-oa/app/core/service/internal/data/ent/workflowinstancejoin"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

type WorkflowInstanceJoinRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewWorkflowInstanceJoinRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *WorkflowInstanceJoinRepo {
	repo := &WorkflowInstanceJoinRepo{
		log:       ctx.NewLoggerHelper("workflow-instance-join/repo/core-service"),
		entClient: entClient,
	}
	return repo
}

// Increment 递增汇聚节点的到达计数。若该 join 尚无行则先建行（arrived_count=1）。
// 返回递增后的到达数。
func (r *WorkflowInstanceJoinRepo) Increment(
	ctx context.Context,
	tenantID uint32,
	instanceID uint32,
	joinNodeID string,
) (int, error) {
	// 查找现有行
	entity, err := r.entClient.Client().WorkflowInstanceJoin.Query().
		Where(
			workflowinstancejoin.HasInstanceWith(
				workflowinstance.IDEQ(instanceID),
				workflowinstance.TenantIDEQ(tenantID),
			),
			workflowinstancejoin.JoinNodeIDEQ(joinNodeID),
			workflowinstancejoin.TenantIDEQ(tenantID),
		).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			// 首次到达：建行 arrived_count=1
			builder := r.entClient.Client().WorkflowInstanceJoin.Create().
				SetInstanceID(instanceID).
				SetJoinNodeID(joinNodeID).
				SetArrivedCount(1).
				SetTenantID(tenantID).
				SetCreatedAt(time.Now())
			created, err := builder.Save(ctx)
			if err != nil {
				r.log.Errorf("create join record failed: %s", err.Error())
				return 0, oaV1.ErrorInternalServerError("create join record failed")
			}
			if created.ArrivedCount == nil {
				return 0, oaV1.ErrorConflict("join record state corrupt")
			}
			return *created.ArrivedCount, nil
		}
		r.log.Errorf("query join record failed: %s", err.Error())
		return 0, oaV1.ErrorInternalServerError("query join record failed")
	}

	if entity.ArrivedCount == nil {
		return 0, oaV1.ErrorConflict("join record state corrupt")
	}
	newCount := *entity.ArrivedCount + 1

	builder := r.entClient.Client().WorkflowInstanceJoin.Update().
		Where(
			workflowinstancejoin.IDEQ(entity.ID),
			workflowinstancejoin.TenantIDEQ(tenantID),
		)
	builder.SetArrivedCount(newCount)
	builder.SetUpdatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("update join record failed: %s", err.Error())
		return 0, oaV1.ErrorInternalServerError("update join record failed")
	}
	return newCount, nil
}

// IncrementWithTx 事务内递增汇聚节点到达计数。
func (r *WorkflowInstanceJoinRepo) IncrementWithTx(
	ctx context.Context, tx *ent.Tx,
	tenantID uint32, instanceID uint32, joinNodeID string,
) (int, error) {
	// 查找现有行
	entity, err := tx.WorkflowInstanceJoin.Query().
		Where(
			workflowinstancejoin.HasInstanceWith(
				workflowinstance.IDEQ(instanceID),
				workflowinstance.TenantIDEQ(tenantID),
			),
			workflowinstancejoin.JoinNodeIDEQ(joinNodeID),
			workflowinstancejoin.TenantIDEQ(tenantID),
		).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			// 首次到达：建行 arrived_count=1
			builder := tx.WorkflowInstanceJoin.Create().
				SetInstanceID(instanceID).
				SetJoinNodeID(joinNodeID).
				SetArrivedCount(1).
				SetTenantID(tenantID).
				SetCreatedAt(time.Now())
			created, err := builder.Save(ctx)
			if err != nil {
				r.log.Errorf("create join record failed: %s", err.Error())
				return 0, oaV1.ErrorInternalServerError("create join record failed")
			}
			if created.ArrivedCount == nil {
				return 0, oaV1.ErrorConflict("join record state corrupt")
			}
			return *created.ArrivedCount, nil
		}
		r.log.Errorf("query join record failed: %s", err.Error())
		return 0, oaV1.ErrorInternalServerError("query join record failed")
	}

	if entity.ArrivedCount == nil {
		return 0, oaV1.ErrorConflict("join record state corrupt")
	}
	newCount := *entity.ArrivedCount + 1

	builder := tx.WorkflowInstanceJoin.Update().
		Where(
			workflowinstancejoin.IDEQ(entity.ID),
			workflowinstancejoin.TenantIDEQ(tenantID),
		)
	builder.SetArrivedCount(newCount)
	builder.SetUpdatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("update join record failed: %s", err.Error())
		return 0, oaV1.ErrorInternalServerError("update join record failed")
	}
	return newCount, nil
}

// DeleteByInstance 删除实例的全部汇聚计数行。
func (r *WorkflowInstanceJoinRepo) DeleteByInstance(
	ctx context.Context,
	tenantID uint32,
	instanceID uint32,
) error {
	_, err := r.entClient.Client().WorkflowInstanceJoin.Delete().
		Where(
			workflowinstancejoin.HasInstanceWith(
				workflowinstance.IDEQ(instanceID),
				workflowinstance.TenantIDEQ(tenantID),
			),
			workflowinstancejoin.TenantIDEQ(tenantID),
		).
		Exec(ctx)
	if err != nil {
		r.log.Errorf("delete join records failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("delete join records failed")
	}
	return nil
}

// DeleteByInstanceWithTx 事务内删除实例的全部汇聚计数行。
func (r *WorkflowInstanceJoinRepo) DeleteByInstanceWithTx(
	ctx context.Context, tx *ent.Tx,
	tenantID uint32, instanceID uint32,
) error {
	_, err := tx.WorkflowInstanceJoin.Delete().
		Where(
			workflowinstancejoin.HasInstanceWith(
				workflowinstance.IDEQ(instanceID),
				workflowinstance.TenantIDEQ(tenantID),
			),
			workflowinstancejoin.TenantIDEQ(tenantID),
		).
		Exec(ctx)
	if err != nil {
		r.log.Errorf("delete join records failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("delete join records failed")
	}
	return nil
}

// HasByInstance 判断实例是否存在汇聚计数行（用于活跃节点判定）。
func (r *WorkflowInstanceJoinRepo) HasByInstance(
	ctx context.Context,
	tenantID uint32,
	instanceID uint32,
) (bool, error) {
	count, err := r.entClient.Client().WorkflowInstanceJoin.Query().
		Where(
			workflowinstancejoin.HasInstanceWith(
				workflowinstance.IDEQ(instanceID),
				workflowinstance.TenantIDEQ(tenantID),
			),
			workflowinstancejoin.TenantIDEQ(tenantID),
		).
		Count(ctx)
	if err != nil {
		r.log.Errorf("count join records failed: %s", err.Error())
		return false, oaV1.ErrorInternalServerError("count join records failed")
	}
	return count > 0, nil
}
