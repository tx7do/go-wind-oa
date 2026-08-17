package data

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-oa/app/oa/service/internal/data/ent"
	"go-wind-oa/app/oa/service/internal/data/ent/predicate"
	"go-wind-oa/app/oa/service/internal/data/ent/workflowinstance"

	oav1 "go-wind-oa/api/gen/go/oa/v1"
)

// WorkflowInstanceRepo OA 工作流实例的数据访问层。
type WorkflowInstanceRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper                   *mapper.CopierMapper[oav1.WorkflowInstance, ent.WorkflowInstance]
	instanceStatusConverter *mapper.EnumTypeConverter[oav1.WorkflowInstance_InstanceStatus, workflowinstance.InstanceStatus]

	repository *entCrud.Repository[
		ent.WorkflowInstanceQuery, ent.WorkflowInstanceSelect,
		ent.WorkflowInstanceCreate, ent.WorkflowInstanceCreateBulk,
		ent.WorkflowInstanceUpdate, ent.WorkflowInstanceUpdateOne,
		ent.WorkflowInstanceDelete,
		predicate.WorkflowInstance,
		oav1.WorkflowInstance, ent.WorkflowInstance,
	]
}

func NewWorkflowInstanceRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *WorkflowInstanceRepo {
	repo := &WorkflowInstanceRepo{
		log:                     ctx.NewLoggerHelper("workflow-instance/repo/oa-service"),
		entClient:               entClient,
		mapper:                  mapper.NewCopierMapper[oav1.WorkflowInstance, ent.WorkflowInstance](),
		instanceStatusConverter: mapper.NewEnumTypeConverter[oav1.WorkflowInstance_InstanceStatus, workflowinstance.InstanceStatus](oav1.WorkflowInstance_InstanceStatus_name, oav1.WorkflowInstance_InstanceStatus_value),
	}
	repo.init()
	return repo
}

func (r *WorkflowInstanceRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.WorkflowInstanceQuery, ent.WorkflowInstanceSelect,
		ent.WorkflowInstanceCreate, ent.WorkflowInstanceCreateBulk,
		ent.WorkflowInstanceUpdate, ent.WorkflowInstanceUpdateOne,
		ent.WorkflowInstanceDelete,
		predicate.WorkflowInstance,
		oav1.WorkflowInstance, ent.WorkflowInstance,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(r.instanceStatusConverter.NewConverterPair())
}

// GetState 直接读取实例的运行时状态，供状态机推进使用。
//
// 不走 mapper / DTO：status 以 0=活跃(PENDING) / 1=非活的整数返回，避免
// any/指针字段在 mapper 中的不确定性。tenant 由 TenantPrivacy 策略按 viewer 隔离。
func (r *WorkflowInstanceRepo) GetState(ctx context.Context, instanceID uint32) (int, int32, uint32, error) {
	if instanceID == 0 {
		return 0, 0, 0, oav1.ErrorBadRequest("invalid parameter")
	}
	entity, err := r.entClient.Client().WorkflowInstance.Query().
		Where(workflowinstance.IDEQ(instanceID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, 0, 0, oav1.ErrorNotFound("workflow instance not found")
		}
		r.log.Errorf("query workflow instance state failed: %s", err)
		return 0, 0, 0, oav1.ErrorInternalError("query workflow instance state failed")
	}
	status := 1
	if entity.InstanceStatus == workflowinstance.InstanceStatusPENDING {
		status = 0
	}
	// CurrentNodeIndex / CreatedBy 为 *int32 / *uint32（Optional+Nilable）；
	// nil 时按 0 返回，调用方在状态机中据此判断“无下一节点 / 未知申请人”。
	var nodeIdx int32
	if entity.CurrentNodeIndex != nil {
		nodeIdx = *entity.CurrentNodeIndex
	}
	var creator uint32
	if entity.CreatedBy != nil {
		creator = *entity.CreatedBy
	}
	return status, nodeIdx, creator, nil
}

