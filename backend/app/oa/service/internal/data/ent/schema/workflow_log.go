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

// WorkflowLog OA 工作流审计日志。append-only：实例的每一次状态迁移都落一条。
// 操作人 = 行 created_by（OperatorID mixin），“已办”列表据此检索。
type WorkflowLog struct{ ent.Schema }

func (WorkflowLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "oa_workflow_log", Charset: "utf8mb4", Collation: "utf8mb4_bin"},
		entsql.WithComments(true),
		schema.Comment("OA工作流审计日志表"),
	}
}

func (WorkflowLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int32("node_index").Comment("节点序号").Optional().Nillable().Default(0),
		field.Enum("log_action").
			Comment("审计动作").
			NamedValues("SUBMIT", "SUBMIT", "APPROVE", "APPROVE", "REJECT", "REJECT", "FORWARD", "FORWARD", "CANCEL", "CANCEL").
			Default("SUBMIT").
			Optional(),
		field.String("comment").Comment("审批意见").Optional().Nillable(),
	}
}

func (WorkflowLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

func (WorkflowLog) Edges() []ent.Edge {
	return []ent.Edge{
		// 反向边：外键列名 / Required / OnDelete 在正向 edge.To（WorkflowInstance.Edges）侧声明。
		edge.From("instance", WorkflowInstance.Type).
			Ref("logs").
			Unique(),
	}
}

func (WorkflowLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id").
			StorageKey("idx_oa_wf_log_tenant"),
		// “已办”列表按 (tenant_id, created_by) 检索。
		index.Fields("tenant_id", "created_by").
			StorageKey("idx_oa_wf_log_tenant_actor"),
	}
}
