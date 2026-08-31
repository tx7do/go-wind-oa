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
	"go-wind-oa/app/core/service/internal/data/ent/predicate"
	"go-wind-oa/app/core/service/internal/data/ent/workflowinstance"
	"go-wind-oa/app/core/service/internal/data/ent/workflowtask"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

type WorkflowTaskRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper              *mapper.CopierMapper[oaV1.WorkflowTask, ent.WorkflowTask]
	taskStatusConverter *mapper.EnumTypeConverter[oaV1.WorkflowTask_TaskStatus, workflowtask.TaskStatus]
}

func NewWorkflowTaskRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *WorkflowTaskRepo {
	repo := &WorkflowTaskRepo{
		log:                 ctx.NewLoggerHelper("workflow-task/repo/core-service"),
		entClient:           entClient,
		mapper:              mapper.NewCopierMapper[oaV1.WorkflowTask, ent.WorkflowTask](),
		taskStatusConverter: mapper.NewEnumTypeConverter[oaV1.WorkflowTask_TaskStatus, workflowtask.TaskStatus](oaV1.WorkflowTask_TaskStatus_name, oaV1.WorkflowTask_TaskStatus_value),
	}

	repo.init()

	return repo
}

func (r *WorkflowTaskRepo) init() {
	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(r.taskStatusConverter.NewConverterPair())
}

func (r *WorkflowTaskRepo) Create(
	ctx context.Context,
	tenantID uint32,
	creatorUserID uint32,
	instanceID uint32,
	nodeID string,
	assigneeUserID uint32,
) (uint32, error) {
	builder := r.entClient.Client().WorkflowTask.Create().
		SetInstanceID(instanceID).
		SetNodeID(nodeID).
		SetAssigneeUserID(assigneeUserID).
		SetTaskStatus(workflowtask.TaskStatusPending).
		SetTenantID(tenantID).
		SetCreatedBy(creatorUserID).
		SetCreatedAt(time.Now())

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert workflow task failed: %s", err.Error())
		return 0, oaV1.ErrorInternalServerError("insert workflow task failed")
	}
	return entity.ID, nil
}

// GetState 直读 entity 的 TaskStatus/AssigneeUserID/NodeID 字段（绕过 mapper），并经
// WithInstance 边加载读取父实例的 ID。
// 返回值供状态机校验：assignee==caller && taskPending && instance 活跃。
func (r *WorkflowTaskRepo) GetState(ctx context.Context, id uint32, tenantID uint32) (uint32, bool, uint32, string, error) {
	entity, err := r.entClient.Client().WorkflowTask.Query().
		Where(
			workflowtask.IDEQ(id),
			workflowtask.TenantIDEQ(tenantID),
		).
		WithInstance().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, false, 0, "", oaV1.ErrorNotFound("workflow task not found")
		}
		r.log.Errorf("query task state failed: %s", err.Error())
		return 0, false, 0, "", oaV1.ErrorInternalServerError("query task state failed")
	}
	if entity.AssigneeUserID == nil || entity.TaskStatus == nil || entity.NodeID == nil {
		return 0, false, 0, "", oaV1.ErrorConflict("task state corrupt")
	}
	inst := entity.Edges.Instance
	if inst == nil {
		return 0, false, 0, "", oaV1.ErrorConflict("instance edge missing")
	}
	assignee := *entity.AssigneeUserID
	taskPending := *entity.TaskStatus == workflowtask.TaskStatusPending
	return assignee, taskPending, inst.ID, *entity.NodeID, nil
}

func (r *WorkflowTaskRepo) UpdateStatus(
	ctx context.Context,
	id uint32,
	tenantID uint32,
	newStatus *oaV1.WorkflowTask_TaskStatus,
) error {
	builder := r.entClient.Client().WorkflowTask.Update()
	builder.Where(
		workflowtask.IDEQ(id),
		workflowtask.TenantIDEQ(tenantID),
	)
	builder.SetNillableTaskStatus(r.taskStatusConverter.ToEntity(newStatus))
	builder.SetUpdatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("update task status failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("update task status failed")
	}
	return nil
}

func (r *WorkflowTaskRepo) UpdateAssignee(
	ctx context.Context,
	id uint32,
	tenantID uint32,
	newAssignee uint32,
) error {
	builder := r.entClient.Client().WorkflowTask.Update()
	builder.Where(
		workflowtask.IDEQ(id),
		workflowtask.TenantIDEQ(tenantID),
	)
	builder.SetAssigneeUserID(newAssignee)
	builder.SetUpdatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("update task assignee failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("update task assignee failed")
	}
	return nil
}

