package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// OvertimeApplication holds the schema definition for the OvertimeApplication entity.
type OvertimeApplication struct {
	ent.Schema
}

func (OvertimeApplication) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_overtime_application",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 加班申请单表"),
	}
}

// Fields of the OvertimeApplication.
func (OvertimeApplication) Fields() []ent.Field {
	return []ent.Field{
		field.String("reason").
			Comment("加班事由").
			MaxLen(512).
			Default(""),

		field.Time("start_time").
			Comment("加班开始时间"),

		field.Time("end_time").
			Comment("加班结束时间"),

		field.Enum("compensation_type").
			Comment("补偿方式").
			NamedValues(
				"CompLeave", "COMP_LEAVE",
				"OvertimePay", "OVERTIME_PAY",
			).
			Default("COMP_LEAVE").
			Optional().
			Nillable(),

		field.Enum("overtime_status").
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

// Mixin of the OvertimeApplication.
func (OvertimeApplication) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

func (OvertimeApplication) Indexes() []ent.Index {
	return []ent.Index{
		// 租户筛选
		index.Fields("tenant_id").
			StorageKey("idx_oa_overtime_tenant"),

		// 租户 + 申请人（“我的申请”）
		index.Fields("tenant_id", "created_by").
			StorageKey("idx_oa_overtime_tenant_created_by"),
	}
}
