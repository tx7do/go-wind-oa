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

// WorkflowLog holds the schema definition for the WorkflowLog entity.
type WorkflowLog struct {
	ent.Schema
}

func (WorkflowLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_workflow_log",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 工作流审批日志表（append-only）"),
	}
}

// Fields of the WorkflowLog.
func (WorkflowLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int("node_index").
			Comment("节点索引").
			Optional().
			Nillable(),

		field.Enum("log_action").
			Comment("日志动作").
			NamedValues(
				"Submit", "SUBMIT",
				"Approve", "APPROVE",
				"Reject", "REJECT",
				"Forward", "FORWARD",
				"Withdraw", "WITHDRAW",
			).
			Default("SUBMIT").
			Optional().
			Nillable(),

		field.String("comment").
			Comment("审批意见").
			Optional().
			Nillable(),
	}
}

// Mixin of the WorkflowLog.
func (WorkflowLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

// Edges of the WorkflowLog.
func (WorkflowLog) Edges() []ent.Edge {
	return []ent.Edge{
		// 反向：日志→实例。外鍵列名（instance_id）與 Required 約束在正向
		// edge.To（WorkflowInstance.Edges）側宣告，此處僅宣告 Ref + Unique。
		edge.From("instance", WorkflowInstance.Type).
			Ref("logs").
			Unique(),
	}
}

func (WorkflowLog) Indexes() []ent.Index {
	return []ent.Index{
		// 租户筛选
		index.Fields("tenant_id").
			StorageKey("idx_oa_workflow_log_tenant"),

		// 按租户 + 操作者，用于“已办”列表（审计回溯）
		index.Fields("tenant_id", "created_by").
			StorageKey("idx_oa_workflow_log_tenant_created_by"),
	}
}
