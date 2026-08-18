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

	"go-wind-oa/app/core/service/internal/data/ent"
	"go-wind-oa/app/core/service/internal/data/ent/predicate"
	"go-wind-oa/app/core/service/internal/data/ent/workflowdefinition"

	oav1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// WorkflowDefinitionRepo OA 工作流定义的数据访问层。
//
// 仓库约定：
//   - entClient + mapper.CopierMapper + EnumTypeConverter + 泛型 entCrud.Repository；
//   - Create 走手写 builder（Any 字段 node_config/form_schema 由 JSON 文本 ↔ any 显式转换）；
//   - List 走泛型 ListWithPaging（tenant 由 TenantPrivacy 策略按 viewer 自动隔离）；
//   - GetByCodeVersion 为状态机提交申请时的定向查询，Any 字段单独落回 DTO。
type WorkflowDefinitionRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper                    *mapper.CopierMapper[oav1.WorkflowDefinition, ent.WorkflowDefinition]
	definitionStatusConverter *mapper.EnumTypeConverter[oav1.WorkflowDefinition_DefinitionStatus, workflowdefinition.DefinitionStatus]

	repository *entCrud.Repository[
		ent.WorkflowDefinitionQuery, ent.WorkflowDefinitionSelect,
		ent.WorkflowDefinitionCreate, ent.WorkflowDefinitionCreateBulk,
		ent.WorkflowDefinitionUpdate, ent.WorkflowDefinitionUpdateOne,
		ent.WorkflowDefinitionDelete,
		predicate.WorkflowDefinition,
		oav1.WorkflowDefinition, ent.WorkflowDefinition,
	]
}

func NewWorkflowDefinitionRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *WorkflowDefinitionRepo {
	repo := &WorkflowDefinitionRepo{
		log:                       ctx.NewLoggerHelper("workflow-definition/repo/oa-service"),
		entClient:                 entClient,
		mapper:                    mapper.NewCopierMapper[oav1.WorkflowDefinition, ent.WorkflowDefinition](),
		definitionStatusConverter: mapper.NewEnumTypeConverter[oav1.WorkflowDefinition_DefinitionStatus, workflowdefinition.DefinitionStatus](oav1.WorkflowDefinition_DefinitionStatus_name, oav1.WorkflowDefinition_DefinitionStatus_value),
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
		oav1.WorkflowDefinition, ent.WorkflowDefinition,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(r.definitionStatusConverter.NewConverterPair())
}

// GetByCodeVersion 按 (code, version) 取定义；tenant 由 TenantPrivacy 按 viewer 自动过滤。
// node_config（entity any → DTO JSON 文本）单独落回，其余字段由 mapper 处理。
func (r *WorkflowDefinitionRepo) GetByCodeVersion(ctx context.Context, code string, version uint32) (*oav1.WorkflowDefinition, error) {
	if code == "" {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	entity, err := r.entClient.Client().WorkflowDefinition.Query().
		Where(
			workflowdefinition.CodeEQ(code),
			workflowdefinition.VersionEQ(version),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, oav1.ErrorNotFound("workflow definition not found")
		}
		r.log.Errorf("query workflow definition failed: %s", err)
		return nil, oav1.ErrorInternalError("query workflow definition failed")
	}

	dto := r.mapper.ToDTO(entity)
	// 节点配置：entity any → DTO JSON 文本。表单 schema 在管理端 Get 中按需补充，此处不返回。
	if v := entity.NodeConfig; v != nil {
		if b, mErr := json.Marshal(v); mErr == nil {
			p := string(b)
			dto.NodeConfig = &p
		}
	}
	return dto, nil
}

// GetByID 按 id 取定义详情，供管理端查看。tenant 由 TenantPrivacy 按 viewer 自动过滤。
// 与 GetByCodeVersion 不同，此处同时落回 node_config 与 form_schema（管理端需查看两者）。
func (r *WorkflowDefinitionRepo) GetByID(ctx context.Context, id uint32) (*oav1.WorkflowDefinition, error) {
	if id == 0 {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	entity, err := r.entClient.Client().WorkflowDefinition.Query().
		Where(workflowdefinition.IDEQ(id)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, oav1.ErrorNotFound("workflow definition not found")
		}
		r.log.Errorf("query workflow definition failed: %s", err)
		return nil, oav1.ErrorInternalError("query workflow definition failed")
	}

	dto := r.mapper.ToDTO(entity)
	// node_config / form_schema：entity any → DTO JSON 文本。
	if v := entity.NodeConfig; v != nil {
		if b, mErr := json.Marshal(v); mErr == nil {
			p := string(b)
			dto.NodeConfig = &p
		}
	}
	if v := entity.FormSchema; v != nil {
		if b, mErr := json.Marshal(v); mErr == nil {
			p := string(b)
			dto.FormSchema = &p
		}
	}
	return dto, nil
}

func (r *WorkflowDefinitionRepo) Create(ctx context.Context, req *oav1.WorkflowDefinition) (*oav1.WorkflowDefinition, error) {
	if req == nil {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	builder := r.entClient.Client().WorkflowDefinition.Create().
		SetNillableTenantID(req.TenantId).
		SetNillableCreatedBy(req.CreatedBy).
		SetNillableName(req.Name).
		SetNillableCode(req.Code).
		SetNillableVersion(req.Version).
		SetNillableDescription(req.Description).
		SetNillableDefinitionStatus(r.definitionStatusConverter.ToEntity(req.DefinitionStatus)).
		SetCreatedAt(time.Now())

	// 节点配置 / 表单 schema：DTO JSON 文本 → entity any。
	if req.NodeConfig != nil {
		var v any
		if err := json.Unmarshal([]byte(*req.NodeConfig), &v); err == nil {
			builder.SetNodeConfig(v)
		}
	}
	if req.FormSchema != nil {
		var v any
		if err := json.Unmarshal([]byte(*req.FormSchema), &v); err == nil {
			builder.SetFormSchema(v)
		}
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert workflow definition failed: %s", err)
		return nil, oav1.ErrorInternalError("insert workflow definition failed")
	}
	return r.mapper.ToDTO(entity), nil
}

func (r *WorkflowDefinitionRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*oav1.ListWorkflowDefinitionResponse, error) {
	if req == nil {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	builder := r.entClient.Client().WorkflowDefinition.Query()
	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &oav1.ListWorkflowDefinitionResponse{Total: 0, Items: nil}, nil
	}
	return &oav1.ListWorkflowDefinitionResponse{Total: ret.Total, Items: ret.Items}, nil
}

// UpdateStatus 切换指定定义的 definition_status（DRAFT↔ENABLED↔DISABLED）。
//
// 鉴权：tenant 谓词显式叠加（IDEQ + TenantIDEQ），与 cms internal_message_repo.Update
// 同款口径——即使 TenantPrivacy 钩子失效，DB 层仍拒绝跨租户写入。updated_at /
// updated_by 强制由服务端 viewer context 推导，忽略客户端传入值。仅更新
// definition_status 与审计字段，其余字段不动。
func (r *WorkflowDefinitionRepo) UpdateStatus(ctx context.Context, id uint32, status oav1.WorkflowDefinition_DefinitionStatus) (*oav1.WorkflowDefinition, error) {
	if id == 0 {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)

	builder := r.entClient.Client().WorkflowDefinition.UpdateOneID(id).
		Where(workflowdefinition.IDEQ(id))
	if hasTenant {
		builder.Where(workflowdefinition.TenantIDEQ(tid))
	}
	builder.SetNillableDefinitionStatus(r.definitionStatusConverter.ToEntity(&status))
	builder.SetUpdatedAt(time.Now())
	if hasUser {
		builder.SetUpdatedBy(callerUserID)
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, oav1.ErrorNotFound("workflow definition not found")
		}
		r.log.Errorf("update workflow definition status failed: %s", err)
		return nil, oav1.ErrorInternalError("update workflow definition status failed")
	}
	return r.mapper.ToDTO(entity), nil
}
