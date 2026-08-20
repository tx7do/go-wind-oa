package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// LeaveType holds the schema definition for the LeaveType entity.
type LeaveType struct {
	ent.Schema
}

func (LeaveType) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_leave_type",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 请假类型表（年假/病假/事假等）"),
	}
}

// Fields of the LeaveType.
func (LeaveType) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			Comment("类型代码（如 ANNUAL/SICK/PERSONAL）").
			MaxLen(64),

		field.String("name").
			Comment("类型名称").
			MaxLen(64),
	}
}

// Mixin of the LeaveType.
func (LeaveType) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
		mixin.Remark{},
	}
}

func (LeaveType) Indexes() []ent.Index {
	return []ent.Index{
		// 租户筛选
		index.Fields("tenant_id").
			StorageKey("idx_oa_leave_type_tenant"),

		// 租户内 code 唯一
		index.Fields("tenant_id", "code").
			Unique().
			StorageKey("uix_oa_leave_type_tenant_code"),
	}
}
