package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"

	"go-wind-oa/app/core/service/internal/data/ent"
	"go-wind-oa/app/core/service/internal/data/ent/workflowinstance"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

type WorkflowInstanceRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper                  *mapper.CopierMapper[oaV1.WorkflowInstance, ent.WorkflowInstance]
	instanceStatusConverter *mapper.EnumTypeConverter[oaV1.WorkflowInstance_InstanceStatus, workflowinstance.InstanceStatus]
}

func NewWorkflowInstanceRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *WorkflowInstanceRepo {
	repo := &WorkflowInstanceRepo{
		log:                     ctx.NewLoggerHelper("workflow-instance/repo/core-service"),
		entClient:               entClient,
		mapper:                  mapper.NewCopierMapper[oaV1.WorkflowInstance, ent.WorkflowInstance](),
		instanceStatusConverter: mapper.NewEnumTypeConverter[oaV1.WorkflowInstance_InstanceStatus, workflowinstance.InstanceStatus](oaV1.WorkflowInstance_InstanceStatus_name, oaV1.WorkflowInstance_InstanceStatus_value),
	}

	repo.init()

	return repo
}

func (r *WorkflowInstanceRepo) init() {
	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(r.instanceStatusConverter.NewConverterPair())
}

func (r *WorkflowInstanceRepo) Create(
	ctx context.Context,
	tenantID uint32,
	creatorUserID uint32,
	definitionID uint32,
	formData string,
	businessType string,
	businessID uint32,
) (uint32, error) {
	builder := r.entClient.Client().WorkflowInstance.Create().
		SetDefinitionID(definitionID).
		SetInstanceStatus(workflowinstance.InstanceStatusPending).
		SetCurrentNodeIndex(0).
		SetNillableFormData(&formData).
		SetTenantID(tenantID).
		SetCreatedBy(creatorUserID).
		SetCreatedAt(time.Now())
	if businessType != "" {
		builder.SetNillableBusinessType(&businessType)
	}
	if businessID != 0 {
		builder.SetNillableBusinessID(&businessID)
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert workflow instance failed: %s", err.Error())
		return 0, oaV1.ErrorInternalServerError("insert workflow instance failed")
	}
	return entity.ID, nil
}

// CreateWithTx 事务内建实例。builder 源自 tx。
func (r *WorkflowInstanceRepo) CreateWithTx(
	ctx context.Context, tx *ent.Tx,
	tenantID uint32, creatorUserID uint32, definitionID uint32, formData string, businessType string, businessID uint32,
) (uint32, error) {
	builder := tx.WorkflowInstance.Create().
		SetDefinitionID(definitionID).
		SetInstanceStatus(workflowinstance.InstanceStatusPending).
		SetCurrentNodeIndex(0).
		SetNillableFormData(&formData).
		SetTenantID(tenantID).
		SetCreatedBy(creatorUserID).
		SetCreatedAt(time.Now())
	if businessType != "" {
		builder.SetNillableBusinessType(&businessType)
	}
	if businessID != 0 {
		builder.SetNillableBusinessID(&businessID)
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert workflow instance failed: %s", err.Error())
		return 0, oaV1.ErrorInternalServerError("insert workflow instance failed")
	}
	return entity.ID, nil
}

// GetState 直读 entity 的 InstanceStatus 字段（绕过 mapper）。
// 仅返回是否处于 PENDING（活跃），供状态机校验。终结态（APPROVED/REJECTED）返回 false。
func (r *WorkflowInstanceRepo) GetState(ctx context.Context, id uint32, tenantID uint32) (bool, error) {
	entity, err := r.entClient.Client().WorkflowInstance.Query().
		Where(
			workflowinstance.IDEQ(id),
			workflowinstance.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return false, oaV1.ErrorNotFound("workflow instance not found")
		}
		r.log.Errorf("query instance state failed: %s", err.Error())
		return false, oaV1.ErrorInternalServerError("query instance state failed")
	}
	if entity.InstanceStatus == nil {
		return false, oaV1.ErrorConflict("instance state corrupt")
	}
	return *entity.InstanceStatus == workflowinstance.InstanceStatusPending, nil
}

// UpdateStatus 推进实例状态。newNodeIndex==nil 表示终结态，清空 current_node_index。
func (r *WorkflowInstanceRepo) UpdateStatus(
	ctx context.Context,
	id uint32,
	tenantID uint32,
	newStatus *oaV1.WorkflowInstance_InstanceStatus,
	newNodeIndex *int,
) error {
	builder := r.entClient.Client().WorkflowInstance.Update()
	builder.Where(
		workflowinstance.IDEQ(id),
		workflowinstance.TenantIDEQ(tenantID),
	)
	builder.SetNillableInstanceStatus(r.instanceStatusConverter.ToEntity(newStatus))
	if newNodeIndex == nil {
		builder.ClearCurrentNodeIndex()
	} else {
		builder.SetNillableCurrentNodeIndex(newNodeIndex)
	}
	builder.SetUpdatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("update instance state failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("update instance state failed")
	}
	return nil
}

// Txn 开启事务，fn 内的跨 repo 写入原子提交/回滚。状态机推进路径专用。
func (r *WorkflowInstanceRepo) Txn(ctx context.Context, fn func(tx *ent.Tx) error) (err error) {
	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf("transaction commit failed: %s", commitErr.Error())
			err = oaV1.ErrorInternalServerError("transaction commit failed")
		}
	}()
	return fn(tx)
}

