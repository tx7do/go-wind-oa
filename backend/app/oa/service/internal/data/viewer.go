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

// ptr 取任意值的指针，用于填充 proto optional 字段（如 MyTaskItem.StatusLabel
// 等 *string 字段）。与 service 包内的同名 helper 等价，因 data 包独立而在此重声明。
func ptr[T any](v T) *T { return &v }
