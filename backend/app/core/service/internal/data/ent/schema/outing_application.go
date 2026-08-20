package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// OutingApplication holds the schema definition for the OutingApplication entity.
type OutingApplication struct {
	ent.Schema
}

func (OutingApplication) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_outing_application",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 外出申请单表"),
	}
}

// Fields of the OutingApplication.
func (OutingApplication) Fields() []ent.Field {
	return []ent.Field{
		field.String("reason").
			Comment("外出事由").
			MaxLen(512).
			Default(""),

		field.String("destination").
			Comment("外出目的地").
			MaxLen(256).
			Default(""),

		field.Time("start_time").
			Comment("外出开始时间"),

		field.Time("end_time").
			Comment("外出结束时间"),

		field.Enum("outing_status").
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

// Mixin of the OutingApplication.
func (OutingApplication) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

func (OutingApplication) Indexes() []ent.Index {
	return []ent.Index{
		// 租户筛选
		index.Fields("tenant_id").
			StorageKey("idx_oa_outing_tenant"),

		// 租户 + 申请人（“我的申请”）
		index.Fields("tenant_id", "created_by").
			StorageKey("idx_oa_outing_tenant_created_by"),
	}
}
