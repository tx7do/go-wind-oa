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

// ExpenseItem holds the schema definition for the ExpenseItem entity.
type ExpenseItem struct {
	ent.Schema
}

func (ExpenseItem) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_expense_item",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 报销明细表"),
	}
}

// Fields of the ExpenseItem.
func (ExpenseItem) Fields() []ent.Field {
	return []ent.Field{
		field.String("category").
			Comment("费用类别（交通/餐饮/办公等）").
			MaxLen(64).
			Default(""),

		field.Float("amount").
			Comment("金额").
			Default(0),

		field.Time("expense_date").
			Comment("费用发生日期").
			Optional().
			Nillable(),

		field.String("description").
			Comment("费用说明").
			MaxLen(512).
			Default(""),

		field.Uint32("invoice_file_id").
			Comment("发票凭证文件ID（storage 域文件，OSS 存储）").
			Optional().
			Nillable(),
	}
}

// Mixin of the ExpenseItem.
func (ExpenseItem) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

// Edges of the ExpenseItem.
func (ExpenseItem) Edges() []ent.Edge {
	return []ent.Edge{
		// 反向：明细→申请单。外键列名（expense_application_id）與 Required 約束在正向
		// edge.To（ExpenseApplication.Edges）側宣告，此處僅宣告 Ref + Unique。
		edge.From("application", ExpenseApplication.Type).
			Ref("items").
			Unique(),
	}
}

func (ExpenseItem) Indexes() []ent.Index {
	return []ent.Index{
		// 租户筛选
		index.Fields("tenant_id").
			StorageKey("idx_oa_expense_item_tenant"),
	}
}
