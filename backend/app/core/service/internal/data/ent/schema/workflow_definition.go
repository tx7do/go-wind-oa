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

// WorkflowDefinition holds the schema definition for the WorkflowDefinition entity.
type WorkflowDefinition struct {
	ent.Schema
}

func (WorkflowDefinition) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_workflow_definition",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 工作流定义表"),
	}
}

// Fields of the WorkflowDefinition.
func (WorkflowDefinition) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			Comment("流程代码").
			Optional().
			Nillable(),

		field.Int("version").
			Comment("版本号").
			Optional().
			Nillable(),

		field.String("node_config").
			Comment("节点配置（JSON 文本）").
			Optional().
			Nillable(),

		field.String("form_schema").
			Comment("表单 schema（JSON 文本）").
			Optional().
			Nillable(),

		field.Enum("definition_status").
			Comment("定义状态").
			NamedValues(
				"Draft", "DRAFT",
				"Enabled", "ENABLED",
				"Disabled", "DISABLED",
			).
			Default("DRAFT").
			Optional().
			Nillable(),
	}
}

// Mixin of the WorkflowDefinition.
func (WorkflowDefinition) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
		mixin.Remark{},
	}
}

// Edges of the WorkflowDefinition.
func (WorkflowDefinition) Edges() []ent.Edge {
	return []ent.Edge{
		// 定义→实例：正向 O2M，外键列 definition_id 在正向 edge.To 側宣告，級聯刪除。
		edge.To("instances", WorkflowInstance.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}).
			StorageKey(edge.Column("definition_id")),
	}
}

func (WorkflowDefinition) Indexes() []ent.Index {
	return []ent.Index{
		// 租户筛选
		index.Fields("tenant_id").
			StorageKey("idx_oa_workflow_def_tenant"),

		// 租户级唯一：同租户下 (code, version) 唯一
		index.Fields("tenant_id", "code", "version").
			Unique().
			StorageKey("uix_oa_workflow_def_tenant_code_version"),
	}
}
