package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-oa/app/core/service/internal/data/ent"
	"go-wind-oa/app/core/service/internal/data/ent/wififingerprint"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

type WifiFingerprintRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewWifiFingerprintRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *WifiFingerprintRepo {
	return &WifiFingerprintRepo{
		log:       ctx.NewLoggerHelper("wifi-fingerprint/repo/core-service"),
		entClient: entClient,
	}
}

// ListByTenant 列出本租户全部 Wi-Fi 指纹白名单。
func (r *WifiFingerprintRepo) ListByTenant(ctx context.Context, tid uint32) ([]*ent.WifiFingerprint, error) {
	entities, err := r.entClient.Client().WifiFingerprint.Query().
		Where(wififingerprint.TenantIDEQ(tid)).
		All(ctx)
	if err != nil {
		r.log.Errorf("list wifi fingerprints failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("list wifi fingerprints failed")
	}
	return entities, nil
}

// ListByTenantDTO 同上，转 DTO。
func (r *WifiFingerprintRepo) ListByTenantDTO(ctx context.Context, tid uint32) ([]*oaV1.WifiFingerprint, error) {
	entities, err := r.ListByTenant(ctx, tid)
	if err != nil {
		return nil, err
	}
	items := make([]*oaV1.WifiFingerprint, 0, len(entities))
	for _, e := range entities {
		items = append(items, wifiFingerprintToDTO(e))
	}
	return items, nil
}

// Upsert 按 id 存在则覆盖，否则创建。
func (r *WifiFingerprintRepo) Upsert(ctx context.Context, tid, uid uint32, id uint32, ssid, bssid string) error {
	if id == 0 {
		if _, err := r.entClient.Client().WifiFingerprint.Create().
			SetSsid(ssid).
			SetBssid(bssid).
			SetTenantID(tid).
			SetCreatedBy(uid).
			SetCreatedAt(time.Now()).
			Save(ctx); err != nil {
			r.log.Errorf("insert wifi fingerprint failed: %s", err.Error())
			return oaV1.ErrorInternalServerError("insert wifi fingerprint failed")
		}
		return nil
	}
	if _, err := r.entClient.Client().WifiFingerprint.Update().
		Where(wififingerprint.IDEQ(id), wififingerprint.TenantIDEQ(tid)).
		SetSsid(ssid).
		SetBssid(bssid).
		SetUpdatedBy(uid).
		SetUpdatedAt(time.Now()).
		Save(ctx); err != nil {
		r.log.Errorf("update wifi fingerprint failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("update wifi fingerprint failed")
	}
	return nil
}

// Delete 删除 Wi-Fi 指纹白名单。
func (r *WifiFingerprintRepo) Delete(ctx context.Context, tid, id uint32) error {
	if _, err := r.entClient.Client().WifiFingerprint.Delete().
		Where(wififingerprint.IDEQ(id), wififingerprint.TenantIDEQ(tid)).
		Exec(ctx); err != nil {
		r.log.Errorf("delete wifi fingerprint failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("delete wifi fingerprint failed")
	}
	return nil
}

func wifiFingerprintToDTO(e *ent.WifiFingerprint) *oaV1.WifiFingerprint {
	dto := &oaV1.WifiFingerprint{
		Id:    trans.Ptr(e.ID),
		Ssid:  trans.Ptr(e.Ssid),
		Bssid: trans.Ptr(e.Bssid),
	}
	if e.TenantID != nil {
		dto.TenantId = trans.Ptr(*e.TenantID)
	}
	dto.CreatedAt = timeutil.TimeToTimestamppb(e.CreatedAt)
	return dto
}
