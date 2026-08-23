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
	"go-wind-oa/app/core/service/internal/data/ent/geofence"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

type GeofenceRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewGeofenceRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *GeofenceRepo {
	return &GeofenceRepo{
		log:       ctx.NewLoggerHelper("geofence/repo/core-service"),
		entClient: entClient,
	}
}

// ListByTenant 列出本租户全部地理围栏。
func (r *GeofenceRepo) ListByTenant(ctx context.Context, tid uint32) ([]*ent.Geofence, error) {
	entities, err := r.entClient.Client().Geofence.Query().
		Where(geofence.TenantIDEQ(tid)).
		All(ctx)
	if err != nil {
		r.log.Errorf("list geofences failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("list geofences failed")
	}
	return entities, nil
}

// ListByTenantDTO 同上，转 DTO。
func (r *GeofenceRepo) ListByTenantDTO(ctx context.Context, tid uint32) ([]*oaV1.Geofence, error) {
	entities, err := r.ListByTenant(ctx, tid)
	if err != nil {
		return nil, err
	}
	items := make([]*oaV1.Geofence, 0, len(entities))
	for _, e := range entities {
		items = append(items, geofenceToDTO(e))
	}
	return items, nil
}

// Upsert 按 id 存在则覆盖，否则创建。
func (r *GeofenceRepo) Upsert(ctx context.Context, tid, uid uint32, id uint32, name string, latitude, longitude, radius float64) error {
	if id == 0 {
		if _, err := r.entClient.Client().Geofence.Create().
			SetName(name).
			SetLatitude(latitude).
			SetLongitude(longitude).
			SetRadiusMeters(radius).
			SetTenantID(tid).
			SetCreatedBy(uid).
			SetCreatedAt(time.Now()).
			Save(ctx); err != nil {
			r.log.Errorf("insert geofence failed: %s", err.Error())
			return oaV1.ErrorInternalServerError("insert geofence failed")
		}
		return nil
	}
	if _, err := r.entClient.Client().Geofence.Update().
		Where(geofence.IDEQ(id), geofence.TenantIDEQ(tid)).
		SetName(name).
		SetLatitude(latitude).
		SetLongitude(longitude).
		SetRadiusMeters(radius).
		SetUpdatedBy(uid).
		SetUpdatedAt(time.Now()).
		Save(ctx); err != nil {
		r.log.Errorf("update geofence failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("update geofence failed")
	}
	return nil
}

// Delete 删除地理围栏。
func (r *GeofenceRepo) Delete(ctx context.Context, tid, id uint32) error {
	if _, err := r.entClient.Client().Geofence.Delete().
		Where(geofence.IDEQ(id), geofence.TenantIDEQ(tid)).
		Exec(ctx); err != nil {
		r.log.Errorf("delete geofence failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("delete geofence failed")
	}
	return nil
}

func geofenceToDTO(e *ent.Geofence) *oaV1.Geofence {
	dto := &oaV1.Geofence{
		Id:           trans.Ptr(e.ID),
		Name:         trans.Ptr(e.Name),
		Latitude:     trans.Ptr(e.Latitude),
		Longitude:    trans.Ptr(e.Longitude),
		RadiusMeters: trans.Ptr(e.RadiusMeters),
	}
	if e.TenantID != nil {
		dto.TenantId = trans.Ptr(*e.TenantID)
	}
	dto.CreatedAt = timeutil.TimeToTimestamppb(e.CreatedAt)
	return dto
}
