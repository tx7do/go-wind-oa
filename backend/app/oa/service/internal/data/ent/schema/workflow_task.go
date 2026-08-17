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

// WorkflowTask OA 工作流任务。实例在某一节点上对指派审批人产生的待办。
// 同一实例同一时刻至多存在一条 PENDING 任务（线性状态机）。
// 任务“开启时间”= 行 created_at（TimeAt mixin）；被处置时间= updated_at。
type WorkflowTask struct{ ent.Schema }

func (WorkflowTask) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "oa_workflow_task", Charset: "utf8mb4", Collation: "utf8mb4_bin"},
		entsql.WithComments(true),
		schema.Comment("OA工作流任务表"),
	}
}

func (WorkflowTask) Fields() []ent.Field {
	return []ent.Field{
		// 该任务对应的定义节点序号（仅用于审计/展示，不参与状态机推进逻辑）。
		field.Int32("node_index").Comment("节点序号").Optional().Nillable().Default(0),
		// 当前指派审批人。转发(转办)时由引擎改写为被转办人。
		field.Uint32("assignee_user_id").Comment("指派审批人").Optional().Nillable(),
		field.Enum("task_status").
			Comment("任务状态").
			NamedValues("PENDING", "PENDING", "APPROVED", "APPROVED", "REJECTED", "REJECTED", "FORWARDED", "FORWARDED").
			Default("PENDING").
			Optional(),
	}
}

func (WorkflowTask) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

func (WorkflowTask) Edges() []ent.Edge {
	return []ent.Edge{
		// 反向边：外键列名 / Required / OnDelete 在正向 edge.To（WorkflowInstance.Edges）侧声明。
		edge.From("instance", WorkflowInstance.Type).
			Ref("tasks").
			Unique(),
	}
}

func (WorkflowTask) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id").
			StorageKey("idx_oa_wf_task_tenant"),
		// “待办”列表按 (tenant_id, assignee_user_id, task_status) 检索。
		index.Fields("tenant_id", "assignee_user_id", "task_status").
			StorageKey("idx_oa_wf_task_tenant_assignee_status"),
	}
}
