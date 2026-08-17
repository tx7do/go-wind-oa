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
	"go-wind-oa/app/oa/service/internal/data/ent/workflowtask"

	oav1 "go-wind-oa/api/gen/go/oa/v1"
)

// WorkflowTaskRepo OA 工作流任务的数据访问层。
type WorkflowTaskRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper              *mapper.CopierMapper[oav1.WorkflowTask, ent.WorkflowTask]
	taskStatusConverter *mapper.EnumTypeConverter[oav1.WorkflowTask_TaskStatus, workflowtask.TaskStatus]

	repository *entCrud.Repository[
		ent.WorkflowTaskQuery, ent.WorkflowTaskSelect,
		ent.WorkflowTaskCreate, ent.WorkflowTaskCreateBulk,
		ent.WorkflowTaskUpdate, ent.WorkflowTaskUpdateOne,
		ent.WorkflowTaskDelete,
		predicate.WorkflowTask,
		oav1.WorkflowTask, ent.WorkflowTask,
	]
}

func NewWorkflowTaskRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *WorkflowTaskRepo {
	repo := &WorkflowTaskRepo{
		log:                 ctx.NewLoggerHelper("workflow-task/repo/oa-service"),
		entClient:           entClient,
		mapper:              mapper.NewCopierMapper[oav1.WorkflowTask, ent.WorkflowTask](),
		taskStatusConverter: mapper.NewEnumTypeConverter[oav1.WorkflowTask_TaskStatus, workflowtask.TaskStatus](oav1.WorkflowTask_TaskStatus_name, oav1.WorkflowTask_TaskStatus_value),
	}
	repo.init()
	return repo
}

func (r *WorkflowTaskRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.WorkflowTaskQuery, ent.WorkflowTaskSelect,
		ent.WorkflowTaskCreate, ent.WorkflowTaskCreateBulk,
		ent.WorkflowTaskUpdate, ent.WorkflowTaskUpdateOne,
		ent.WorkflowTaskDelete,
		predicate.WorkflowTask,
		oav1.WorkflowTask, ent.WorkflowTask,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(r.taskStatusConverter.NewConverterPair())
}

// GetState 直接读取任务的运行时状态，供状态机审批决策使用。
// status 以 0=待办(PENDING) / 1=非待办整数返回。
func (r *WorkflowTaskRepo) GetState(ctx context.Context, taskID uint32) (uint32, int, uint32, int32, error) {
	if taskID == 0 {
		return 0, 0, 0, 0, oav1.ErrorBadRequest("invalid parameter")
	}
	entity, err := r.entClient.Client().WorkflowTask.Query().
		Where(workflowtask.IDEQ(taskID)).
		WithInstance().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, 0, 0, 0, oav1.ErrorNotFound("workflow task not found")
		}
		r.log.Errorf("query workflow task state failed: %s", err)
		return 0, 0, 0, 0, oav1.ErrorInternalError("query workflow task state failed")
	}
	status := 1
	if entity.TaskStatus == workflowtask.TaskStatusPENDING {
		status = 0
	}
	// instance 外键经 edge 显式加载后取其 ID；AssigneeUserID / NodeIndex 为
	// *uint32 / *int32（Optional+Nilable），nil 时按 0 返回。
	var instanceID uint32
	if entity.Edges.Instance != nil {
		instanceID = entity.Edges.Instance.ID
	}
	var assignee uint32
	if entity.AssigneeUserID != nil {
		assignee = *entity.AssigneeUserID
	}
	var nodeIdx int32
	if entity.NodeIndex != nil {
		nodeIdx = *entity.NodeIndex
	}
	return assignee, status, instanceID, nodeIdx, nil
}

func (r *WorkflowTaskRepo) Create(ctx context.Context, req *oav1.WorkflowTask) (*oav1.WorkflowTask, error) {
	if req == nil {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	builder := r.entClient.Client().WorkflowTask.Create().
		SetNillableTenantID(req.TenantId).
		SetNillableCreatedBy(req.CreatedBy).
		SetInstanceID(req.GetInstanceId()).
		SetNillableNodeIndex(req.NodeIndex).
		SetNillableAssigneeUserID(req.AssigneeUserId).
		SetNillableTaskStatus(r.taskStatusConverter.ToEntity(req.TaskStatus)).
		SetCreatedAt(time.Now())
	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert workflow task failed: %s", err)
		return nil, oav1.ErrorInternalError("insert workflow task failed")
	}
	return r.mapper.ToDTO(entity), nil
}

