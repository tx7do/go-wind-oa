package service

import "math"

// haversineMeters 计算两个经纬度坐标之间的球面大圆距离（米）。
// 用于地理围栏打卡点比对。
func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMeters = 6371000.0
	lat1r := lat1 * math.Pi / 180
	lat2r := lat2 * math.Pi / 180
	dlat := (lat2 - lat1) * math.Pi / 180
	dlon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1r)*math.Cos(lat2r)*math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusMeters * c
}

// wgs84ToGcj02 将 WGS-84 坐标转换为 GCJ-02（火星坐标系）。
// 围栏按 GCJ-02 存储（高德地图圈选直接产出 GCJ-02）；
// 打卡端 GPS 为 WGS-84，判定前先经此函数转换为 GCJ-02 再与围栏比对，
// 保证两侧坐标系一致。算法为公开的确定性偏移公式。
func wgs84ToGcj02(wgsLat, wgsLon float64) (gcjLat, gcjLon float64) {
	if outOfChina(wgsLat, wgsLon) {
		return wgsLat, wgsLon
	}
	dLat := transformLat(wgsLon-105.0, wgsLat-35.0)
	dLon := transformLng(wgsLon-105.0, wgsLat-35.0)
	radLat := wgsLat / 180.0 * math.Pi
	magic := math.Sin(radLat)
	magic = 1 - ee*magic*magic
	sqrtMagic := math.Sqrt(magic)
	dLat = (dLat * 180.0) / ((a * (1 - ee)) / (magic * sqrtMagic) * math.Pi)
	dLon = (dLon * 180.0) / (a / sqrtMagic * math.Cos(radLat) * math.Pi)
	gcjLat = wgsLat + dLat
	gcjLon = wgsLon + dLon
	return gcjLat, gcjLon
}

func outOfChina(lat, lon float64) bool {
	return lon < 72.004 || lon > 137.8347 || lat < 0.8293 || lat > 55.8271
}

func transformLat(x, y float64) float64 {
	ret := -100.0 + 2.0*x + 3.0*y + 0.2*y*y + 0.1*x*y + 0.2*math.Sqrt(math.Abs(x))
	ret += (20.0*math.Sin(6.0*x*math.Pi) + 20.0*math.Sin(2.0*x*math.Pi)) * 2.0 / 3.0
	ret += (20.0*math.Sin(y*math.Pi) + 40.0*math.Sin(y/3.0*math.Pi)) * 2.0 / 3.0
	ret += (160.0*math.Sin(y/12.0*math.Pi) + 320*math.Sin(y*math.Pi/30.0)) * 2.0 / 3.0
	return ret
}

func transformLng(x, y float64) float64 {
	ret := 300.0 + x + 2.0*y + 0.1*x*x + 0.1*x*y + 0.1*math.Sqrt(math.Abs(x))
	ret += (20.0*math.Sin(6.0*x*math.Pi) + 20.0*math.Sin(2.0*x*math.Pi)) * 2.0 / 3.0
	ret += (20.0*math.Sin(x*math.Pi) + 40.0*math.Sin(x/3.0*math.Pi)) * 2.0 / 3.0
	ret += (150.0*math.Sin(x/12.0*math.Pi) + 300.0*math.Sin(x/30.0*math.Pi)) * 2.0 / 3.0
	return ret
}

const (
	a  = 6378245.0
	ee = 0.00669342162296594
)
