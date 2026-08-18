package data

import (
	"context"

	"github.com/tx7do/go-crud/viewer"
)

// ViewerUserIDFromContext 从 ent viewer context 提取操作人用户 ID。
//
// OA viewer 中间件（internal/server/middleware/viewer.go）在鉴权之后把 JWT 身份
// 翻译为 viewer context，供 TenantPrivacy 策略做行级租户隔离。本 helper 仅取
// 用户 ID，用于“待办 / 已办 / 我的申请”等按调用人过滤的行级查询。
// 无 viewer（未鉴权 / 系统上下文）返回 (0, false)，调用方据此 fail-closed。
func ViewerUserIDFromContext(ctx context.Context) (uint32, bool) {
	vc, exist := viewer.FromContext(ctx)
	if !exist || vc == nil {
		return 0, false
	}
	uid := vc.UserID()
	if uid == 0 {
		return 0, false
	}
	return uint32(uid), true
}

// viewerUserIDFromContext 是 ViewerUserIDFromContext 的包內別名，供 cms 移植的
// internal_message 倉庫按原調用名引用（語義完全一致）。
func viewerUserIDFromContext(ctx context.Context) (uint32, bool) {
	return ViewerUserIDFromContext(ctx)
}

// maybeTenantFromViewer returns the viewer's tenant id if present and >0, plus whether a viewer context exists.
// 供 cms 移植的 internal_message 倉庫按原調用名引用，語義與 cms 一致。
func maybeTenantFromViewer(ctx context.Context) (tenantID uint32, hasTenant bool) {
	vc, exist := viewer.FromContext(ctx)
	if !exist {
		return 0, false
	}
	tid := vc.TenantID()
	if tid == 0 {
		return 0, false
	}
	return uint32(tid), true
}

// ptr 取任意值的指针，用于填充 proto optional 字段（如 MyTaskItem.StatusLabel
// 等 *string 字段）。与 service 包内的同名 helper 等价，因 data 包独立而在此重声明。
func ptr[T any](v T) *T { return &v }