// GetDefinitionNodeConfig 经实例的 definition 边取其定义的节点配置（any → JSON 文本）。
// 状态机审批通过、推进下一节点时调用。tenant 隔离对边遍历到的定义查询同样生效。
func (r *WorkflowInstanceRepo) GetDefinitionNodeConfig(ctx context.Context, instanceID uint32) (string, error) {
	if instanceID == 0 {
		return "", oav1.ErrorBadRequest("invalid parameter")
	}
	entity, err := r.entClient.Client().WorkflowInstance.Query().
		Where(workflowinstance.IDEQ(instanceID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", oav1.ErrorNotFound("workflow instance not found")
		}
		r.log.Errorf("query workflow instance for definition failed: %s", err)
		return "", oav1.ErrorInternalError("query workflow instance for definition failed")
	}
	defEntity, err := entity.QueryDefinition().Only(ctx)
	if err != nil {
		r.log.Errorf("query definition via instance edge failed: %s", err)
		return "", oav1.ErrorInternalError("query definition via instance edge failed")
	}
	var nodeConfig string
	if v := defEntity.NodeConfig; v != nil {
		if b, mErr := json.Marshal(v); mErr == nil {
			p := string(b)
			nodeConfig = p
		}
	}
	return nodeConfig, nil
}

func (r *WorkflowInstanceRepo) Create(ctx context.Context, req *oav1.WorkflowInstance) (*oav1.WorkflowInstance, error) {
	if req == nil {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	builder := r.entClient.Client().WorkflowInstance.Create().
		SetNillableTenantID(req.TenantId).
		SetNillableCreatedBy(req.CreatedBy).
		SetNillableTitle(req.Title).
		SetNillableInstanceStatus(r.instanceStatusConverter.ToEntity(req.InstanceStatus)).
		SetNillableCurrentNodeIndex(req.CurrentNodeIndex).
		SetDefinitionID(req.GetDefinitionId()).
		SetCreatedAt(time.Now())

	// 申请表单数据：DTO JSON 文本 → entity any。
	if req.FormData != nil {
		var v any
		if err := json.Unmarshal([]byte(*req.FormData), &v); err == nil {
			builder.SetFormData(v)
		}
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert workflow instance failed: %s", err)
		return nil, oav1.ErrorInternalError("insert workflow instance failed")
	}
	return r.mapper.ToDTO(entity), nil
}

// UpdateStatus 状态机推进：写实例状态 + 当前节点指针。currentNodeIndex 为 nil 时清空指针（终结态）。
func (r *WorkflowInstanceRepo) UpdateStatus(ctx context.Context, id uint32, status oav1.WorkflowInstance_InstanceStatus, currentNodeIndex *int32) error {
	builder := r.entClient.Client().WorkflowInstance.UpdateOneID(id).
		SetNillableInstanceStatus(r.instanceStatusConverter.ToEntity(&status)).
		SetNillableCurrentNodeIndex(currentNodeIndex).
		SetUpdatedAt(time.Now())
	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf("update workflow instance status failed: %s", err)
		return oav1.ErrorInternalError("update workflow instance status failed")
	}
	return nil
}

// ListByCreator “我的申请”视图：按调用人 created_by 过滤，tenant 由策略自动隔离。
func (r *WorkflowInstanceRepo) ListByCreator(ctx context.Context, creatorUserID uint32, req *paginationV1.PagingRequest) (*oav1.GetMyTasksResponse, error) {
	if req == nil {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	builder := r.entClient.Client().WorkflowInstance.Query().
		Where(workflowinstance.CreatedByEQ(creatorUserID))
	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &oav1.GetMyTasksResponse{Total: 0, Items: nil}, nil
	}
	items := make([]*oav1.MyTaskItem, 0, len(ret.Items))
	for _, inst := range ret.Items {
		items = append(items, &oav1.MyTaskItem{
			InstanceId:  inst.Id,
			Title:       inst.Title,
			StatusLabel: ptr(inst.GetInstanceStatus().String()),
			OccurredAt:  inst.CreatedAt,
		})
	}
	return &oav1.GetMyTasksResponse{Total: ret.Total, Items: items}, nil
}
