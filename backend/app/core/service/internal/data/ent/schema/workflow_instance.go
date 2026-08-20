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

// WorkflowInstance holds the schema definition for the WorkflowInstance entity.
type WorkflowInstance struct {
	ent.Schema
}

func (WorkflowInstance) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_workflow_instance",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 工作流实例表"),
	}
}

// Fields of the WorkflowInstance.
func (WorkflowInstance) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("instance_status").
			Comment("实例状态").
			NamedValues(
				"Pending", "PENDING",
				"Approved", "APPROVED",
				"Rejected", "REJECTED",
				"Withdrawn", "WITHDRAWN",
			).
			Default("PENDING").
			Optional().
			Nillable(),

		field.Int("current_node_index").
			Comment("当前节点索引").
			Optional().
			Nillable(),

		field.String("form_data").
			Comment("申请表单数据（JSON 文本）").
			Optional().
			Nillable(),

		field.String("business_type").
			Comment("业务单据类型（LEAVE/EXPENSE 等，审批终结时回调业务模块）").
			Optional().
			Nillable(),

		field.Uint32("business_id").
			Comment("业务单据ID").
			Optional().
			Nillable(),
	}
}

// Mixin of the WorkflowInstance.
func (WorkflowInstance) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

// Edges of the WorkflowInstance.
func (WorkflowInstance) Edges() []ent.Edge {
	return []ent.Edge{
		// 反向：实例→定义。外鍵列名（definition_id）與 Required 約束在正向
		// edge.To（WorkflowDefinition.Edges）側宣告，此處僅宣告 Ref + Unique。
		edge.From("definition", WorkflowDefinition.Type).
			Ref("instances").
			Unique(),

		// 正向：实例→任务。外鍵列 instance_id 在此側宣告，級聯刪除。
		edge.To("tasks", WorkflowTask.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}).
			StorageKey(edge.Column("instance_id")),

		// 正向：实例→日志。外鍵列 instance_id 在此側宣告，級聯刪除。
		edge.To("logs", WorkflowLog.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}).
			StorageKey(edge.Column("instance_id")),
	}
}

func (WorkflowInstance) Indexes() []ent.Index {
	return []ent.Index{
		// 租户筛选
		index.Fields("tenant_id").
			StorageKey("idx_oa_workflow_inst_tenant"),

		// 按租户 + 创建者，用于“我的申请”列表
		index.Fields("tenant_id", "created_by").
			StorageKey("idx_oa_workflow_inst_tenant_created_by"),
	}
}
