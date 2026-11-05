package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	adminV1 "go-wind-oa/api/gen/go/admin/service/v1"

	"go-wind-oa/pkg/middleware/auth"
)

// requirePlatformAdmin 把平台专用服务的变更入口挡在平台管理员之内。
//
// 移植自上游 go-wind-admin 的 notification_platform_guard.go（da2b3a5f）：
// 模块白名单闸门（ServiceTagToBusinessModule）把 Tenant/Plan 登记在 MODULE_TENANT、
// Menu/Api/Permission 登记在 MODULE_PERMISSION——这些模块因租户自助用量与角色抽屉
// 读权限树而常态在租户套餐白名单内，叠加 authz.type 默认 noop 全放行，意味着默认
// 部署下租户操作者即可打到平台专用变更端点。而规则类的口之所以被拒，只是因为
// ServiceTagToBusinessModule 少登记了那些服务 ⇒ business_module 为空 ⇒ 租户闸门把
// "未归类"按不在任何白名单内拒绝。一道护栏不该靠漏配存在，所以判定显式放这里。
func requirePlatformAdmin(ctx context.Context, log *log.Helper, resource string) error {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return err
	}

	if operator.GetIsPlatformAdmin() {
		return nil
	}

	log.Errorf("operator [%d] (tenant %d) is not a platform administrator, %s access denied",
		operator.GetUserId(), operator.GetTenantId(), resource)

	return adminV1.ErrorForbidden("%s is platform-level configuration, only platform administrators may access it", resource)
}
