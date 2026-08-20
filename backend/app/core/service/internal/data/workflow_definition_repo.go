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

	"go-wind-oa/app/core/service/internal/data/ent"
	"go-wind-oa/app/core/service/internal/data/ent/predicate"
	"go-wind-oa/app/core/service/internal/data/ent/workflowdefinition"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

type WorkflowDefinitionRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper                    *mapper.CopierMapper[oaV1.WorkflowDefinition, ent.WorkflowDefinition]
	definitionStatusConverter *mapper.EnumTypeConverter[oaV1.WorkflowDefinition_DefinitionStatus, workflowdefinition.DefinitionStatus]

	repository *entCrud.Repository[
		ent.WorkflowDefinitionQuery, ent.WorkflowDefinitionSelect,
		ent.WorkflowDefinitionCreate, ent.WorkflowDefinitionCreateBulk,
		ent.WorkflowDefinitionUpdate, ent.WorkflowDefinitionUpdateOne,
		ent.WorkflowDefinitionDelete,
		predicate.WorkflowDefinition,
		oaV1.WorkflowDefinition, ent.WorkflowDefinition,
	]
}

func NewWorkflowDefinitionRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *WorkflowDefinitionRepo {
	repo := &WorkflowDefinitionRepo{
		log:                       ctx.NewLoggerHelper("workflow-definition/repo/core-service"),
		entClient:                 entClient,
		mapper:                    mapper.NewCopierMapper[oaV1.WorkflowDefinition, ent.WorkflowDefinition](),
		definitionStatusConverter: mapper.NewEnumTypeConverter[oaV1.WorkflowDefinition_DefinitionStatus, workflowdefinition.DefinitionStatus](oaV1.WorkflowDefinition_DefinitionStatus_name, oaV1.WorkflowDefinition_DefinitionStatus_value),
	}

	repo.init()

	return repo
}

func (r *WorkflowDefinitionRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.WorkflowDefinitionQuery, ent.WorkflowDefinitionSelect,
		ent.WorkflowDefinitionCreate, ent.WorkflowDefinitionCreateBulk,
		ent.WorkflowDefinitionUpdate, ent.WorkflowDefinitionUpdateOne,
		ent.WorkflowDefinitionDelete,
		predicate.WorkflowDefinition,
		oaV1.WorkflowDefinition, ent.WorkflowDefinition,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.definitionStatusConverter.NewConverterPair())
}

// toDTO 在 mapper 之後顯式回填 DefinitionStatus。
// 原因：CopierMapper 對 entity 字符串枚舉（非指針存儲但 API 暴露為指針）→ proto 枚舉指針
// 的映射不可靠，會靜默落 nil。狀態機的啟用校驗（SubmitApply 檢查 ENABLED）依賴該字段，
// 故此處顯式回填，避免已啟用定義被誤判為未啟用。
func (r *WorkflowDefinitionRepo) toDTO(entity *ent.WorkflowDefinition) *oaV1.WorkflowDefinition {
	dto := r.mapper.ToDTO(entity)
	if dto != nil && entity != nil {
		dto.DefinitionStatus = r.definitionStatusConverter.ToDTO(entity.DefinitionStatus)
	}
	return dto
}

func (r *WorkflowDefinitionRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*oaV1.ListWorkflowDefinitionResponse, error) {
	if req == nil {
		return nil, oaV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().WorkflowDefinition.Query()
	if tid, hasTenant := maybeTenantFromViewer(ctx); hasTenant {
		builder.Where(workflowdefinition.TenantIDEQ(tid))
	}

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &oaV1.ListWorkflowDefinitionResponse{Total: 0, Items: nil}, nil
	}

	return &oaV1.ListWorkflowDefinitionResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *WorkflowDefinitionRepo) Get(ctx context.Context, req *oaV1.GetWorkflowDefinitionRequest) (*oaV1.WorkflowDefinition, error) {
	if req == nil {
		return nil, oaV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	builder := r.entClient.Client().WorkflowDefinition.Query().
		Where(workflowdefinition.IDEQ(req.GetId()))
	if hasTenant {
		builder.Where(workflowdefinition.TenantIDEQ(tid))
	}

	entity, err := builder.Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, oaV1.ErrorNotFound("workflow definition not found")
		}
		r.log.Errorf("query definition failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query definition failed")
	}

	return r.toDTO(entity), nil
}

func (r *WorkflowDefinitionRepo) Create(ctx context.Context, req *oaV1.CreateWorkflowDefinitionRequest) (*oaV1.WorkflowDefinition, error) {
	if req == nil || req.Data == nil {
		return nil, oaV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().WorkflowDefinition.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableRemark(req.Data.Remark).
		SetNillableCode(req.Data.Code).
		SetNillableNodeConfig(req.Data.NodeConfig).
		SetNillableFormSchema(req.Data.FormSchema).
		SetNillableDefinitionStatus(r.definitionStatusConverter.ToEntity(req.Data.DefinitionStatus)).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if req.Data.Version != nil {
		builder.SetVersion(int(*req.Data.Version))
	}

	if req.Data.Id != nil {
		builder.SetID(req.Data.GetId())
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert workflow definition failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("insert workflow definition failed")
	}

	return r.toDTO(entity), nil
}

// UpdateStatus 切換定義狀態（啟用/禁用）。服務層已校驗 update_mask 僅含 definition_status。
func (r *WorkflowDefinitionRepo) UpdateStatus(ctx context.Context, id uint32, newStatus *oaV1.WorkflowDefinition_DefinitionStatus) error {
	if id == 0 || newStatus == nil {
		return oaV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)
	builder := r.entClient.Client().WorkflowDefinition.Update()
	builder.Where(workflowdefinition.IDEQ(id))
	if hasTenant {
		builder.Where(workflowdefinition.TenantIDEQ(tid))
	}
	builder.SetNillableDefinitionStatus(r.definitionStatusConverter.ToEntity(newStatus))
	builder.SetUpdatedAt(time.Now())
	if hasUser {
		builder.SetUpdatedBy(callerUserID)
	}

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("update definition status failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("update definition status failed")
	}
	return nil
}

// GetByCodeVersion 供 SubmitApply 按流程代碼+版本查詢定義。返回完整 DTO，服務層讀取
// DefinitionStatus（經 toDTO 回填，可靠）與 NodeConfig（JSON 文本）。
func (r *WorkflowDefinitionRepo) GetByCodeVersion(ctx context.Context, code string, version int32) (*oaV1.WorkflowDefinition, error) {
	if code == "" {
		return nil, oaV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	if !hasTenant {
		return nil, oaV1.ErrorForbidden("missing viewer context")
	}

	builder := r.entClient.Client().WorkflowDefinition.Query().
		Where(
			workflowdefinition.CodeEQ(code),
			workflowdefinition.VersionEQ(int(version)),
			workflowdefinition.TenantIDEQ(tid),
		)

	entity, err := builder.Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, oaV1.ErrorNotFound("workflow definition not found")
		}
		r.log.Errorf("query definition by code/version failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query definition failed")
	}

	return r.toDTO(entity), nil
}
