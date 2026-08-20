package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// LeaveApplication holds the schema definition for the LeaveApplication entity.
type LeaveApplication struct {
	ent.Schema
}

func (LeaveApplication) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_leave_application",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 请假申请单表"),
	}
}

// Fields of the LeaveApplication.
func (LeaveApplication) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("leave_type_id").
			Comment("请假类型ID"),

		field.Time("start_date").
			Comment("开始日期"),

		field.Time("end_date").
			Comment("结束日期（含）"),

		field.Float("days").
			Comment("请假天数（含首尾）").
			Default(0),

		field.String("reason").
			Comment("请假事由").
			MaxLen(512).
			Default(""),

		field.Uint8("start_half").
			Comment("开始半日（0=AM 上午起，1=PM 下午起）").
			Default(0),

		field.Uint8("end_half").
			Comment("结束半日（0=AM 上午止，1=PM 下午止）").
			Default(1),

		field.Enum("leave_status").
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

// Mixin of the LeaveApplication.
func (LeaveApplication) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

func (LeaveApplication) Indexes() []ent.Index {
	return []ent.Index{
		// 租户筛选
		index.Fields("tenant_id").
			StorageKey("idx_oa_leave_app_tenant"),

		// 租户 + 申请人（“我的申请”）
		index.Fields("tenant_id", "created_by").
			StorageKey("idx_oa_leave_app_tenant_created_by"),

		// 租户 + 日期范围扫描（考勤结算判断请假覆盖）
		index.Fields("tenant_id", "start_date", "end_date").
			StorageKey("idx_oa_leave_app_tenant_date_range"),
	}
}
