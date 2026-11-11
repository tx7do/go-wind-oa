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

// WorkflowToken holds the schema definition for the WorkflowToken entity.
// 流程令牌：go-wind-flow 引擎在图中的执行游标（Token 状态机持久化）。
// 引擎内部数据，不面向客户端——故不带租户/操作人 mixin（flow.Token 无对应字段，
// 由 instance_id 关联到租户）。
type WorkflowToken struct {
	ent.Schema
}

func (WorkflowToken) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_workflow_token",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 工作流令牌表（go-wind-flow Token 状态机持久化）"),
	}
}

// Fields of the WorkflowToken.
func (WorkflowToken) Fields() []ent.Field {
	return []ent.Field{
		field.String("node_id").
			Comment("令牌当前所在节点ID").
			Optional().
			Nillable(),

		field.Enum("token_status").
			Comment("令牌状态").
			NamedValues(
				"Running", "RUNNING",
				"Waiting", "WAITING",
				"Ended", "ENDED",
			).
			Default("RUNNING").
			Optional().
			Nillable(),

		field.Int64("parent_id").
			Comment("fork 产生的子令牌指向分叉源令牌").
			Optional().
			Nillable(),

		field.Int64("child_instance_id").
			Comment("阻塞于子流程节点时记录的子流程实例ID").
			Optional().
			Nillable(),

		field.Time("due_at").
			Comment("阻塞于定时器节点时的到期时刻").
			Optional().
			Nillable(),

		field.String("locals").
			Comment("令牌级局部变量（JSON 文本，并行分支隔离）").
			Optional().
			Nillable(),
	}
}

// Mixin of the WorkflowToken.
func (WorkflowToken) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
	}
}

// Edges of the WorkflowToken.
func (WorkflowToken) Edges() []ent.Edge {
	return []ent.Edge{
		// 反向：令牌→实例。外鍵列名（instance_id）與 Required 約束在正向
		// edge.To（WorkflowInstance.Edges）側宣告，此處僅宣告 Ref + Unique。
		edge.From("instance", WorkflowInstance.Type).
			Ref("tokens").
			Unique(),
	}
}

func (WorkflowToken) Indexes() []ent.Index {
	return []ent.Index{
		// 到期定时器令牌扫描（go-wind-flow ResumeDueTimers）。
		// instance_id 外键列由 edge 注入，ent 自动为其建索引。
		index.Fields("token_status", "due_at").
			StorageKey("idx_oa_workflow_token_status_due"),
	}
}
