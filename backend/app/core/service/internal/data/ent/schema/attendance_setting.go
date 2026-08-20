package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// AttendanceSetting holds the schema definition for the AttendanceSetting entity.
type AttendanceSetting struct {
	ent.Schema
}

func (AttendanceSetting) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_attendance_setting",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 考勤设置表（每租户一行）"),
	}
}

// Fields of the AttendanceSetting.
func (AttendanceSetting) Fields() []ent.Field {
	return []ent.Field{
		field.String("work_start_time").
			Comment("上班时间（HH:MM），晚于记迟到").
			MaxLen(8).
			Default("09:00"),

		field.String("work_end_time").
			Comment("下班时间（HH:MM），早于记早退").
			MaxLen(8).
			Default("18:00"),
	}
}

// Mixin of the AttendanceSetting.
func (AttendanceSetting) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

func (AttendanceSetting) Indexes() []ent.Index {
	return []ent.Index{
		// 租户唯一（每租户一行）
		index.Fields("tenant_id").
			Unique().
			StorageKey("uix_oa_attendance_setting_tenant"),
	}
}
