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

// WorkflowTask holds the schema definition for the WorkflowTask entity.
type WorkflowTask struct {
	ent.Schema
}

func (WorkflowTask) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_workflow_task",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 工作流任务表"),
	}
}

// Fields of the WorkflowTask.
func (WorkflowTask) Fields() []ent.Field {
	return []ent.Field{
		field.String("node_id").
			Comment("图节点ID").
			Optional().
			Nillable(),

		field.Uint32("assignee_user_id").
			Comment("指派审批人ID").
			Optional().
			Nillable(),

		field.Time("reminded_at").
			Comment("最近一次超时催办时间；非空表示已催办过，避免重复催办").
			Optional().
			Nillable(),

		field.Time("escalated_at").
			Comment("超时自动升级时间；非空表示已升级过，避免沿组织树连环上转").
			Optional().
			Nillable(),

		field.Enum("task_status").
			Comment("任务状态").
			NamedValues(
				"Pending", "PENDING",
				"Approved", "APPROVED",
				"Rejected", "REJECTED",
				"Cancelled", "CANCELLED",
			).
			Default("PENDING").
			Optional().
			Nillable(),
	}
}

// Mixin of the WorkflowTask.
func (WorkflowTask) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

// Edges of the WorkflowTask.
func (WorkflowTask) Edges() []ent.Edge {
	return []ent.Edge{
		// 反向：任务→实例。外鍵列名（instance_id）與 Required 約束在正向
		// edge.To（WorkflowInstance.Edges）側宣告，此處僅宣告 Ref + Unique。
		edge.From("instance", WorkflowInstance.Type).
			Ref("tasks").
			Unique(),
	}
}

func (WorkflowTask) Indexes() []ent.Index {
	return []ent.Index{
		// 租户筛选
		index.Fields("tenant_id").
			StorageKey("idx_oa_workflow_task_tenant"),

		// 按租户 + 指派审批人 + 任务状态，用于“待办”列表
		index.Fields("tenant_id", "assignee_user_id", "task_status").
			StorageKey("idx_oa_workflow_task_tenant_assignee_status"),
	}
}
