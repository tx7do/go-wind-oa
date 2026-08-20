package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// SealApplication holds the schema definition for the SealApplication entity.
type SealApplication struct {
	ent.Schema
}

func (SealApplication) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "oa_seal_application",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OA 用印申请单表"),
	}
}

// Fields of the SealApplication.
func (SealApplication) Fields() []ent.Field {
	return []ent.Field{
		field.String("purpose").
			Comment("用印事由").
			MaxLen(512).
			Default(""),

		field.Enum("seal_type").
			Comment("印章类型").
			NamedValues(
				"OfficialSeal", "OFFICIAL_SEAL",
				"ContractSeal", "CONTRACT_SEAL",
				"FinanceSeal", "FINANCE_SEAL",
				"LegalSeal", "LEGAL_SEAL",
			).
			Default("OFFICIAL_SEAL").
			Optional().
			Nillable(),

		field.Int32("file_count").
			Comment("用印文件份数").
			Default(0),

		field.String("recipient").
			Comment("收件方").
			MaxLen(256).
			Default(""),

		field.Enum("seal_status").
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

// Mixin of the SealApplication.
func (SealApplication) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

func (SealApplication) Indexes() []ent.Index {
	return []ent.Index{
		// 租户筛选
		index.Fields("tenant_id").
			StorageKey("idx_oa_seal_app_tenant"),

		// 租户 + 申请人（“我的申请”）
		index.Fields("tenant_id", "created_by").
			StorageKey("idx_oa_seal_app_tenant_created_by"),
	}
}
