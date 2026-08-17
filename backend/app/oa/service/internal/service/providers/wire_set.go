//go:build wireinject
// +build wireinject

// Package providers 暴露 OA 业务层的 wire provider 集合。
package providers

import (
	"github.com/google/wire"

	"go-wind-oa/app/oa/service/internal/service"
)

// ProviderSet OA 业务层的依赖注入集合。
//
// 含两个服务构造：
//  - service.NewWorkflowService：注入四个 OA 仓库 + cms 站内信通知 gRPC 客户端，
//    实现 kratos 生成的 WorkflowServiceHTTPServer（SubmitApply / AuditTask / GetMyTasks
//    及定义管理）。
//  - service.NewAuthenticationService：注入 cms admin-service 鉴权 gRPC 客户端，
//    实现 kratos 生成的 AuthenticationServiceHTTPServer（Login / Logout /
//    RefreshToken / GenerateCaptcha / VerifyCaptcha），各方法转发至 cms 同名 RPC。
//    oa↔cms 消息翻译经 proto.Marshal/Unmarshal（两包对应消息 wire 格式逐字段对齐）。
//    鉴权操作符回填（Logout / RefreshToken）经 auth.FromContext 取 OperatorMetadata，
//    与 cms admin-service 同款流程。
//
// 无 wire.Bind。
var ProviderSet = wire.NewSet(
	service.NewWorkflowService,
	service.NewAuthenticationService,
)
