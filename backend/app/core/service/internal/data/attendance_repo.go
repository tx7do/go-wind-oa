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
	"go-wind-oa/app/core/service/internal/data/ent/attendancerecord"
	"go-wind-oa/app/core/service/internal/data/ent/attendancesetting"
	"go-wind-oa/app/core/service/internal/data/ent/user"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

type AttendanceRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewAttendanceRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *AttendanceRepo {
	return &AttendanceRepo{
		log:       ctx.NewLoggerHelper("attendance/repo/core-service"),
		entClient: entClient,
	}
}

func dayResultToProto(s attendancerecord.DayResult) oaV1.AttendanceRecord_DayResult {
	switch s {
	case attendancerecord.DayResultNormal:
		return oaV1.AttendanceRecord_NORMAL
	case attendancerecord.DayResultLate:
		return oaV1.AttendanceRecord_LATE
	case attendancerecord.DayResultEarlyLeave:
		return oaV1.AttendanceRecord_EARLY_LEAVE
	case attendancerecord.DayResultAbsent:
		return oaV1.AttendanceRecord_ABSENT
	case attendancerecord.DayResultOnLeave:
		return oaV1.AttendanceRecord_ON_LEAVE
	default:
		return oaV1.AttendanceRecord_PENDING
	}
}

func dayResultToEntity(s oaV1.AttendanceRecord_DayResult) attendancerecord.DayResult {
	switch s {
	case oaV1.AttendanceRecord_NORMAL:
		return attendancerecord.DayResultNormal
	case oaV1.AttendanceRecord_LATE:
		return attendancerecord.DayResultLate
	case oaV1.AttendanceRecord_EARLY_LEAVE:
		return attendancerecord.DayResultEarlyLeave
	case oaV1.AttendanceRecord_ABSENT:
		return attendancerecord.DayResultAbsent
	case oaV1.AttendanceRecord_ON_LEAVE:
		return attendancerecord.DayResultOnLeave
	default:
		return attendancerecord.DayResultPending
	}
}

// AttendanceRecordToDTO 导出供 service 层复用（CheckIn 响应）。
func AttendanceRecordToDTO(e *ent.AttendanceRecord) *oaV1.AttendanceRecord {
	if e == nil {
		return nil
	}
	dto := &oaV1.AttendanceRecord{
		Id:       trans.Ptr(e.ID),
		UserId:   trans.Ptr(e.UserID),
		TenantId: e.TenantID,
	}
	workDate := e.WorkDate
	dto.WorkDate = timeutil.TimeToTimestamppb(&workDate)
	if e.CheckInAt != nil {
		dto.CheckInAt = timeutil.TimeToTimestamppb(e.CheckInAt)
		dto.CheckInLatitude = e.CheckInLatitude
		dto.CheckInLongitude = e.CheckInLongitude
		dto.CheckInWifiBssid = e.CheckInWifiBssid
	}
	if e.CheckOutAt != nil {
		dto.CheckOutAt = timeutil.TimeToTimestamppb(e.CheckOutAt)
		dto.CheckOutLatitude = e.CheckOutLatitude
		dto.CheckOutLongitude = e.CheckOutLongitude
		dto.CheckOutWifiBssid = e.CheckOutWifiBssid
	}
	if e.DayResult != nil {
		dto.DayResult = dayResultToProto(*e.DayResult).Enum()
	}
	dto.CreatedAt = timeutil.TimeToTimestamppb(e.CreatedAt)
	return dto
}

// GetByUserDate 读取用户当日记录，不存在返回 nil。
func (r *AttendanceRepo) GetByUserDate(ctx context.Context, tid, userID uint32, workDate time.Time) (*ent.AttendanceRecord, error) {
	entity, err := r.entClient.Client().AttendanceRecord.Query().
		Where(
			attendancerecord.TenantIDEQ(tid),
			attendancerecord.UserIDEQ(userID),
			attendancerecord.WorkDateEQ(workDate),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf("query attendance record failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query attendance record failed")
	}
	return entity, nil
}

// CreateCheckIn 创建当日首条签到记录（含 GPS/BSSID）。
func (r *AttendanceRepo) CreateCheckIn(
	ctx context.Context, tid, uid uint32,
	workDate, checkInAt time.Time,
	latitude, longitude float64, wifiBSSID string,
) (*ent.AttendanceRecord, error) {
	entity, err := r.entClient.Client().AttendanceRecord.Create().
		SetUserID(uid).
		SetWorkDate(workDate).
		SetCheckInAt(checkInAt).
		SetCheckInLatitude(latitude).
		SetCheckInLongitude(longitude).
		SetCheckInWifiBssid(wifiBSSID).
		SetTenantID(tid).
		SetCreatedBy(uid).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		r.log.Errorf("insert attendance record failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("insert attendance record failed")
	}
	return entity, nil
}

