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

// WorkflowInstanceParentLink holds the schema definition for the WorkflowInstanceParentLink entity.
type WorkflowInstanceParentLink struct {
	ent.Schema
}

func (WorkflowInstanceParentLink) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_workflow_instance_parent_link",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 工作流父子实例关联表（子流程挂起/恢复用）"),
	}
}

// Fields of the WorkflowInstanceParentLink.
func (WorkflowInstanceParentLink) Fields() []ent.Field {
	return []ent.Field{
		field.String("subprocess_node_id").
			Comment("父流程中 SUBPROCESS 节点的ID").
			Optional().
			Nillable(),

		field.Uint32("child_instance_id").
			Comment("子流程实例ID").
			Optional().
			Nillable(),
	}
}

// Mixin of the WorkflowInstanceParentLink.
func (WorkflowInstanceParentLink) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

// Edges of the WorkflowInstanceParentLink.
func (WorkflowInstanceParentLink) Edges() []ent.Edge {
	return []ent.Edge{
		// 反向：父子关联→父实例。外鍵列名（parent_instance_id）與 Required 約束在正向
		// edge.To（WorkflowInstance.Edges）側宣告，此處僅宣告 Ref + Unique。
		edge.From("parent_instance", WorkflowInstance.Type).
			Ref("parent_links").
			Unique(),
	}
}

func (WorkflowInstanceParentLink) Indexes() []ent.Index {
	return []ent.Index{
		// 租户筛选
		index.Fields("tenant_id").
			StorageKey("idx_oa_workflow_inst_pl_tenant"),
	}
}