// UpdateStatusWithTx 事务内推进实例状态。builder 源自 tx 而非 client。
func (r *WorkflowInstanceRepo) UpdateStatusWithTx(
	ctx context.Context, tx *ent.Tx,
	id uint32, tenantID uint32,
	newStatus *oaV1.WorkflowInstance_InstanceStatus, newNodeIndex *int,
) error {
	builder := tx.WorkflowInstance.Update()
	builder.Where(
		workflowinstance.IDEQ(id),
		workflowinstance.TenantIDEQ(tenantID),
	)
	builder.SetNillableInstanceStatus(r.instanceStatusConverter.ToEntity(newStatus))
	if newNodeIndex == nil {
		builder.ClearCurrentNodeIndex()
	} else {
		builder.SetNillableCurrentNodeIndex(newNodeIndex)
	}
	builder.SetUpdatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("update instance state failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("update instance state failed")
	}
	return nil
}
func (r *WorkflowInstanceRepo) GetMeta(ctx context.Context, id uint32, tenantID uint32) (uint32, string, uint32, error) {
	entity, err := r.entClient.Client().WorkflowInstance.Query().
		Where(
			workflowinstance.IDEQ(id),
			workflowinstance.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, "", 0, oaV1.ErrorNotFound("workflow instance not found")
		}
		r.log.Errorf("query instance meta failed: %s", err.Error())
		return 0, "", 0, oaV1.ErrorInternalServerError("query instance failed")
	}
	if entity.CreatedBy == nil || *entity.CreatedBy == 0 {
		return 0, "", 0, oaV1.ErrorConflict("instance creator missing")
	}
	creator := *entity.CreatedBy
	businessType := ""
	if entity.BusinessType != nil {
		businessType = *entity.BusinessType
	}
	businessID := uint32(0)
	if entity.BusinessID != nil {
		businessID = *entity.BusinessID
	}
	return creator, businessType, businessID, nil
}

// GetFormData 读取实例的申请表单数据（JSON 文本），供审批人查看申请内容。
func (r *WorkflowInstanceRepo) GetFormData(ctx context.Context, id uint32, tenantID uint32) (string, error) {
	entity, err := r.entClient.Client().WorkflowInstance.Query().
		Where(
			workflowinstance.IDEQ(id),
			workflowinstance.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", oaV1.ErrorNotFound("workflow instance not found")
		}
		r.log.Errorf("query instance form data failed: %s", err.Error())
		return "", oaV1.ErrorInternalServerError("query instance failed")
	}
	if entity.FormData == nil {
		return "", nil
	}
	return *entity.FormData, nil
}

// GetDefinitionNodeConfig 经 instance→definition 边读取定义的 node_config（JSON 文本）。
// 供状态机 APPROVE 推进时解析下一节点审批人。
func (r *WorkflowInstanceRepo) GetDefinitionNodeConfig(ctx context.Context, id uint32, tenantID uint32) (string, error) {
	entity, err := r.entClient.Client().WorkflowInstance.Query().
		Where(
			workflowinstance.IDEQ(id),
			workflowinstance.TenantIDEQ(tenantID),
		).
		WithDefinition().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", oaV1.ErrorNotFound("workflow instance not found")
		}
		r.log.Errorf("query instance for node config failed: %s", err.Error())
		return "", oaV1.ErrorInternalServerError("query instance failed")
	}
	def := entity.Edges.Definition
	if def == nil || def.NodeConfig == nil {
		return "", oaV1.ErrorConflict("definition node config missing")
	}
	return *def.NodeConfig, nil
}

// ListByCreator “我的申请”列表。按创建者+租户查询实例。pageSize>0 时分页并返回真实总数。
func (r *WorkflowInstanceRepo) ListByCreator(ctx context.Context, tenantID uint32, creatorUserID uint32, page, pageSize int32) ([]*oaV1.MyTaskItem, int, error) {
	query := r.entClient.Client().WorkflowInstance.Query().
		Where(
			workflowinstance.CreatedByEQ(creatorUserID),
			workflowinstance.TenantIDEQ(tenantID),
		)
	total, err := query.Clone().Count(ctx)
	if err != nil {
		r.log.Errorf("count instances by creator failed: %s", err.Error())
		return nil, 0, oaV1.ErrorInternalServerError("count instances failed")
	}
	if pageSize > 0 {
		if page < 1 {
			page = 1
		}
		query = query.Offset((int(page) - 1) * int(pageSize)).Limit(int(pageSize))
	}
	entities, err := query.Order(ent.Desc(workflowinstance.FieldID)).All(ctx)
	if err != nil {
		r.log.Errorf("list instances by creator failed: %s", err.Error())
		return nil, 0, oaV1.ErrorInternalServerError("list instances failed")
	}

	items := make([]*oaV1.MyTaskItem, 0, len(entities))
	for _, e := range entities {
		item := &oaV1.MyTaskItem{}
		if e.ID != 0 {
			item.InstanceId = trans.Ptr(e.ID)
		}
		if e.CreatedAt != nil {
			item.CreatedAt = timeutil.TimeToTimestamppb(e.CreatedAt)
		}
		label := instanceStatusLabel(e.InstanceStatus)
		if label != "" {
			item.StatusLabel = trans.Ptr(label)
		}
		items = append(items, item)
	}
	return items, total, nil
}

func instanceStatusLabel(s *workflowinstance.InstanceStatus) string {
	if s == nil {
		return ""
	}
	switch *s {
	case workflowinstance.InstanceStatusPending:
		return "进行中"
	case workflowinstance.InstanceStatusApproved:
		return "已通过"
	case workflowinstance.InstanceStatusRejected:
		return "已驳回"
	case workflowinstance.InstanceStatusWithdrawn:
		return "已撤回"
	}
	return ""
}
