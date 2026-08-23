package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// WifiFingerprint holds the schema definition for the WifiFingerprint entity.
type WifiFingerprint struct {
	ent.Schema
}

func (WifiFingerprint) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_wifi_fingerprint",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 考勤 Wi-Fi 指纹白名单表（打卡时 BSSID 命中任一条即放行）"),
	}
}

// Fields of the WifiFingerprint.
func (WifiFingerprint) Fields() []ent.Field {
	return []ent.Field{
		field.String("ssid").
			Comment("SSID（仅描述，不参与校验）").
			MaxLen(64).
			Default(""),

		field.String("bssid").
			Comment("BSSID（校验比对键）").
			MaxLen(64),
	}
}

// Mixin of the WifiFingerprint.
func (WifiFingerprint) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

func (WifiFingerprint) Indexes() []ent.Index {
	return []ent.Index{
		// 租户筛选（一租户多条指纹，非唯一）
		index.Fields("tenant_id").
			StorageKey("idx_oa_wifi_fingerprint_tenant"),
	}
}
