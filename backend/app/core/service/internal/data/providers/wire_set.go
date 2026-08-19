//go:build wireinject
// +build wireinject

//go:generate go run github.com/google/wire/cmd/wire

// This file defines the dependency injection ProviderSet for the data layer and contains no business logic.
// The build tag `wireinject` excludes this source from normal `go build` and final binaries.
// Run `go generate ./...` or `go run github.com/google/wire/cmd/wire` to regenerate the Wire output (e.g. `wire_gen.go`), which will be included in final builds.
// Keep provider constructors here only; avoid init-time side effects or runtime logic in this file.

package providers

import (
	"github.com/google/wire"

	"go-wind-oa/app/core/service/internal/data"
	"go-wind-oa/app/core/service/internal/data/client"
)

// ProviderSet is the Wire provider set for data layer.
//
// OA core-service 只持有工作流引擎與站內信通知兩類數據訪問：
//   - infra：ent/redis/discovery 客戶端
//   - internal_message 倉庫（工作流通知落 cms 站內信表）
//   - workflow 倉庫（定義/實例/任務/日誌）
var ProviderSet = wire.NewSet(
	client.NewRedisClient,
	client.NewEntClient,
	client.NewDiscovery,

	data.NewInternalMessageRepo,
	data.NewInternalMessageCategoryRepo,
	data.NewInternalMessageRecipientRepo,

	data.NewWorkflowDefinitionRepo,
	data.NewWorkflowInstanceRepo,
	data.NewWorkflowTaskRepo,
	data.NewWorkflowLogRepo,

	data.NewAttendanceFenceRepo,
	data.NewAttendanceWifiRepo,
	data.NewAttendanceRecordRepo,
)