// MarkReminded 记录超时催办时间；非空后调度器不再重复催办同一待办。
func (r *WorkflowTaskRepo) MarkReminded(
	ctx context.Context,
	id uint32,
	tenantID uint32,
) error {
	return r.markTimeoutStamp(ctx, id, tenantID, true)
}

// MarkEscalated 记录超时升级时间；非空后调度器不再对同一待办重复升级。
func (r *WorkflowTaskRepo) MarkEscalated(
	ctx context.Context,
	id uint32,
	tenantID uint32,
) error {
	return r.markTimeoutStamp(ctx, id, tenantID, false)
}

func (r *WorkflowTaskRepo) markTimeoutStamp(
	ctx context.Context,
	id uint32,
	tenantID uint32,
	reminded bool,
) error {
	builder := r.entClient.Client().WorkflowTask.Update()
	builder.Where(
		workflowtask.IDEQ(id),
		workflowtask.TenantIDEQ(tenantID),
	)
	if reminded {
		builder.SetRemindedAt(time.Now())
	} else {
		builder.SetEscalatedAt(time.Now())
	}
	builder.SetUpdatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("mark timeout stamp failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("mark timeout stamp failed")
	}
	return nil
}

// ListPendingByAssignee “待办”列表。三重谓词：指派审批人 + 任务状态=PENDING + 租户。
// pageSize>0 时分页并返回真实总数。
func (r *WorkflowTaskRepo) ListPendingByAssignee(ctx context.Context, tenantID uint32, assigneeUserID uint32, page, pageSize int32) ([]*oaV1.MyTaskItem, int, error) {
	query := r.entClient.Client().WorkflowTask.Query().
		Where(
			workflowtask.AssigneeUserIDEQ(assigneeUserID),
			workflowtask.TaskStatusEQ(workflowtask.TaskStatusPending),
			workflowtask.TenantIDEQ(tenantID),
		)
	total, err := query.Clone().Count(ctx)
	if err != nil {
		r.log.Errorf("count pending tasks failed: %s", err.Error())
		return nil, 0, oaV1.ErrorInternalServerError("count pending tasks failed")
	}
	if pageSize > 0 {
		if page < 1 {
			page = 1
		}
		query = query.Offset((int(page) - 1) * int(pageSize)).Limit(int(pageSize))
	}
	entities, err := query.Order(ent.Desc(workflowtask.FieldID)).WithInstance().All(ctx)
	if err != nil {
		r.log.Errorf("list pending tasks failed: %s", err.Error())
		return nil, 0, oaV1.ErrorInternalServerError("list pending tasks failed")
	}

	items := make([]*oaV1.MyTaskItem, 0, len(entities))
	for _, e := range entities {
		inst := e.Edges.Instance
		if inst == nil {
			continue
		}
		item := &oaV1.MyTaskItem{}
		item.TaskId = trans.Ptr(e.ID)
		item.InstanceId = trans.Ptr(inst.ID)
		item.StatusLabel = trans.Ptr("待办")
		if e.CreatedAt != nil {
			item.CreatedAt = timeutil.TimeToTimestamppb(e.CreatedAt)
		}
		items = append(items, item)
	}
	return items, total, nil
}

// GetDetailByAssignee 任务详情。三重谓词 + IDEQ。返回 task DTO（经 mapper）+ 父实例 ID
// （从 WithInstance 边加载取，供服务层查询审批历史）。task DTO 本身不暴露 instance_id 字段。
func (r *WorkflowTaskRepo) GetDetailByAssignee(ctx context.Context, id uint32, tenantID uint32, assigneeUserID uint32) (*oaV1.WorkflowTask, uint32, error) {
	entity, err := r.entClient.Client().WorkflowTask.Query().
		Where(
			workflowtask.IDEQ(id),
			workflowtask.AssigneeUserIDEQ(assigneeUserID),
			workflowtask.TaskStatusEQ(workflowtask.TaskStatusPending),
			workflowtask.TenantIDEQ(tenantID),
		).
		WithInstance().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, 0, oaV1.ErrorNotFound("workflow task not found")
		}
		r.log.Errorf("query task detail failed: %s", err.Error())
		return nil, 0, oaV1.ErrorInternalServerError("query task detail failed")
	}

	inst := entity.Edges.Instance
	if inst == nil {
		return nil, 0, oaV1.ErrorConflict("instance edge missing")
	}
	return r.mapper.ToDTO(entity), inst.ID, nil
}

