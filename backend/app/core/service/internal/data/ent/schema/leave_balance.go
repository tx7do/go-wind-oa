package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// LeaveBalance holds the schema definition for the LeaveBalance entity.
type LeaveBalance struct {
	ent.Schema
}

func (LeaveBalance) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_leave_balance",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 假期额度表（用户 x 类型 x 年度）"),
	}
}

// Fields of the LeaveBalance.
func (LeaveBalance) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("user_id").
			Comment("用户ID"),

		field.Uint32("leave_type_id").
			Comment("请假类型ID"),

		field.Int("year").
			Comment("年度"),

		field.Float("total_days").
			Comment("总额度（天）").
			Default(0),

		field.Float("used_days").
			Comment("已用（天）").
			Default(0),
	}
}

// Mixin of the LeaveBalance.
func (LeaveBalance) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

func (LeaveBalance) Indexes() []ent.Index {
	return []ent.Index{
		// 租户筛选
		index.Fields("tenant_id").
			StorageKey("idx_oa_leave_balance_tenant"),

		// 租户 + 用户（查本人额度）
		index.Fields("tenant_id", "user_id").
			StorageKey("idx_oa_leave_balance_tenant_user"),

		// 租户 + 用户 + 类型 + 年度 唯一
		index.Fields("tenant_id", "user_id", "leave_type_id", "year").
			Unique().
			StorageKey("uix_oa_leave_balance_tenant_user_type_year"),
	}
}
