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

// WorkflowInstance OA 工作流实例。一次具体申请（请假/报销等）。
// form_data 为申请人提交的动态表单数据，按任意 JSON 落盘；status 由状态机驱动。
// current_node_index 仅在 status==PENDING 时有效，指向定义中当前待办节点序号。
type WorkflowInstance struct{ ent.Schema }

func (WorkflowInstance) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "oa_workflow_instance", Charset: "utf8mb4", Collation: "utf8mb4_bin"},
		entsql.WithComments(true),
		schema.Comment("OA工作流实例表"),
	}
}

func (WorkflowInstance) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").Comment("申请标题").Optional().Nillable(),
		// 申请人提交的动态表单数据，任意 JSON 结构。引擎不解释其字段语义。
		field.Any("form_data").Comment("申请表单数据(JSON)").Optional(),
		field.Enum("instance_status").
			Comment("实例状态").
			NamedValues("PENDING", "PENDING", "APPROVED", "APPROVED", "REJECTED", "REJECTED", "CANCELED", "CANCELED").
			Default("PENDING").
			Optional(),
		// 当前待办节点序号（指向定义节点序列的下标）。状态非 PENDING 时无意义。
		field.Int32("current_node_index").Comment("当前节点序号").Optional().Nillable().Default(0),
	}
}

func (WorkflowInstance) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
		mixin.Remark{},
	}
}

func (WorkflowInstance) Edges() []ent.Edge {
	return []ent.Edge{
		// 实例必须归属一个定义；定义删除时级联清除实例。
		// 反向边：外键列名 / Required 约束在正向 edge.To（WorkflowDefinition.Edges）侧声明，
		// 此处仅声明 Ref + Unique；外键列名与 Required 约束在正向 edge.To 侧声明。
		edge.From("definition", WorkflowDefinition.Type).
			Ref("instances").
			Unique(),
		edge.To("tasks", WorkflowTask.Type).
			Required().
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}).
			StorageKey(edge.Column("instance_id")),
		edge.To("logs", WorkflowLog.Type).
			Required().
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}).
			StorageKey(edge.Column("instance_id")),
	}
}

func (WorkflowInstance) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id").
			StorageKey("idx_oa_wf_inst_tenant"),
		// “我的申请”列表按 (tenant_id, created_by) 检索；created_by 由 OperatorID mixin 提供。
		index.Fields("tenant_id", "created_by").
			StorageKey("idx_oa_wf_inst_tenant_creator"),
	}
}
