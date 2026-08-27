package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-oa/app/core/service/internal/data/ent"
	"go-wind-oa/app/core/service/internal/data/ent/workflowinstance"
	"go-wind-oa/app/core/service/internal/data/ent/workflowinstanceparentlink"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

type WorkflowInstanceParentLinkRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewWorkflowInstanceParentLinkRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *WorkflowInstanceParentLinkRepo {
	repo := &WorkflowInstanceParentLinkRepo{
		log:       ctx.NewLoggerHelper("workflow-instance-parent-link/repo/core-service"),
		entClient: entClient,
	}
	return repo
}

// Create 创建父子实例关联记录。
func (r *WorkflowInstanceParentLinkRepo) Create(
	ctx context.Context,
	tenantID uint32,
	parentInstanceID uint32,
	subprocessNodeID string,
	childInstanceID uint32,
) error {
	builder := r.entClient.Client().WorkflowInstanceParentLink.Create().
		SetParentInstanceID(parentInstanceID).
		SetSubprocessNodeID(subprocessNodeID).
		SetChildInstanceID(childInstanceID).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now())
	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf("create parent link failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("create parent link failed")
	}
	return nil
}

// GetByChildInstance 按子实例 ID 查找关联记录（子流程终结时恢复父流程用）。
// 经 parent_instance 边加载取父实例 ID。
func (r *WorkflowInstanceParentLinkRepo) GetByChildInstance(
	ctx context.Context,
	tenantID uint32,
	childInstanceID uint32,
) (parentInstanceID uint32, subprocessNodeID string, err error) {
	entity, err := r.entClient.Client().WorkflowInstanceParentLink.Query().
		Where(
			workflowinstanceparentlink.ChildInstanceIDEQ(childInstanceID),
			workflowinstanceparentlink.TenantIDEQ(tenantID),
		).
		WithParentInstance().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, "", oaV1.ErrorNotFound("parent link not found")
		}
		r.log.Errorf("query parent link by child failed: %s", err.Error())
		return 0, "", oaV1.ErrorInternalServerError("query parent link failed")
	}
	parentInst := entity.Edges.ParentInstance
	if parentInst == nil || entity.SubprocessNodeID == nil {
		return 0, "", oaV1.ErrorConflict("parent link state corrupt")
	}
	return parentInst.ID, *entity.SubprocessNodeID, nil
}

// HasByParentInstance 判断父实例是否有未完成的子流程（用于活跃节点判定）。
func (r *WorkflowInstanceParentLinkRepo) HasByParentInstance(
	ctx context.Context,
	tenantID uint32,
	parentInstanceID uint32,
) (bool, error) {
	count, err := r.entClient.Client().WorkflowInstanceParentLink.Query().
		Where(
			workflowinstanceparentlink.HasParentInstanceWith(
				workflowinstance.IDEQ(parentInstanceID),
				workflowinstance.TenantIDEQ(tenantID),
			),
			workflowinstanceparentlink.TenantIDEQ(tenantID),
		).
		Count(ctx)
	if err != nil {
		r.log.Errorf("count parent links failed: %s", err.Error())
		return false, oaV1.ErrorInternalServerError("count parent links failed")
	}
	return count > 0, nil
}

// DeleteByParentInstance 删除父实例的全部关联记录。
func (r *WorkflowInstanceParentLinkRepo) DeleteByParentInstance(
	ctx context.Context,
	tenantID uint32,
	parentInstanceID uint32,
) error {
	_, err := r.entClient.Client().WorkflowInstanceParentLink.Delete().
		Where(
			workflowinstanceparentlink.HasParentInstanceWith(
				workflowinstance.IDEQ(parentInstanceID),
				workflowinstance.TenantIDEQ(tenantID),
			),
			workflowinstanceparentlink.TenantIDEQ(tenantID),
		).
		Exec(ctx)
	if err != nil {
		r.log.Errorf("delete parent links failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("delete parent links failed")
	}
	return nil
}

// DeleteByParentInstanceWithTx 事务内删除父实例的全部关联记录。
func (r *WorkflowInstanceParentLinkRepo) DeleteByParentInstanceWithTx(
	ctx context.Context, tx *ent.Tx,
	tenantID uint32,
	parentInstanceID uint32,
) error {
	_, err := tx.WorkflowInstanceParentLink.Delete().
		Where(
			workflowinstanceparentlink.HasParentInstanceWith(
				workflowinstance.IDEQ(parentInstanceID),
				workflowinstance.TenantIDEQ(tenantID),
			),
			workflowinstanceparentlink.TenantIDEQ(tenantID),
		).
		Exec(ctx)
	if err != nil {
		r.log.Errorf("delete parent links failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("delete parent links failed")
	}
	return nil
}

// DeleteByChildInstance 删除指定子实例的关联记录（子流程终结后清理）。
func (r *WorkflowInstanceParentLinkRepo) DeleteByChildInstance(
	ctx context.Context,
	tenantID uint32,
	childInstanceID uint32,
) error {
	_, err := r.entClient.Client().WorkflowInstanceParentLink.Delete().
		Where(
			workflowinstanceparentlink.ChildInstanceIDEQ(childInstanceID),
			workflowinstanceparentlink.TenantIDEQ(tenantID),
		).
		Exec(ctx)
	if err != nil {
		r.log.Errorf("delete parent link by child failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("delete parent link by child failed")
	}
	return nil
}