// ListNodeTaskStatuses 读取同实例同节点的全部任务状态（含已终结的），供会签/或签的
// 计数判定：会签看是否仍有 PENDING，或签看是否全部终结且无 APPROVED。
// instance_id 列由边持有（无独立谓词），经 HasInstanceWith 过滤。
func (r *WorkflowTaskRepo) ListNodeTaskStatuses(
	ctx context.Context,
	tenantID uint32,
	instanceID uint32,
	nodeID string,
) ([]workflowtask.TaskStatus, error) {
	entities, err := r.entClient.Client().WorkflowTask.Query().
		Where(
			workflowtask.HasInstanceWith(
				workflowinstance.IDEQ(instanceID),
				workflowinstance.TenantIDEQ(tenantID),
			),
			workflowtask.NodeIDEQ(nodeID),
			workflowtask.TenantIDEQ(tenantID),
		).
		All(ctx)
	if err != nil {
		r.log.Errorf("list node task statuses failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("list node task statuses failed")
	}
	statuses := make([]workflowtask.TaskStatus, 0, len(entities))
	for _, e := range entities {
		if e.TaskStatus != nil {
			statuses = append(statuses, *e.TaskStatus)
		}
	}
	return statuses, nil
}

// CancelPendingByInstanceNode 取消同实例同节点的 PENDING 任务（或签一人通过后取消
// 其余、会签驳回后取消其余、撤回时取消全部）。excludeTaskID==0 表示不排除任何任务。
func (r *WorkflowTaskRepo) CancelPendingByInstanceNode(
	ctx context.Context,
	tenantID uint32,
	instanceID uint32,
	nodeID string,
	excludeTaskID uint32,
) error {
	predicates := []predicate.WorkflowTask{
		workflowtask.HasInstanceWith(workflowinstance.IDEQ(instanceID)),
		workflowtask.NodeIDEQ(nodeID),
		workflowtask.TaskStatusEQ(workflowtask.TaskStatusPending),
		workflowtask.TenantIDEQ(tenantID),
	}
	if excludeTaskID != 0 {
		predicates = append(predicates, workflowtask.IDNEQ(excludeTaskID))
	}

	builder := r.entClient.Client().WorkflowTask.Update()
	builder.Where(predicates...)
	builder.SetTaskStatus(workflowtask.TaskStatusCancelled)
	builder.SetUpdatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("cancel pending tasks failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("cancel pending tasks failed")
	}
	return nil
}

// CreateWithTx 事务内建任务。builder 源自 tx。
func (r *WorkflowTaskRepo) CreateWithTx(
	ctx context.Context, tx *ent.Tx,
	tenantID uint32, creatorUserID uint32, instanceID uint32, nodeID string, assigneeUserID uint32,
) (uint32, error) {
	builder := tx.WorkflowTask.Create().
		SetInstanceID(instanceID).
		SetNodeID(nodeID).
		SetAssigneeUserID(assigneeUserID).
		SetTaskStatus(workflowtask.TaskStatusPending).
		SetTenantID(tenantID).
		SetCreatedBy(creatorUserID).
		SetCreatedAt(time.Now())

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert workflow task failed: %s", err.Error())
		return 0, oaV1.ErrorInternalServerError("insert workflow task failed")
	}
	return entity.ID, nil
}

// UpdateStatusWithTx 事务内更新任务状态。
func (r *WorkflowTaskRepo) UpdateStatusWithTx(
	ctx context.Context, tx *ent.Tx, id uint32, tenantID uint32, newStatus *oaV1.WorkflowTask_TaskStatus,
) error {
	builder := tx.WorkflowTask.Update()
	builder.Where(
		workflowtask.IDEQ(id),
		workflowtask.TenantIDEQ(tenantID),
	)
	builder.SetNillableTaskStatus(r.taskStatusConverter.ToEntity(newStatus))
	builder.SetUpdatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("update task status failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("update task status failed")
	}
	return nil
}

