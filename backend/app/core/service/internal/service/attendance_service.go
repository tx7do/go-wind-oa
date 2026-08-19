package service

import (
	"context"
	"math"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-oa/app/core/service/internal/data"
	oav1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// AttendanceService 考勤服务。
//
// CheckIn：移动端打卡，上传 GPS 坐标与 BSSID，服务端按本租户围栏库与
// Wi-Fi 指纹库判定，并落打卡记录。围栏判定走 Haversine 球面距离与围栏半径
// 比较；Wi-Fi 判定走 BSSID 精确匹配。任一通过即记 IN_FENCE / IN_WIFI，
// 否则 DENIED。无论结果均落记录，便于审计。
//
// 围栏库 / Wi-Fi 指纹库的 CRUD 供管理端维护，均经 TenantPrivacy 租户隔离。
type AttendanceService struct {
	oav1.AttendanceServiceServer
	log *log.Helper

	fenceRepo  *data.AttendanceFenceRepo
	wifiRepo   *data.AttendanceWifiRepo
	recordRepo *data.AttendanceRecordRepo
}

func NewAttendanceService(
	ctx *bootstrap.Context,
	fenceRepo *data.AttendanceFenceRepo,
	wifiRepo *data.AttendanceWifiRepo,
	recordRepo *data.AttendanceRecordRepo,
) *AttendanceService {
	return &AttendanceService{
		log:        ctx.NewLoggerHelper("attendance/service/core-service"),
		fenceRepo:  fenceRepo,
		wifiRepo:   wifiRepo,
		recordRepo: recordRepo,
	}
}

// haversine 返回兩點間球面距離（米）。WGS84 椭球近似為球體，誤差可接受。
func haversine(lon1, lat1, lon2, lat2 float64) float64 {
	const r = 6371000.0 // 地球平均半徑（米）
	la1 := lat1 * math.Pi / 180
	la2 := lat2 * math.Pi / 180
	dla := (lat2 - lat1) * math.Pi / 180
	dlo := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dla/2)*math.Sin(dla/2) +
		math.Cos(la1)*math.Cos(la2)*math.Sin(dlo/2)*math.Sin(dlo/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return r * c
}

// CheckIn 打卡判定。
func (s *AttendanceService) CheckIn(ctx context.Context, req *oav1.CheckInRequest) (*oav1.CheckInResponse, error) {
	if req == nil {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	tenantID, userID, ok := callerFromContext(ctx)
	if !ok {
		return nil, oav1.ErrorBadRequest("missing viewer context")
	}

	result := oav1.CheckInResponse_DENIED
	msg := "denied: not in fence nor wifi whitelist"

	// 围栏判定：遍历本租户围栏，Haversine 距离 ≤ 半径即 IN_FENCE。
	lon := req.GetLongitude()
	lat := req.GetLatitude()
	if lon != 0 && lat != 0 {
		fences, err := s.fenceRepo.ListAllForCheckIn(ctx)
		if err == nil {
			for _, f := range fences {
				if f.Longitude == nil || f.Latitude == nil || f.Radius == nil {
					continue
				}
				dist := haversine(lon, lat, *f.Longitude, *f.Latitude)
				if dist <= *f.Radius {
					result = oav1.CheckInResponse_IN_FENCE
					msg = "in fence"
					break
				}
			}
		}
	}

	// Wi-Fi 判定：BSSID 精确匹配本租户白名单。
	if result == oav1.CheckInResponse_DENIED && req.GetBssid() != "" {
		wifis, err := s.wifiRepo.ListAllForCheckIn(ctx)
		if err == nil {
			for _, w := range wifis {
				if w.Bssid != nil && *w.Bssid == req.GetBssid() {
					result = oav1.CheckInResponse_IN_WIFI
					msg = "in wifi whitelist"
					break
				}
			}
		}
	}

	// 落打卡记录（无论结果）。
	var lonPtr, latPtr *float64
	if lon != 0 && lat != 0 {
		lonPtr = &lon
		latPtr = &lat
	}
	var bssidPtr *string
	if b := req.GetBssid(); b != "" {
		bssidPtr = &b
	}
	_ = s.recordRepo.CreateCheckInRecord(ctx, tenantID, userID, lonPtr, latPtr, bssidPtr, result)

	resultVal := result
	return &oav1.CheckInResponse{
		CheckResult: &resultVal,
		Message:     &msg,
	}, nil
}

// ---- 围栏库 CRUD ----

func (s *AttendanceService) CreateAttendanceFence(ctx context.Context, req *oav1.CreateAttendanceFenceRequest) (*oav1.AttendanceFence, error) {
	if req == nil || req.Data == nil {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	tenantID, userID, ok := callerFromContext(ctx)
	if !ok {
		return nil, oav1.ErrorBadRequest("missing viewer context")
	}
	req.Data.TenantId = &tenantID
	req.Data.CreatedBy = &userID
	return s.fenceRepo.Create(ctx, req.Data)
}

func (s *AttendanceService) ListAttendanceFence(ctx context.Context, req *oav1.ListAttendanceFenceRequest) (*oav1.ListAttendanceFenceResponse, error) {
	if req == nil || req.Paging == nil {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	if _, _, ok := callerFromContext(ctx); !ok {
		return nil, oav1.ErrorBadRequest("missing viewer context")
	}
	return s.fenceRepo.List(ctx, req.Paging)
}

func (s *AttendanceService) UpdateAttendanceFence(ctx context.Context, req *oav1.UpdateAttendanceFenceRequest) (*oav1.AttendanceFence, error) {
	if req == nil || req.Id == 0 {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	if _, _, ok := callerFromContext(ctx); !ok {
		return nil, oav1.ErrorBadRequest("missing viewer context")
	}
	return s.fenceRepo.Update(ctx, req)
}

func (s *AttendanceService) DeleteAttendanceFence(ctx context.Context, req *oav1.DeleteAttendanceFenceRequest) (*emptypb.Empty, error) {
	if req == nil || req.Id == 0 {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	if _, _, ok := callerFromContext(ctx); !ok {
		return nil, oav1.ErrorBadRequest("missing viewer context")
	}
	if err := s.fenceRepo.Delete(ctx, req.Id); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// ---- Wi-Fi 指纹库 CRUD ----

func (s *AttendanceService) CreateAttendanceWifi(ctx context.Context, req *oav1.CreateAttendanceWifiRequest) (*oav1.AttendanceWifi, error) {
	if req == nil || req.Data == nil {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	tenantID, userID, ok := callerFromContext(ctx)
	if !ok {
		return nil, oav1.ErrorBadRequest("missing viewer context")
	}
	req.Data.TenantId = &tenantID
	req.Data.CreatedBy = &userID
	return s.wifiRepo.Create(ctx, req.Data)
}

func (s *AttendanceService) ListAttendanceWifi(ctx context.Context, req *oav1.ListAttendanceWifiRequest) (*oav1.ListAttendanceWifiResponse, error) {
	if req == nil || req.Paging == nil {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	if _, _, ok := callerFromContext(ctx); !ok {
		return nil, oav1.ErrorBadRequest("missing viewer context")
	}
	return s.wifiRepo.List(ctx, req.Paging)
}

func (s *AttendanceService) DeleteAttendanceWifi(ctx context.Context, req *oav1.DeleteAttendanceWifiRequest) (*emptypb.Empty, error) {
	if req == nil || req.Id == 0 {
		return nil, oav1.ErrorBadRequest("invalid parameter")
	}
	if _, _, ok := callerFromContext(ctx); !ok {
		return nil, oav1.ErrorBadRequest("missing viewer context")
	}
	if err := s.wifiRepo.Delete(ctx, req.Id); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
