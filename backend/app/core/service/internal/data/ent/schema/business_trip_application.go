package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// BusinessTripApplication holds the schema definition for the BusinessTripApplication entity.
type BusinessTripApplication struct {
	ent.Schema
}

func (BusinessTripApplication) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_business_trip_application",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 出差申请单表"),
	}
}

// Fields of the BusinessTripApplication.
func (BusinessTripApplication) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			Comment("出差事由标题").
			MaxLen(256).
			Default(""),

		field.String("destination").
			Comment("出差目的地").
			MaxLen(256).
			Default(""),

		field.Time("start_date").
			Comment("开始日期"),

		field.Time("end_date").
			Comment("结束日期（含）"),

		field.String("itinerary").
			Comment("行程安排说明").
			MaxLen(1024).
			Default(""),

		field.Enum("trip_status").
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

// Mixin of the BusinessTripApplication.
func (BusinessTripApplication) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

func (BusinessTripApplication) Indexes() []ent.Index {
	return []ent.Index{
		// 租户筛选
		index.Fields("tenant_id").
			StorageKey("idx_oa_biz_trip_tenant"),

		// 租户 + 申请人（“我的申请”）
		index.Fields("tenant_id", "created_by").
			StorageKey("idx_oa_biz_trip_tenant_created_by"),
	}
}
