package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// AttendanceWifi 考勤 Wi-Fi 指纹白名单。每条记录定义一个允许打卡的
// SSID/BSSID。移动端打卡时上传当前连接 Wi-Fi 的 BSSID，服务端比对白名单。
// 仅本租户指纹可达（TenantPrivacy 隔离）。
type AttendanceWifi struct{ ent.Schema }

func (AttendanceWifi) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "oa_attendance_wifi", Charset: "utf8mb4", Collation: "utf8mb4_bin"},
		entsql.WithComments(true),
		schema.Comment("OA考勤Wi-Fi指纹白名单表"),
	}
}

func (AttendanceWifi) Fields() []ent.Field {
	return []ent.Field{
		field.String("ssid").Comment("允许的SSID").Optional().Nillable(),
		field.String("bssid").Comment("允许的BSSID").Optional().Nillable(),
	}
}

func (AttendanceWifi) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
		mixin.Remark{},
	}
}

func (AttendanceWifi) Edges() []ent.Edge {
	return nil
}

func (AttendanceWifi) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id").
			StorageKey("idx_oa_att_wifi_tenant"),
	}
}
