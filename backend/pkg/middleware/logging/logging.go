package logging

// 本包原為 cms 的 API/登錄審計日誌中間件，依賴已裁剪的 audit 域。
// OA admin-service 的 REST 中間件鏈改用 kratos 自帶的
// github.com/go-kratos/kratos/v2/middleware/logging 做請求日誌，
// 不再接入本包的審計日誌中間件。此處保留 constants.go 的 header 常量，
// Server 函數留作空殼以避免外部引用斷裂。
