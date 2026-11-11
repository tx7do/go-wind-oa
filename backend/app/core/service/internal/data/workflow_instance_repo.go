package data

import (
	"context"
	"encoding/json"
	"strings"
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
	"go-wind-oa/app/core/service/internal/data/ent/workflowtoken"

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
	tenantID int64,
	creatorUserID int64,
	definitionID int64,
	formData string,
	businessType string,
	businessID int64,
) (int64, error) {
	builder := r.entClient.Client().WorkflowInstance.Create().
		SetDefinitionID(definitionID).
		SetInstanceStatus(workflowinstance.InstanceStatusPending).
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
	tenantID int64, creatorUserID int64, definitionID int64, formData string, businessType string, businessID int64,
) (int64, error) {
	builder := tx.WorkflowInstance.Create().
		SetDefinitionID(definitionID).
		SetInstanceStatus(workflowinstance.InstanceStatusPending).
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
func (r *WorkflowInstanceRepo) GetState(ctx context.Context, id int64, tenantID int64) (bool, error) {
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

// UpdateStatus 切换实例状态（活跃↔终结态）。
func (r *WorkflowInstanceRepo) UpdateStatus(
	ctx context.Context,
	id int64,
	tenantID int64,
	newStatus *oaV1.WorkflowInstance_InstanceStatus,
) error {
	builder := r.entClient.Client().WorkflowInstance.Update()
	builder.Where(
		workflowinstance.IDEQ(id),
		workflowinstance.TenantIDEQ(tenantID),
	)
	builder.SetNillableInstanceStatus(r.instanceStatusConverter.ToEntity(newStatus))
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

// UpdateStatusWithTx 事务内切换实例状态。builder 源自 tx 而非 client。
func (r *WorkflowInstanceRepo) UpdateStatusWithTx(
	ctx context.Context, tx *ent.Tx,
	id int64, tenantID int64,
	newStatus *oaV1.WorkflowInstance_InstanceStatus,
) error {
	builder := tx.WorkflowInstance.Update()
	builder.Where(
		workflowinstance.IDEQ(id),
		workflowinstance.TenantIDEQ(tenantID),
	)
	builder.SetNillableInstanceStatus(r.instanceStatusConverter.ToEntity(newStatus))
	builder.SetUpdatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("update instance state failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("update instance state failed")
	}
	return nil
}
func (r *WorkflowInstanceRepo) GetMeta(ctx context.Context, id int64, tenantID int64) (int64, string, int64, error) {
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
	businessID := int64(0)
	if entity.BusinessID != nil {
		businessID = *entity.BusinessID
	}
	return creator, businessType, businessID, nil
}

// GetFormData 读取实例的申请表单数据（JSON 文本），供审批人查看申请内容。
// 引擎保留键（flow:* 前缀，如包容网关汇合计数）属内部状态，展示前剔除。
func (r *WorkflowInstanceRepo) GetFormData(ctx context.Context, id int64, tenantID int64) (string, error) {
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
	return stripFlowReservedKeys(*entity.FormData), nil
}

// stripFlowReservedKeys 剔除表单 JSON 中引擎保留键（flow:* 前缀）。
// 非对象 JSON（理论上不出现）原样返回。
func stripFlowReservedKeys(formData string) string {
	if !strings.Contains(formData, "flow:") {
		return formData
	}
	m := map[string]any{}
	if err := json.Unmarshal([]byte(formData), &m); err != nil {
		return formData
	}
	changed := false
	for k := range m {
		if strings.HasPrefix(k, "flow:") {
			delete(m, k)
			changed = true
		}
	}
	if !changed {
		return formData
	}
	b, err := json.Marshal(m)
	if err != nil {
		return formData
	}
	return string(b)
}

// ListByCreator “我的申请”列表。按创建者+租户查询实例。pageSize>0 时分页并返回真实总数。
func (r *WorkflowInstanceRepo) ListByCreator(ctx context.Context, tenantID int64, creatorUserID int64, page, pageSize int32) ([]*oaV1.MyTaskItem, int, error) {
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

	// 挂起推断：go-wind-flow 中父实例在子流程运行期间保持 PENDING（令牌停在
	// callActivity）。对仍在途的实例反查令牌表，挂有子流程的展示为 SUSPENDED，
	// 与旧引擎的状态语义对齐。
	suspended := map[int64]bool{}
	var pendingIDs []int64
	for _, e := range entities {
		if e.InstanceStatus != nil && *e.InstanceStatus == workflowinstance.InstanceStatusPending && e.ID != 0 {
			pendingIDs = append(pendingIDs, e.ID)
		}
	}
	if len(pendingIDs) > 0 {
		ids, terr := r.entClient.Client().WorkflowToken.Query().
			Where(
				workflowtoken.ChildInstanceIDNotNil(),
				workflowtoken.HasInstanceWith(workflowinstance.IDIn(pendingIDs...)),
			).
			QueryInstance().
			IDs(ctx)
		if terr != nil {
			r.log.Errorf("query suspended instances failed: %s", terr.Error())
		} else {
			for _, id := range ids {
				suspended[id] = true
			}
		}
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
		if e.ID != 0 && suspended[e.ID] {
			// 挂起（等待子流程完成）；旧引擎该状态无文本标签，保持一致。
			item.InstanceStatus = oaV1.WorkflowInstance_SUSPENDED.Enum()
			items = append(items, item)
			continue
		}
		label := instanceStatusLabel(e.InstanceStatus)
		if label != "" {
			item.StatusLabel = trans.Ptr(label)
		}
		// 枚举形式供新客户端本地化渲染（status_label 保留兼容旧客户端）
		item.InstanceStatus = r.instanceStatusConverter.ToDTO(e.InstanceStatus)
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