// UpdateAssigneeWithTx 事务内更新任务指派人（转办）。
func (r *WorkflowTaskRepo) UpdateAssigneeWithTx(
	ctx context.Context, tx *ent.Tx, id uint32, tenantID uint32, newAssignee uint32,
) error {
	builder := tx.WorkflowTask.Update()
	builder.Where(
		workflowtask.IDEQ(id),
		workflowtask.TenantIDEQ(tenantID),
	)
	builder.SetAssigneeUserID(newAssignee)
	builder.SetUpdatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("update task assignee failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("update task assignee failed")
	}
	return nil
}

// CancelPendingByInstanceNodeWithTx 事务内取消同实例同节点 PENDING 任务。
func (r *WorkflowTaskRepo) CancelPendingByInstanceNodeWithTx(
	ctx context.Context, tx *ent.Tx, tenantID uint32, instanceID uint32, nodeID string, excludeTaskID uint32,
) error {
	predicates := []predicate.WorkflowTask{
		workflowtask.HasInstanceWith(workflowinstance.IDEQ(instanceID)),
		workflowtask.NodeIDEQ(nodeID),
		workflowtask.TaskStatusEQ(workflowtask.TaskStatusPending),
		workflowtask.TenantIDEQ(tenantID),
	}
	if excludeTaskID != 0 {
		predicates = append(predicates, workflowtask.IDNEQ(excludeTaskID))
	}

	builder := tx.WorkflowTask.Update()
	builder.Where(predicates...)
	builder.SetTaskStatus(workflowtask.TaskStatusCancelled)
	builder.SetUpdatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("cancel pending tasks failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("cancel pending tasks failed")
	}
	return nil
}
func (r *WorkflowTaskRepo) ListPendingAssigneesByInstance(
	ctx context.Context,
	tenantID uint32,
	instanceID uint32,
) ([]uint32, error) {
	entities, err := r.entClient.Client().WorkflowTask.Query().
		Where(
			workflowtask.HasInstanceWith(workflowinstance.IDEQ(instanceID)),
			workflowtask.TaskStatusEQ(workflowtask.TaskStatusPending),
			workflowtask.TenantIDEQ(tenantID),
		).
		All(ctx)
	if err != nil {
		r.log.Errorf("list pending assignees failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("list pending assignees failed")
	}
	assignees := make([]uint32, 0, len(entities))
	for _, e := range entities {
		if e.AssigneeUserID != nil && *e.AssigneeUserID != 0 {
			assignees = append(assignees, *e.AssigneeUserID)
		}
	}
	return assignees, nil
}

// HasPendingByInstance 判断实例是否存在 PENDING 任务（用于活跃节点判定）。
func (r *WorkflowTaskRepo) HasPendingByInstance(
	ctx context.Context,
	tenantID uint32,
	instanceID uint32,
) (bool, error) {
	count, err := r.entClient.Client().WorkflowTask.Query().
		Where(
			workflowtask.HasInstanceWith(
				workflowinstance.IDEQ(instanceID),
				workflowinstance.TenantIDEQ(tenantID),
			),
			workflowtask.TaskStatusEQ(workflowtask.TaskStatusPending),
			workflowtask.TenantIDEQ(tenantID),
		).
		Count(ctx)
	if err != nil {
		r.log.Errorf("count pending tasks by instance failed: %s", err.Error())
		return false, oaV1.ErrorInternalServerError("count pending tasks failed")
	}
	return count > 0, nil
}

// PendingTaskInfo 待办任务的超时检查信息。
type PendingTaskInfo struct {
	TaskID      uint32
	InstanceID  uint32
	TenantID    uint32
	Assignee    uint32
	NodeID      string
	CreatedAt   time.Time
	RemindedAt  *time.Time
	EscalatedAt *time.Time
}

// ListAllPendingWithAge 列出全部租户的 PENDING 任务及其审批人、实例、节点、创建时间。
// 供超时催办调度器扫描用。经 WithInstance 边加载取实例 ID。
func (r *WorkflowTaskRepo) ListAllPendingWithAge(
	ctx context.Context,
) ([]PendingTaskInfo, error) {
	entities, err := r.entClient.Client().WorkflowTask.Query().
		Where(
			workflowtask.TaskStatusEQ(workflowtask.TaskStatusPending),
		).
		WithInstance().
		All(ctx)
	if err != nil {
		r.log.Errorf("list all pending with age failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("list all pending with age failed")
	}
	tasks := make([]PendingTaskInfo, 0, len(entities))
	for _, e := range entities {
		if e.AssigneeUserID == nil || *e.AssigneeUserID == 0 {
			continue
		}
		if e.NodeID == nil || *e.NodeID == "" {
			continue
		}
		inst := e.Edges.Instance
		if inst == nil {
			continue
		}
		createdAt := time.Time{}
		if e.CreatedAt != nil {
			createdAt = *e.CreatedAt
		}
		tasks = append(tasks, PendingTaskInfo{
			TaskID:      e.ID,
			InstanceID:  inst.ID,
			TenantID:    *e.TenantID,
			Assignee:    *e.AssigneeUserID,
			NodeID:      *e.NodeID,
			CreatedAt:   createdAt,
			RemindedAt:  e.RemindedAt,
			EscalatedAt: e.EscalatedAt,
		})
	}
	return tasks, nil
}

// CancelAllPendingByInstance 取消实例全部待办任务（撤回场景，跨所有节点）。
func (r *WorkflowTaskRepo) CancelAllPendingByInstance(
	ctx context.Context,
	tenantID uint32,
	instanceID uint32,
) error {
	builder := r.entClient.Client().WorkflowTask.Update()
	builder.Where(
		workflowtask.HasInstanceWith(workflowinstance.IDEQ(instanceID)),
		workflowtask.TaskStatusEQ(workflowtask.TaskStatusPending),
		workflowtask.TenantIDEQ(tenantID),
	)
	builder.SetTaskStatus(workflowtask.TaskStatusCancelled)
	builder.SetUpdatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("cancel all pending tasks failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("cancel all pending tasks failed")
	}
	return nil
}

// CancelAllPendingByInstanceWithTx 事务内取消实例全部待办任务。
func (r *WorkflowTaskRepo) CancelAllPendingByInstanceWithTx(
	ctx context.Context, tx *ent.Tx, tenantID uint32, instanceID uint32,
) error {
	builder := tx.WorkflowTask.Update()
	builder.Where(
		workflowtask.HasInstanceWith(
			workflowinstance.IDEQ(instanceID),
			workflowinstance.TenantIDEQ(tenantID),
		),
		workflowtask.TaskStatusEQ(workflowtask.TaskStatusPending),
		workflowtask.TenantIDEQ(tenantID),
	)
	builder.SetTaskStatus(workflowtask.TaskStatusCancelled)
	builder.SetUpdatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("cancel all pending tasks failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("cancel all pending tasks failed")
	}
	return nil
}

// CancelAllByInstanceNode 清除指定节点的全部任务（不限状态，置 CANCELLED）。
// 用于回退边场景：walker 再次到达某节点时，该节点可能残留上一轮的 APPROVED/REJECTED 任务，
// 会污染会签/或签收敛判定，故建新任务前先清除。
func (r *WorkflowTaskRepo) CancelAllByInstanceNode(
	ctx context.Context,
	tenantID uint32,
	instanceID uint32,
	nodeID string,
) error {
	builder := r.entClient.Client().WorkflowTask.Update()
	builder.Where(
		workflowtask.HasInstanceWith(
			workflowinstance.IDEQ(instanceID),
			workflowinstance.TenantIDEQ(tenantID),
		),
		workflowtask.NodeIDEQ(nodeID),
		workflowtask.TenantIDEQ(tenantID),
	)
	builder.SetTaskStatus(workflowtask.TaskStatusCancelled)
	builder.SetUpdatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("cancel all by instance node failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("cancel all by instance node failed")
	}
	return nil
}

// CancelAllByInstanceNodeWithTx 事务内清除指定节点的全部任务。
func (r *WorkflowTaskRepo) CancelAllByInstanceNodeWithTx(
	ctx context.Context, tx *ent.Tx, tenantID uint32, instanceID uint32, nodeID string,
) error {
	builder := tx.WorkflowTask.Update()
	builder.Where(
		workflowtask.HasInstanceWith(
			workflowinstance.IDEQ(instanceID),
			workflowinstance.TenantIDEQ(tenantID),
		),
		workflowtask.NodeIDEQ(nodeID),
		workflowtask.TenantIDEQ(tenantID),
	)
	builder.SetTaskStatus(workflowtask.TaskStatusCancelled)
	builder.SetUpdatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("cancel all by instance node failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("cancel all by instance node failed")
	}
	return nil
}
