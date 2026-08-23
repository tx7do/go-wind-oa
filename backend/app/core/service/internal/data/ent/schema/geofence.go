package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// Geofence holds the schema definition for the Geofence entity.
type Geofence struct {
	ent.Schema
}

func (Geofence) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_geofence",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 考勤地理围栏表（每租户可多条；打卡点落在任一围栏半径内即放行）"),
	}
}

// Fields of the Geofence.
func (Geofence) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("围栏名称（如 总部/分公司A）").
			MaxLen(64).
			Default(""),

		field.Float("latitude").
			Comment("围栏中心纬度"),

		field.Float("longitude").
			Comment("围栏中心经度"),

		field.Float("radius_meters").
			Comment("围栏半径（米）"),
	}
}

// Mixin of the Geofence.
func (Geofence) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

func (Geofence) Indexes() []ent.Index {
	return []ent.Index{
		// 租户筛选（一租户多条围栏，非唯一）
		index.Fields("tenant_id").
			StorageKey("idx_oa_geofence_tenant"),
	}
}
