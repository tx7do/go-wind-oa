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

// WorkflowInstanceJoin holds the schema definition for the WorkflowInstanceJoin entity.
type WorkflowInstanceJoin struct {
	ent.Schema
}

func (WorkflowInstanceJoin) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_workflow_instance_join",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 工作流汇聚节点到达计数表"),
	}
}

// Fields of the WorkflowInstanceJoin.
func (WorkflowInstanceJoin) Fields() []ent.Field {
	return []ent.Field{
		field.String("join_node_id").
			Comment("汇聚节点ID（图节点 id）").
			Optional().
			Nillable(),

		field.Int("arrived_count").
			Comment("已到达分支数").
			Default(0).
			Optional().
			Nillable(),
	}
}

// Mixin of the WorkflowInstanceJoin.
func (WorkflowInstanceJoin) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

// Edges of the WorkflowInstanceJoin.
func (WorkflowInstanceJoin) Edges() []ent.Edge {
	return []ent.Edge{
		// 反向：汇聚计数→实例。外鍵列名（instance_id）與 Required 約束在正向
		// edge.To（WorkflowInstance.Edges）側宣告，此處僅宣告 Ref + Unique。
		edge.From("instance", WorkflowInstance.Type).
			Ref("instance_joins").
			Unique(),
	}
}

func (WorkflowInstanceJoin) Indexes() []ent.Index {
	return []ent.Index{
		// 租户筛选
		index.Fields("tenant_id").
			StorageKey("idx_oa_workflow_inst_join_tenant"),
	}
}
