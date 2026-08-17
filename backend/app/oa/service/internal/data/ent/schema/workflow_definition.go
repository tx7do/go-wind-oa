package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// WorkflowDefinition OA 工作流定义。流程模板以 JSON 形式存储有序审批节点，
// 由引擎在提交申请时解析为状态机节点序列。同一 (tenant_id, code, version) 唯一。
type WorkflowDefinition struct{ ent.Schema }

func (WorkflowDefinition) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "oa_workflow_definition", Charset: "utf8mb4", Collation: "utf8mb4_bin"},
		entsql.WithComments(true),
		schema.Comment("OA工作流定义表"),
	}
}

func (WorkflowDefinition) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").Comment("流程名称").Optional().Nillable(),
		field.String("code").Comment("流程编码").Immutable().Optional().Nillable(),
		field.Uint32("version").Comment("版本号").Optional().Nillable().Default(1),
		field.String("description").Comment("描述").Optional().Nillable(),
		// 节点配置：有序审批节点（approver 类型、节点序号、元数据）。
		// 结构由引擎在 internal/workflow 包中解析，DB 仅按任意 JSON 落盘。
		field.Any("node_config").Comment("节点配置(JSON)").Optional(),
		// 动态表单 schema：供前端按定义渲染表单字段。
		field.Any("form_schema").Comment("动态表单schema(JSON)").Optional(),
		field.Enum("definition_status").
			Comment("定义状态").
			NamedValues("DRAFT", "DRAFT", "ENABLED", "ENABLED", "DISABLED", "DISABLED").
			Default("DRAFT").
			Optional(),
	}
}

func (WorkflowDefinition) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
		mixin.Remark{},
	}
}

func (WorkflowDefinition) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("instances", WorkflowInstance.Type).
			Required().
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}).
			StorageKey(edge.Column("definition_id")),
	}
}

func (WorkflowDefinition) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id").
			StorageKey("idx_oa_wf_def_tenant"),
		index.Fields("tenant_id", "code", "version").
			Unique().
			StorageKey("uix_oa_wf_def_tenant_code_version"),
	}
}
