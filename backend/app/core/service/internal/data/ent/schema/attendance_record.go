package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// AttendanceRecord holds the schema definition for the AttendanceRecord entity.
type AttendanceRecord struct {
	ent.Schema
}

func (AttendanceRecord) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_attendance_record",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 考勤打卡记录表（用户 x 工作日 唯一）"),
	}
}

// Fields of the AttendanceRecord.
func (AttendanceRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("user_id").
			Comment("用户ID"),

		field.Time("work_date").
			Comment("工作日（日期，零点）"),

		field.Time("check_in_at").
			Comment("上班签到时间").
			Optional().
			Nillable(),

		field.Float("check_in_latitude").
			Comment("签到 GPS 纬度").
			Optional().
			Nillable(),

		field.Float("check_in_longitude").
			Comment("签到 GPS 经度").
			Optional().
			Nillable(),

		field.String("check_in_wifi_bssid").
			Comment("签到 Wifi BSSID").
			MaxLen(64).
			Optional().
			Nillable(),

		field.Time("check_out_at").
			Comment("下班签退时间").
			Optional().
			Nillable(),

		field.Float("check_out_latitude").
			Comment("签退 GPS 纬度").
			Optional().
			Nillable(),

		field.Float("check_out_longitude").
			Comment("签退 GPS 经度").
			Optional().
			Nillable(),

		field.String("check_out_wifi_bssid").
			Comment("签退 Wifi BSSID").
			MaxLen(64).
			Optional().
			Nillable(),

		field.Enum("day_result").
			Comment("当日结算结果").
			NamedValues(
				"Pending", "PENDING",
				"Normal", "NORMAL",
				"Late", "LATE",
				"EarlyLeave", "EARLY_LEAVE",
				"Absent", "ABSENT",
				"OnLeave", "ON_LEAVE",
			).
			Default("PENDING").
			Optional().
			Nillable(),
	}
}

// Mixin of the AttendanceRecord.
func (AttendanceRecord) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

func (AttendanceRecord) Indexes() []ent.Index {
	return []ent.Index{
		// 租户筛选
		index.Fields("tenant_id").
			StorageKey("idx_oa_attendance_tenant"),

		// 租户 + 工作日（admin 按日查全部）
		index.Fields("tenant_id", "work_date").
			StorageKey("idx_oa_attendance_tenant_work_date"),

		// 租户 + 用户 + 工作日 唯一
		index.Fields("tenant_id", "user_id", "work_date").
			Unique().
			StorageKey("uix_oa_attendance_tenant_user_date"),
	}
}
