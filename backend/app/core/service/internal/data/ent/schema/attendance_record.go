package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// AttendanceRecord 考勤打卡记录。每次打卡落一条，记录打卡人、时间、提交的
// GPS 坐标、提交的 BSSID、判定结果（IN_FENCE / IN_WIFI / DENIED）。
// 由 TenantPrivacy 按租户隔离，并由 (tenant_id, created_by) 索引支撑
// "我的打卡记录"查询。
type AttendanceRecord struct{ ent.Schema }

func (AttendanceRecord) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "oa_attendance_record", Charset: "utf8mb4", Collation: "utf8mb4_bin"},
		entsql.WithComments(true),
		schema.Comment("OA考勤打卡记录表"),
	}
}

func (AttendanceRecord) Fields() []ent.Field {
	return []ent.Field{
		// 打卡判定结果。
		field.Enum("check_result").
			Comment("打卡判定结果").
			NamedValues("IN_FENCE", "IN_FENCE", "IN_WIFI", "IN_WIFI", "DENIED", "DENIED").
			Default("DENIED").
			Optional(),
		// 打卡提交的 GPS 坐标（WGS84）。
		field.Float("longitude").Comment("提交经度").Optional().Nillable(),
		field.Float("latitude").Comment("提交纬度").Optional().Nillable(),
		// 打卡提交的 BSSID（如可用）。
		field.String("bssid").Comment("提交BSSID").Optional().Nillable(),
	}
}

func (AttendanceRecord) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
		mixin.Remark{},
	}
}

func (AttendanceRecord) Edges() []ent.Edge {
	return nil
}

func (AttendanceRecord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id").
			StorageKey("idx_oa_att_rec_tenant"),
		index.Fields("tenant_id", "created_by").
			StorageKey("idx_oa_att_rec_tenant_user"),
	}
}