// UpdateStatus 关闭任务（审批通过 / 驳回）。
func (r *WorkflowTaskRepo) UpdateStatus(ctx context.Context, id uint32, status oav1.WorkflowTask_TaskStatus) error {
	builder := r.entClient.Client().WorkflowTask.UpdateOneID(id).
		SetNillableTaskStatus(r.taskStatusConverter.ToEntity(&status)).
		SetUpdatedAt(time.Now())
	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf("update workflow task status failed: %s", err)
		return oav1.ErrorInternalError("update workflow task status failed")
	}
	return nil
}

// UpdateAssignee 转办：改写当前任务的指派审批人，任务状态保持 PENDING。
func (r *WorkflowTaskRepo) UpdateAssignee(ctx context.Context, id uint32, assigneeUserId uint32) error {
	builder := r.entClient.Client().WorkflowTask.UpdateOneID(id).
		SetNillableAssigneeUserID(&assigneeUserId).
		SetUpdatedAt(time.Now())
	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf("update workflow task assignee failed: %s", err)
		return oav1.ErrorInternalError("update workflow task assignee failed")
	}
	return nil
}

// ListPendingByAssignee “待办”视图：按 (assignee_user_id, task_status=PENDING) 过滤。
// tenant 由 TenantPrivacy 策略按 viewer 自动隔离。
func (r *WorkflowTaskRepo) ListPendingByAssignee(ctx context.Context, assigneeUserId uint32, req *paginationV1.PagingRequest) (*oav1.GetMyTasksResponse, error) {
	if req == nil {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	builder := r.entClient.Client().WorkflowTask.Query().
		Where(
			workflowtask.AssigneeUserIDEQ(assigneeUserId),
			workflowtask.TaskStatusEQ(workflowtask.TaskStatusPENDING),
		)
	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &oav1.GetMyTasksResponse{Total: 0, Items: nil}, nil
	}
	items := make([]*oav1.MyTaskItem, 0, len(ret.Items))
	for _, task := range ret.Items {
		items = append(items, &oav1.MyTaskItem{
			TaskId:      task.Id,
			InstanceId:  task.InstanceId,
			StatusLabel: ptr(task.GetTaskStatus().String()),
			OccurredAt:  task.CreatedAt,
		})
	}
	return &oav1.GetMyTasksResponse{Total: ret.Total, Items: items}, nil
}

// GetDetailByAssignee 单任务详情视图：仅当 task 指派给当前审批人且处于 PENDING
// 时返回其关联实例的申请标题与表单数据，供 GetTask 服务渲染详情页。
//
// 授权口径与 ListPendingByAssignee 一致——ID + AssigneeUserID + TaskStatus=PENDING
// 三重过滤，task 与 caller 不匹配则 NotFound。tenant 由 TenantPrivacy 自动隔离。
// instance 边显式加载后取 Title / FormData（FormData 为 field.Any，按
// GetDefinitionNodeConfig 同款 any→string 经 json.Marshal 落回）。
//
// 其余字段（definition_id / current_node_index / tenant_id 等内部字段）不投影，
// 对齐 MyTaskItem 的最小披露原则。
func (r *WorkflowTaskRepo) GetDetailByAssignee(ctx context.Context, taskID uint32, assigneeUserId uint32) (*oav1.GetTaskResponse, error) {
	if taskID == 0 || assigneeUserId == 0 {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	entity, err := r.entClient.Client().WorkflowTask.Query().
		Where(
			workflowtask.IDEQ(taskID),
			workflowtask.AssigneeUserIDEQ(assigneeUserId),
			workflowtask.TaskStatusEQ(workflowtask.TaskStatusPENDING),
		).
		WithInstance().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, oav1.ErrorNotFound("workflow task not found")
		}
		r.log.Errorf("query workflow task detail failed: %s", err)
		return nil, oav1.ErrorInternalError("query workflow task detail failed")
	}

	resp := &oav1.GetTaskResponse{
		TaskId: ptr(entity.ID),
	}
	inst := entity.Edges.Instance
	if inst != nil {
		resp.InstanceId = ptr(inst.ID)
		if inst.Title != nil {
			resp.Title = inst.Title
		}
		// FormData 为 field.Any，按 GetDefinitionNodeConfig 同款 any→string 经
		// json.Marshal 落回原始 JSON 文本。引擎不解释其字段语义。
		if v := inst.FormData; v != nil {
			if b, mErr := json.Marshal(v); mErr == nil {
				p := string(b)
				resp.FormData = &p
			}
		}
	}
	return resp, nil
}
