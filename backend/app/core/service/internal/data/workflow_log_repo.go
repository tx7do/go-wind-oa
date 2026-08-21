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
	"go-wind-oa/app/core/service/internal/data/ent/workflowlog"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

type WorkflowLogRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper             *mapper.CopierMapper[oaV1.WorkflowLog, ent.WorkflowLog]
	logActionConverter *mapper.EnumTypeConverter[oaV1.WorkflowLog_LogAction, workflowlog.LogAction]
}

func NewWorkflowLogRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *WorkflowLogRepo {
	repo := &WorkflowLogRepo{
		log:                ctx.NewLoggerHelper("workflow-log/repo/core-service"),
		entClient:          entClient,
		mapper:             mapper.NewCopierMapper[oaV1.WorkflowLog, ent.WorkflowLog](),
		logActionConverter: mapper.NewEnumTypeConverter[oaV1.WorkflowLog_LogAction, workflowlog.LogAction](oaV1.WorkflowLog_LogAction_name, oaV1.WorkflowLog_LogAction_value),
	}

	repo.init()

	return repo
}

func (r *WorkflowLogRepo) init() {
	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(r.logActionConverter.NewConverterPair())
}

func (r *WorkflowLogRepo) Create(
	ctx context.Context,
	tenantID uint32,
	creatorUserID uint32,
	instanceID uint32,
	nodeIndex int,
	action *oaV1.WorkflowLog_LogAction,
	comment string,
) (uint32, error) {
	builder := r.entClient.Client().WorkflowLog.Create().
		SetInstanceID(instanceID).
		SetNodeIndex(nodeIndex).
		SetNillableLogAction(r.logActionConverter.ToEntity(action)).
		SetNillableComment(&comment).
		SetTenantID(tenantID).
		SetCreatedBy(creatorUserID).
		SetCreatedAt(time.Now())

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert workflow log failed: %s", err.Error())
		return 0, oaV1.ErrorInternalServerError("insert workflow log failed")
	}
	return entity.ID, nil
}

// CreateWithTx 事务内写日志。builder 源自 tx。
func (r *WorkflowLogRepo) CreateWithTx(
	ctx context.Context, tx *ent.Tx,
	tenantID uint32, creatorUserID uint32, instanceID uint32, nodeIndex int,
	action *oaV1.WorkflowLog_LogAction, comment string,
) (uint32, error) {
	builder := tx.WorkflowLog.Create().
		SetInstanceID(instanceID).
		SetNodeIndex(nodeIndex).
		SetNillableLogAction(r.logActionConverter.ToEntity(action)).
		SetNillableComment(&comment).
		SetTenantID(tenantID).
		SetCreatedBy(creatorUserID).
		SetCreatedAt(time.Now())

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert workflow log failed: %s", err.Error())
		return 0, oaV1.ErrorInternalServerError("insert workflow log failed")
	}
	return entity.ID, nil
}

// ListByActor “已办”列表。按操作者+租户查询日志，WithInstance 取实例 ID。
// pageSize>0 时分页并返回真实总数。
func (r *WorkflowLogRepo) ListByActor(ctx context.Context, tenantID uint32, actorUserID uint32, page, pageSize int32) ([]*oaV1.MyTaskItem, int, error) {
	query := r.entClient.Client().WorkflowLog.Query().
		Where(
			workflowlog.CreatedByEQ(actorUserID),
			workflowlog.TenantIDEQ(tenantID),
		)
	total, err := query.Clone().Count(ctx)
	if err != nil {
		r.log.Errorf("count logs by actor failed: %s", err.Error())
		return nil, 0, oaV1.ErrorInternalServerError("count logs failed")
	}
	if pageSize > 0 {
		if page < 1 {
			page = 1
		}
		query = query.Offset((int(page) - 1) * int(pageSize)).Limit(int(pageSize))
	}
	entities, err := query.Order(ent.Desc(workflowlog.FieldID)).WithInstance().All(ctx)
	if err != nil {
		r.log.Errorf("list logs by actor failed: %s", err.Error())
		return nil, 0, oaV1.ErrorInternalServerError("list logs failed")
	}

	items := make([]*oaV1.MyTaskItem, 0, len(entities))
	for _, e := range entities {
		inst := e.Edges.Instance
		if inst == nil {
			continue
		}
		item := &oaV1.MyTaskItem{}
		item.InstanceId = trans.Ptr(inst.ID)
		if e.CreatedAt != nil {
			item.CreatedAt = timeutil.TimeToTimestamppb(e.CreatedAt)
		}
		label := logActionLabel(e.LogAction)
		if label != "" {
			item.StatusLabel = trans.Ptr(label)
		}
		items = append(items, item)
	}
	return items, total, nil
}

// ListByInstance 审批历史。经 instance→logs 边遍历（WithLogs）取该实例的全部日志。
func (r *WorkflowLogRepo) ListByInstance(ctx context.Context, tenantID uint32, instanceID uint32) ([]*oaV1.WorkflowLog, error) {
	instance, err := r.entClient.Client().WorkflowInstance.Query().
		Where(
			workflowinstance.IDEQ(instanceID),
			workflowinstance.TenantIDEQ(tenantID),
		).
		WithLogs().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, oaV1.ErrorNotFound("workflow instance not found")
		}
		r.log.Errorf("query instance for logs failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query instance failed")
	}

	logs := instance.Edges.Logs
	if logs == nil {
		return []*oaV1.WorkflowLog{}, nil
	}
	items := make([]*oaV1.WorkflowLog, 0, len(logs))
	for _, lg := range logs {
		items = append(items, r.mapper.ToDTO(lg))
	}
	return items, nil
}

func logActionLabel(a *workflowlog.LogAction) string {
	if a == nil {
		return ""
	}
	switch *a {
	case workflowlog.LogActionSubmit:
		return "已提交"
	case workflowlog.LogActionApprove:
		return "已审批通过"
	case workflowlog.LogActionReject:
		return "已审批驳回"
	case workflowlog.LogActionForward:
		return "已转办"
	case workflowlog.LogActionWithdraw:
		return "已撤回"
	}
	return ""
}