// SetCheckOut 写入签退并结算结果。返回后由调用方经 GetByUserDate 重读最新记录。
func (r *AttendanceRepo) SetCheckOut(
	ctx context.Context, tid, recordID uint32,
	checkOutAt time.Time,
	latitude, longitude float64, wifiBSSID string,
	result oaV1.AttendanceRecord_DayResult,
) error {
	if _, err := r.entClient.Client().AttendanceRecord.Update().
		Where(
			attendancerecord.IDEQ(recordID),
			attendancerecord.TenantIDEQ(tid),
		).
		SetCheckOutAt(checkOutAt).
		SetCheckOutLatitude(latitude).
		SetCheckOutLongitude(longitude).
		SetCheckOutWifiBssid(wifiBSSID).
		SetDayResult(dayResultToEntity(result)).
		SetUpdatedAt(time.Now()).
		Save(ctx); err != nil {
		r.log.Errorf("update check-out failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("update check-out failed")
	}
	return nil
}

// SettleMaterialize 结算物化：为无记录用户创建 ABSENT/ON_LEAVE 记录。
func (r *AttendanceRepo) SettleMaterialize(
	ctx context.Context, tid, userID uint32, workDate time.Time,
	result oaV1.AttendanceRecord_DayResult, operatorID uint32,
) error {
	if _, err := r.entClient.Client().AttendanceRecord.Create().
		SetUserID(userID).
		SetWorkDate(workDate).
		SetDayResult(dayResultToEntity(result)).
		SetTenantID(tid).
		SetCreatedBy(operatorID).
		SetCreatedAt(time.Now()).
		Save(ctx); err != nil {
		r.log.Errorf("materialize attendance record failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("materialize attendance record failed")
	}
	return nil
}

// SettlePending 为已签退但仍 PENDING 的记录补结算。
func (r *AttendanceRepo) SettlePending(
	ctx context.Context, tid, recordID uint32,
	result oaV1.AttendanceRecord_DayResult, operatorID uint32,
) error {
	if _, err := r.entClient.Client().AttendanceRecord.Update().
		Where(
			attendancerecord.IDEQ(recordID),
			attendancerecord.TenantIDEQ(tid),
		).
		SetDayResult(dayResultToEntity(result)).
		SetUpdatedBy(operatorID).
		SetUpdatedAt(time.Now()).
		Save(ctx); err != nil {
		r.log.Errorf("settle pending record failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("settle pending record failed")
	}
	return nil
}

// ListByUserRange 用户打卡记录（按日期范围，含端点）。
func (r *AttendanceRepo) ListByUserRange(ctx context.Context, tid, userID uint32, start, end time.Time) ([]*oaV1.AttendanceRecord, error) {
	entities, err := r.entClient.Client().AttendanceRecord.Query().
		Where(
			attendancerecord.TenantIDEQ(tid),
			attendancerecord.UserIDEQ(userID),
			attendancerecord.WorkDateGTE(start),
			attendancerecord.WorkDateLTE(end),
		).
		All(ctx)
	if err != nil {
		r.log.Errorf("list attendance records failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("list attendance records failed")
	}
	items := make([]*oaV1.AttendanceRecord, 0, len(entities))
	for _, e := range entities {
		items = append(items, AttendanceRecordToDTO(e))
	}
	return items, nil
}

// ListByDate 按工作日查全部（admin，可选用户过滤）。
func (r *AttendanceRepo) ListByDate(ctx context.Context, tid, userID uint32, workDate time.Time) ([]*oaV1.AttendanceRecord, error) {
	query := r.entClient.Client().AttendanceRecord.Query().
		Where(
			attendancerecord.TenantIDEQ(tid),
			attendancerecord.WorkDateEQ(workDate),
		)
	if userID != 0 {
		query = query.Where(attendancerecord.UserIDEQ(userID))
	}
	entities, err := query.All(ctx)
	if err != nil {
		r.log.Errorf("list attendance records by date failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("list attendance records failed")
	}
	items := make([]*oaV1.AttendanceRecord, 0, len(entities))
	for _, e := range entities {
		items = append(items, AttendanceRecordToDTO(e))
	}
	return items, nil
}

// ListUserIDs 结算用：租户内全部用户 ID。
func (r *AttendanceRepo) ListUserIDs(ctx context.Context, tid uint32) ([]uint32, error) {
	ids, err := r.entClient.Client().User.Query().
		Where(user.TenantIDEQ(tid)).
		IDs(ctx)
	if err != nil {
		r.log.Errorf("list user ids failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("list user ids failed")
	}
	return ids, nil
}

// ListTenantIDs 定时结算用：全部租户 ID。
func (r *AttendanceRepo) ListTenantIDs(ctx context.Context) ([]uint32, error) {
	ids, err := r.entClient.Client().Tenant.Query().
		IDs(ctx)
	if err != nil {
		r.log.Errorf("list tenant ids failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("list tenant ids failed")
	}
	return ids, nil
}

// GetSetting 读取租户考勤设置（不存在时按默认值创建）。
func (r *AttendanceRepo) GetSetting(ctx context.Context, tid uint32) (*oaV1.AttendanceSetting, error) {
	entity, err := r.entClient.Client().AttendanceSetting.Query().
		Where(attendancesetting.TenantIDEQ(tid)).
		Only(ctx)
	if ent.IsNotFound(err) {
		entity, err = r.entClient.Client().AttendanceSetting.Create().
			SetWorkStartTime("09:00").
			SetWorkEndTime("18:00").
			SetTenantID(tid).
			SetCreatedAt(time.Now()).
			Save(ctx)
	}
	if err != nil {
		r.log.Errorf("query attendance setting failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query attendance setting failed")
	}
	return &oaV1.AttendanceSetting{
		Id:            trans.Ptr(entity.ID),
		WorkStartTime: trans.Ptr(entity.WorkStartTime),
		WorkEndTime:   trans.Ptr(entity.WorkEndTime),
		TenantId:      entity.TenantID,
	}, nil
}

// UpdateSetting 更新租户考勤设置。
func (r *AttendanceRepo) UpdateSetting(ctx context.Context, tid, uid uint32, workStart, workEnd string) error {
	if _, err := r.GetSetting(ctx, tid); err != nil {
		return err
	}
	if _, err := r.entClient.Client().AttendanceSetting.Update().
		Where(attendancesetting.TenantIDEQ(tid)).
		SetWorkStartTime(workStart).
		SetWorkEndTime(workEnd).
		SetUpdatedBy(uid).
		SetUpdatedAt(time.Now()).
		Save(ctx); err != nil {
		r.log.Errorf("update attendance setting failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("update attendance setting failed")
	}
	return nil
}
