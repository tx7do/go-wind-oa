package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// AttendanceFence 考勤地理围栏。每条记录定义一个圆形围栏（中心经纬度 + 半径，
// 米）。移动端打卡时上传当前 GPS 坐标，服务端按 Haversine 距离判定是否落入
// 任一围栏内。仅本租户围栏可达（TenantPrivacy 隔离）。
type AttendanceFence struct{ ent.Schema }

func (AttendanceFence) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "oa_attendance_fence", Charset: "utf8mb4", Collation: "utf8mb4_bin"},
		entsql.WithComments(true),
		schema.Comment("OA考勤地理围栏表"),
	}
}

func (AttendanceFence) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").Comment("围栏名称").Optional().Nillable(),
		// 围栏中心点经纬度（WGS84）。
		field.Float("longitude").Comment("中心经度").Optional().Nillable(),
		field.Float("latitude").Comment("中心纬度").Optional().Nillable(),
		// 围栏半径，单位米。
		field.Float("radius").Comment("围栏半径(米)").Optional().Nillable(),
	}
}

func (AttendanceFence) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
		mixin.Remark{},
	}
}

func (AttendanceFence) Edges() []ent.Edge {
	return nil
}

func (AttendanceFence) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id").
			StorageKey("idx_oa_att_fence_tenant"),
	}
}
