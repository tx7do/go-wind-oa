package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// Holiday holds the schema definition for the Holiday entity.
type Holiday struct {
	ent.Schema
}

func (Holiday) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_holiday",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 节假日表（HOLIDAY=法定假日休息，WORKDAY=调休上班；优先于周末判定）"),
	}
}

// Fields of the Holiday.
func (Holiday) Fields() []ent.Field {
	return []ent.Field{
		field.Time("date").
			Comment("日期（零点，租户内唯一）"),

		field.Enum("holiday_type").
			Comment("类型").
			NamedValues(
				"Holiday", "HOLIDAY",
				"Workday", "WORKDAY",
			).
			Default("HOLIDAY").
			Optional().
			Nillable(),

		field.String("name").
			Comment("名称（如 国庆节/调休上班）").
			MaxLen(64).
			Default(""),
	}
}

// Mixin of the Holiday.
func (Holiday) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

func (Holiday) Indexes() []ent.Index {
	return []ent.Index{
		// 租户 + 日期 唯一（一日一条）
		index.Fields("tenant_id", "date").
			Unique().
			StorageKey("uix_oa_holiday_tenant_date"),
	}
}
