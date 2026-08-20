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

// ExpenseApplication holds the schema definition for the ExpenseApplication entity.
type ExpenseApplication struct {
	ent.Schema
}

func (ExpenseApplication) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_expense_application",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 费用报销申请单表"),
	}
}

// Fields of the ExpenseApplication.
func (ExpenseApplication) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			Comment("报销事由标题").
			MaxLen(255).
			Default(""),

		field.Float("total_amount").
			Comment("报销总额（各明细金额之和）").
			Default(0),

		field.Enum("expense_status").
			Comment("申请单状态").
			NamedValues(
				"Pending", "PENDING",
				"Approved", "APPROVED",
				"Rejected", "REJECTED",
				"Withdrawn", "WITHDRAWN",
			).
			Default("PENDING").
			Optional().
			Nillable(),

		field.Uint32("instance_id").
			Comment("关联工作流实例ID").
			Optional().
			Nillable(),
	}
}

// Mixin of the ExpenseApplication.
func (ExpenseApplication) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

// Edges of the ExpenseApplication.
func (ExpenseApplication) Edges() []ent.Edge {
	return []ent.Edge{
		// 正向：申请单→明细。外键列 expense_application_id 在此侧声明，级联删除。
		edge.To("items", ExpenseItem.Type).
			Required().
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}).
			StorageKey(edge.Column("expense_application_id")),
	}
}

func (ExpenseApplication) Indexes() []ent.Index {
	return []ent.Index{
		// 租户筛选
		index.Fields("tenant_id").
			StorageKey("idx_oa_expense_app_tenant"),

		// 租户 + 申请人（“我的申请”）
		index.Fields("tenant_id", "created_by").
			StorageKey("idx_oa_expense_app_tenant_created_by"),
	}
}
