package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// WorkflowDelegation holds the schema definition for the WorkflowDelegation entity.
type WorkflowDelegation struct {
	ent.Schema
}

func (WorkflowDelegation) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_workflow_delegation",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 工作流审批委托表（审批人指定代理人，待办自动转发）"),
	}
}

func (WorkflowDelegation) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("delegator_user_id").
			Comment("委托人用户ID（原审批人）").
			Optional().
			Nillable(),

		field.Uint32("delegate_user_id").
			Comment("被委托人用户ID（代理人）").
			Optional().
			Nillable(),
	}
}

func (WorkflowDelegation) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

func (WorkflowDelegation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id").
			StorageKey("idx_oa_workflow_delegation_tenant"),

		// 租户内委托人唯一：一人只能指定一名代理人
		index.Fields("tenant_id", "delegator_user_id").
			Unique().
			StorageKey("uix_oa_workflow_delegation_tenant_delegator"),
	}
}
