package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-oa/app/core/service/internal/data/ent"
	"go-wind-oa/app/core/service/internal/data/ent/attendancerecord"
	"go-wind-oa/app/core/service/internal/data/ent/attendancefence"
	"go-wind-oa/app/core/service/internal/data/ent/attendancewifi"
	"go-wind-oa/app/core/service/internal/data/ent/predicate"

	oav1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// ==============================
// 围栏库
// ==============================

type AttendanceFenceRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[oav1.AttendanceFence, ent.AttendanceFence]

	repository *entCrud.Repository[
		ent.AttendanceFenceQuery, ent.AttendanceFenceSelect,
		ent.AttendanceFenceCreate, ent.AttendanceFenceCreateBulk,
		ent.AttendanceFenceUpdate, ent.AttendanceFenceUpdateOne,
		ent.AttendanceFenceDelete,
		predicate.AttendanceFence,
		oav1.AttendanceFence, ent.AttendanceFence,
	]
}

func NewAttendanceFenceRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *AttendanceFenceRepo {
	repo := &AttendanceFenceRepo{
		log:       ctx.NewLoggerHelper("attendance-fence/repo/oa-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[oav1.AttendanceFence, ent.AttendanceFence](),
	}
	repo.init()
	return repo
}

func (r *AttendanceFenceRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.AttendanceFenceQuery, ent.AttendanceFenceSelect,
		ent.AttendanceFenceCreate, ent.AttendanceFenceCreateBulk,
		ent.AttendanceFenceUpdate, ent.AttendanceFenceUpdateOne,
		ent.AttendanceFenceDelete,
		predicate.AttendanceFence,
		oav1.AttendanceFence, ent.AttendanceFence,
	](r.mapper)
	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *AttendanceFenceRepo) Create(ctx context.Context, req *oav1.AttendanceFence) (*oav1.AttendanceFence, error) {
	if req == nil {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	builder := r.entClient.Client().AttendanceFence.Create().
		SetNillableTenantID(req.TenantId).
		SetNillableCreatedBy(req.CreatedBy).
		SetNillableName(req.Name).
		SetNillableLongitude(req.Longitude).
		SetNillableLatitude(req.Latitude).
		SetNillableRadius(req.Radius).
		SetCreatedAt(time.Now())

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert attendance fence failed: %s", err)
		return nil, oav1.ErrorInternalError("insert attendance fence failed")
	}
	return r.mapper.ToDTO(entity), nil
}

func (r *AttendanceFenceRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*oav1.ListAttendanceFenceResponse, error) {
	if req == nil {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	builder := r.entClient.Client().AttendanceFence.Query()
	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &oav1.ListAttendanceFenceResponse{Total: 0, Items: nil}, nil
	}
	return &oav1.ListAttendanceFenceResponse{Total: ret.Total, Items: ret.Items}, nil
}

func (r *AttendanceFenceRepo) Update(ctx context.Context, req *oav1.UpdateAttendanceFenceRequest) (*oav1.AttendanceFence, error) {
	if req == nil || req.Id == 0 {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)

	builder := r.entClient.Client().AttendanceFence.UpdateOneID(req.Id).
		Where(attendancefence.IDEQ(req.Id))
	if hasTenant {
		builder.Where(attendancefence.TenantIDEQ(tid))
	}
	builder.SetNillableName(req.Data.Name).
		SetNillableLongitude(req.Data.Longitude).
		SetNillableLatitude(req.Data.Latitude).
		SetNillableRadius(req.Data.Radius).
		SetUpdatedAt(time.Now())
	if hasUser {
		builder.SetUpdatedBy(callerUserID)
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, oav1.ErrorNotFound("attendance fence not found")
		}
		r.log.Errorf("update attendance fence failed: %s", err)
		return nil, oav1.ErrorInternalError("update attendance fence failed")
	}
	return r.mapper.ToDTO(entity), nil
}

func (r *AttendanceFenceRepo) Delete(ctx context.Context, id uint32) error {
	if id == 0 {
		return oav1.ErrorBadRequest("invalid parameter")
	}
	tid, hasTenant := maybeTenantFromViewer(ctx)
	preds := []predicate.AttendanceFence{attendancefence.IDEQ(id)}
	if hasTenant {
		preds = append(preds, attendancefence.TenantIDEQ(tid))
	}
	if _, err := r.entClient.Client().AttendanceFence.Delete().
		Where(preds...).
		Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return oav1.ErrorNotFound("attendance fence not found")
		}
		r.log.Errorf("delete attendance fence failed: %s", err)
		return oav1.ErrorInternalError("delete attendance fence failed")
	}
	return nil
}

// ListAllForCheckIn 取本租户全部围栏（tenant 由 TenantPrivacy 按 viewer 自动隔离），
// 供 CheckIn 距离判定遍历。返回 entity 列表（含 Longitude/Latitude/Radius）。
func (r *AttendanceFenceRepo) ListAllForCheckIn(ctx context.Context) ([]*ent.AttendanceFence, error) {
	return r.entClient.Client().AttendanceFence.Query().All(ctx)
}

// ==============================
// Wi-Fi 指纹库
// ==============================

type AttendanceWifiRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[oav1.AttendanceWifi, ent.AttendanceWifi]

	repository *entCrud.Repository[
		ent.AttendanceWifiQuery, ent.AttendanceWifiSelect,
		ent.AttendanceWifiCreate, ent.AttendanceWifiCreateBulk,
		ent.AttendanceWifiUpdate, ent.AttendanceWifiUpdateOne,
		ent.AttendanceWifiDelete,
		predicate.AttendanceWifi,
		oav1.AttendanceWifi, ent.AttendanceWifi,
	]
}

func NewAttendanceWifiRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *AttendanceWifiRepo {
	repo := &AttendanceWifiRepo{
		log:       ctx.NewLoggerHelper("attendance-wifi/repo/oa-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[oav1.AttendanceWifi, ent.AttendanceWifi](),
	}
	repo.init()
	return repo
}

func (r *AttendanceWifiRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.AttendanceWifiQuery, ent.AttendanceWifiSelect,
		ent.AttendanceWifiCreate, ent.AttendanceWifiCreateBulk,
		ent.AttendanceWifiUpdate, ent.AttendanceWifiUpdateOne,
		ent.AttendanceWifiDelete,
		predicate.AttendanceWifi,
		oav1.AttendanceWifi, ent.AttendanceWifi,
	](r.mapper)
	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *AttendanceWifiRepo) Create(ctx context.Context, req *oav1.AttendanceWifi) (*oav1.AttendanceWifi, error) {
	if req == nil {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	builder := r.entClient.Client().AttendanceWifi.Create().
		SetNillableTenantID(req.TenantId).
		SetNillableCreatedBy(req.CreatedBy).
		SetNillableSsid(req.Ssid).
		SetNillableBssid(req.Bssid).
		SetCreatedAt(time.Now())

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert attendance wifi failed: %s", err)
		return nil, oav1.ErrorInternalError("insert attendance wifi failed")
	}
	return r.mapper.ToDTO(entity), nil
}

func (r *AttendanceWifiRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*oav1.ListAttendanceWifiResponse, error) {
	if req == nil {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	builder := r.entClient.Client().AttendanceWifi.Query()
	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &oav1.ListAttendanceWifiResponse{Total: 0, Items: nil}, nil
	}
	return &oav1.ListAttendanceWifiResponse{Total: ret.Total, Items: ret.Items}, nil
}

func (r *AttendanceWifiRepo) Delete(ctx context.Context, id uint32) error {
	if id == 0 {
		return oav1.ErrorBadRequest("invalid parameter")
	}
	tid, hasTenant := maybeTenantFromViewer(ctx)
	preds := []predicate.AttendanceWifi{attendancewifi.IDEQ(id)}
	if hasTenant {
		preds = append(preds, attendancewifi.TenantIDEQ(tid))
	}
	if _, err := r.entClient.Client().AttendanceWifi.Delete().
		Where(preds...).
		Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return oav1.ErrorNotFound("attendance wifi not found")
		}
		r.log.Errorf("delete attendance wifi failed: %s", err)
		return oav1.ErrorInternalError("delete attendance wifi failed")
	}
	return nil
}

// ListAllForCheckIn 取本租户全部 Wi-Fi 指纹白名单（tenant 由 TenantPrivacy 按
// viewer 自动隔离），供 CheckIn BSSID 匹配遍历。返回 entity 列表（含 Bssid）。
func (r *AttendanceWifiRepo) ListAllForCheckIn(ctx context.Context) ([]*ent.AttendanceWifi, error) {
	return r.entClient.Client().AttendanceWifi.Query().All(ctx)
}

// ==============================
// 打卡记录（内部表，无 DTO 映射器）
// ==============================

type AttendanceRecordRepo struct {
	entClient  *entCrud.EntClient[*ent.Client]
	log        *log.Helper
	checkResultConverter *mapper.EnumTypeConverter[oav1.CheckInResponse_CheckResult, attendancerecord.CheckResult]
}

func NewAttendanceRecordRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *AttendanceRecordRepo {
	return &AttendanceRecordRepo{
		log:                  ctx.NewLoggerHelper("attendance-record/repo/oa-service"),
		entClient:            entClient,
		checkResultConverter: mapper.NewEnumTypeConverter[oav1.CheckInResponse_CheckResult, attendancerecord.CheckResult](oav1.CheckInResponse_CheckResult_name, oav1.CheckInResponse_CheckResult_value),
	}
}

// CreateCheckInRecord 落一条打卡记录。由 AttendanceService.CheckIn 在判定完成后调用。
// checkResult 为 DTO 侧判定枚举，经 converter 转为实体侧枚举落盘。
func (r *AttendanceRecordRepo) CreateCheckInRecord(
	ctx context.Context,
	tenantID, userID uint32,
	longitude, latitude *float64,
	bssid *string,
	checkResult oav1.CheckInResponse_CheckResult,
) error {
	entityResult := r.checkResultConverter.ToEntity(&checkResult)
	builder := r.entClient.Client().AttendanceRecord.Create().
		SetNillableTenantID(&tenantID).
		SetNillableCreatedBy(&userID).
		SetNillableLongitude(longitude).
		SetNillableLatitude(latitude).
		SetNillableBssid(bssid).
		SetCheckResult(*entityResult).
		SetCreatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("insert attendance record failed: %s", err)
		return oav1.ErrorInternalError("insert attendance record failed")
	}
	return nil
}
