package data

import (
	"context"
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
	"go-wind-oa/app/oa/service/internal/data/ent/workflowlog"
	"google.golang.org/protobuf/types/known/timestamppb"

	oav1 "go-wind-oa/api/gen/go/oa/v1"
)

// WorkflowLogRepo OA 工作流审计日志的数据访问层。append-only：仅 Create + 按操作人查询。
type WorkflowLogRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper             *mapper.CopierMapper[oav1.WorkflowLog, ent.WorkflowLog]
	logActionConverter *mapper.EnumTypeConverter[oav1.WorkflowLog_LogAction, workflowlog.LogAction]

	repository *entCrud.Repository[
		ent.WorkflowLogQuery, ent.WorkflowLogSelect,
		ent.WorkflowLogCreate, ent.WorkflowLogCreateBulk,
		ent.WorkflowLogUpdate, ent.WorkflowLogUpdateOne,
		ent.WorkflowLogDelete,
		predicate.WorkflowLog,
		oav1.WorkflowLog, ent.WorkflowLog,
	]
}

func NewWorkflowLogRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *WorkflowLogRepo {
	repo := &WorkflowLogRepo{
		log:                ctx.NewLoggerHelper("workflow-log/repo/oa-service"),
		entClient:          entClient,
		mapper:             mapper.NewCopierMapper[oav1.WorkflowLog, ent.WorkflowLog](),
		logActionConverter: mapper.NewEnumTypeConverter[oav1.WorkflowLog_LogAction, workflowlog.LogAction](oav1.WorkflowLog_LogAction_name, oav1.WorkflowLog_LogAction_value),
	}
	repo.init()
	return repo
}

func (r *WorkflowLogRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.WorkflowLogQuery, ent.WorkflowLogSelect,
		ent.WorkflowLogCreate, ent.WorkflowLogCreateBulk,
		ent.WorkflowLogUpdate, ent.WorkflowLogUpdateOne,
		ent.WorkflowLogDelete,
		predicate.WorkflowLog,
		oav1.WorkflowLog, ent.WorkflowLog,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(r.logActionConverter.NewConverterPair())
}

func (r *WorkflowLogRepo) Create(ctx context.Context, req *oav1.WorkflowLog) (*oav1.WorkflowLog, error) {
	if req == nil {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	builder := r.entClient.Client().WorkflowLog.Create().
		SetNillableTenantID(req.TenantId).
		SetNillableCreatedBy(req.CreatedBy).
		SetInstanceID(req.GetInstanceId()).
		SetNillableNodeIndex(req.NodeIndex).
		SetNillableLogAction(r.logActionConverter.ToEntity(req.LogAction)).
		SetNillableComment(req.Comment).
		SetCreatedAt(time.Now())
	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert workflow log failed: %s", err)
		return nil, oav1.ErrorInternalError("insert workflow log failed")
	}
	return r.mapper.ToDTO(entity), nil
}

// ListByActor “已办”视图：按调用人 created_by 过滤，且仅含审批类动作（APPROVE/REJECT/FORWARD）。
// 提交动作（SUBMIT）属申请人，由“我的申请”视图覆盖，此处排除。
func (r *WorkflowLogRepo) ListByActor(ctx context.Context, actorUserID uint32, req *paginationV1.PagingRequest) (*oav1.GetMyTasksResponse, error) {
	if req == nil {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	builder := r.entClient.Client().WorkflowLog.Query().
		Where(
			workflowlog.CreatedByEQ(actorUserID),
			workflowlog.LogActionIn(
				workflowlog.LogActionAPPROVE,
				workflowlog.LogActionREJECT,
				workflowlog.LogActionFORWARD,
			),
		)
	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &oav1.GetMyTasksResponse{Total: 0, Items: nil}, nil
	}
	items := make([]*oav1.MyTaskItem, 0, len(ret.Items))
	for _, lg := range ret.Items {
		items = append(items, &oav1.MyTaskItem{
			LogId:        lg.Id,
			InstanceId:   lg.InstanceId,
			ActionLabel:  ptr(lg.GetLogAction().String()),
			OccurredAt:   lg.CreatedAt,
		})
	}
	return &oav1.GetMyTasksResponse{Total: ret.Total, Items: items}, nil
}

// ListByInstance 取指定实例的审批日志轨迹，供 GetTask 服务渲染详情页的历史区。
//
// 过滤口径与 ListByActor 一致——仅含审批类动作（APPROVE / REJECT / FORWARD），
// 提交动作（SUBMIT）不在此视图。实例归属由 GetDetailByAssignee 的授权口径
// （task 指派给 caller 且 PENDING）在调用链上游保证，此处不再二次校验实例归属。
//
// instance 为 M2O 边，以 HasInstanceWith(workflowinstance.IDEQ(instanceID))
// 过滤。tenant 由 TenantPrivacy 策略按 viewer 自动隔离。
func (r *WorkflowLogRepo) ListByInstance(ctx context.Context, instanceID uint32) ([]*oav1.AuditLogEntry, error) {
	if instanceID == 0 {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	entityLogs, err := r.entClient.Client().WorkflowLog.Query().
		Where(
			workflowlog.HasInstanceWith(
				workflowinstance.IDEQ(instanceID),
			),
			workflowlog.LogActionIn(
				workflowlog.LogActionAPPROVE,
				workflowlog.LogActionREJECT,
				workflowlog.LogActionFORWARD,
			),
		).
		All(ctx)
	if err != nil {
		r.log.Errorf("query workflow log by instance failed: %s", err)
		return nil, oav1.ErrorInternalError("query workflow log by instance failed")
	}
	if len(entityLogs) == 0 {
		return nil, nil
	}
	items := make([]*oav1.AuditLogEntry, 0, len(entityLogs))
	for _, lg := range entityLogs {
		// entity 字段直读：LogAction 为 workflowlog.LogAction（string，带 .String()），
		// Comment 为 *string，CreatedAt 为 *time.Time（需转 *timestamppb.Timestamp，
		// 与 repository 注册的 TimeTimestamppbConverterPair 同款转换，此处因绕过
		// 泛型 mapper 而手写）。nil 字段跳过（proto optional 留空）。
		entry := &oav1.AuditLogEntry{
			ActionLabel: ptr(lg.LogAction.String()),
			Comment:     lg.Comment,
		}
		if lg.CreatedAt != nil {
			entry.OccurredAt = timestamppb.New(*lg.CreatedAt)
		}
		items = append(items, entry)
	}
	return items, nil
}
