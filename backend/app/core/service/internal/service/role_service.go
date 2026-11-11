package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/go-utils/aggregator"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"go-wind-oa/app/core/service/internal/data"

	identityV1 "go-wind-oa/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-oa/api/gen/go/permission/service/v1"

	"go-wind-oa/pkg/constants"
	appViewer "go-wind-oa/pkg/entgo/viewer"
	"go-wind-oa/pkg/middleware/auth"
	"go-wind-oa/pkg/utils"
)

type RoleService struct {
	permissionV1.UnimplementedRoleServiceServer

	log *log.Helper

	roleRepo     *data.RoleRepo
	tenantRepo   *data.TenantRepo
	userRoleRepo *data.UserRoleRepo
	userRepo     data.UserRepo
}

func NewRoleService(
	ctx *bootstrap.Context,
	roleRepo *data.RoleRepo,
	tenantRepo *data.TenantRepo,
	userRoleRepo *data.UserRoleRepo,
	userRepo data.UserRepo,
) *RoleService {
	svc := &RoleService{
		log:          ctx.NewLoggerHelper("role/service/core-service"),
		roleRepo:     roleRepo,
		tenantRepo:   tenantRepo,
		userRoleRepo: userRoleRepo,
		userRepo:     userRepo,
	}

	svc.init()

	return svc
}

func (s *RoleService) init() {
	ctx := appViewer.NewSystemViewerContext(context.Background())
	if count, _ := s.roleRepo.Count(ctx, nil); count == 0 {
		_ = s.createDefaultRoles(ctx)
	}
}

func (s *RoleService) extractRelationIDs(
	roles []*permissionV1.Role,
	tenantSet aggregator.ResourceMap[int64, *identityV1.Tenant],
	roleSet aggregator.ResourceMap[int64, *permissionV1.Role],
) {
	for _, p := range roles {
		if p.GetTenantId() > 0 {
			tenantSet[p.GetTenantId()] = nil
		}
		if p.GetParentId() > 0 {
			roleSet[p.GetParentId()] = nil
		}
	}
}

func (s *RoleService) fetchRelationInfo(
	ctx context.Context,
	tenantSet aggregator.ResourceMap[int64, *identityV1.Tenant],
	roleSet aggregator.ResourceMap[int64, *permissionV1.Role],
) error {
	if len(tenantSet) > 0 {
		tenantIds := make([]int64, 0, len(tenantSet))
		for id := range tenantSet {
			tenantIds = append(tenantIds, id)
		}

		tenants, err := s.tenantRepo.ListTenantsByIds(ctx, tenantIds)
		if err != nil {
			s.log.Errorf("query tenants err: %v", err)
			return err
		}

		for _, tenant := range tenants {
			tenantSet[tenant.GetId()] = tenant
		}
	}

	if len(roleSet) > 0 {
		roleIds := make([]int64, 0, len(roleSet))
		for id := range roleSet {
			roleIds = append(roleIds, id)
		}

		parents, err := s.roleRepo.ListRolesByRoleIds(ctx, roleIds)
		if err != nil {
			s.log.Errorf("query parent roles err: %v", err)
			return err
		}

		for _, parent := range parents {
			roleSet[parent.GetId()] = parent
		}
	}

	return nil
}

func (s *RoleService) bindRelations(
	roles []*permissionV1.Role,
	tenantSet aggregator.ResourceMap[int64, *identityV1.Tenant],
	roleSet aggregator.ResourceMap[int64, *permissionV1.Role],
) {
	aggregator.Populate(
		roles,
		tenantSet,
		func(ou *permissionV1.Role) int64 { return ou.GetTenantId() },
		func(ou *permissionV1.Role, r *identityV1.Tenant) {
			ou.TenantName = r.Name
		},
	)
	aggregator.Populate(
		roles,
		roleSet,
		func(ou *permissionV1.Role) int64 { return ou.GetParentId() },
		func(ou *permissionV1.Role, r *permissionV1.Role) {
			ou.ParentName = trans.Ptr(r.GetName())
		},
	)
}

func (s *RoleService) enrichRelations(ctx context.Context, roles []*permissionV1.Role) error {
	var tenantSet = make(aggregator.ResourceMap[int64, *identityV1.Tenant])
	var roleSet = make(aggregator.ResourceMap[int64, *permissionV1.Role])
	s.extractRelationIDs(roles, tenantSet, roleSet)
	if err := s.fetchRelationInfo(ctx, tenantSet, roleSet); err != nil {
		return err
	}
	s.bindRelations(roles, tenantSet, roleSet)
	return nil
}

// hasMaskPath 探测 updateMask 是否声明了给定字段（proto 字段名与 json_name 双探测）。
// mask 为 nil 视为全字段写入（FilterByFieldMask 语义），返回 true。
func hasMaskPath(mask *fieldmaskpb.FieldMask, names ...string) bool {
	if mask == nil {
		return true
	}
	for _, p := range mask.GetPaths() {
		for _, n := range names {
			if p == n {
				return true
			}
		}
	}
	return false
}

// validateParentBinding 校验父子角色绑定的合法性（权限继承核心约束，编辑时校验语义）。
// parentID 为 0 表示根角色，直接放行。
func (s *RoleService) validateParentBinding(ctx context.Context, roleID, parentID, tenantID int64, wantPerms []int64) error {
	if parentID == 0 {
		return nil
	}

	parent, err := s.roleRepo.Get(ctx, &permissionV1.GetRoleRequest{
		QueryBy: &permissionV1.GetRoleRequest_Id{Id: parentID},
	})
	if err != nil {
		s.log.Errorf("get parent role failed: %v", err)
		return identityV1.ErrorBadRequest("parent role not found")
	}

	if parent.GetTenantId() != tenantID {
		return identityV1.ErrorForbidden("parent role must belong to the same tenant")
	}

	if len(wantPerms) == 0 {
		return nil
	}

	capSet := make(map[int64]struct{}, len(parent.GetPermissions()))
	for _, id := range parent.GetPermissions() {
		capSet[id] = struct{}{}
	}

	var exceeding int
	for _, id := range wantPerms {
		if _, ok := capSet[id]; !ok {
			exceeding++
		}
	}
	if exceeding > 0 {
		return identityV1.ErrorForbidden(
			"%d permission(s) exceed the parent role's granted permission set", exceeding)
	}
	return nil
}

// ensureNoRoleCycle 沿候选父角色上溯检测层级环（Update 改父时调用）。
func (s *RoleService) ensureNoRoleCycle(ctx context.Context, roleID, parentID int64) error {
	const maxDepth = 32

	current := parentID
	for depth := 0; current != 0 && depth < maxDepth; depth++ {
		if current == roleID {
			return identityV1.ErrorForbidden("circular role hierarchy detected")
		}
		parent, err := s.roleRepo.Get(ctx, &permissionV1.GetRoleRequest{
			QueryBy: &permissionV1.GetRoleRequest_Id{Id: current},
		})
		if err != nil {
			return nil
		}
		current = parent.GetParentId()
	}
	return nil
}

func (s *RoleService) List(ctx context.Context, req *paginationV1.PagingRequest) (*permissionV1.ListRoleResponse, error) {
	resp, err := s.roleRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	_ = s.enrichRelations(ctx, resp.Items)

	return resp, nil
}

func (s *RoleService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*permissionV1.CountRoleResponse, error) {
	count, err := s.roleRepo.Count(ctx, req)
	if err != nil {
		return nil, err
	}

	return &permissionV1.CountRoleResponse{
		Count: int64(count),
	}, nil
}

func (s *RoleService) Get(ctx context.Context, req *permissionV1.GetRoleRequest) (*permissionV1.Role, error) {
	resp, err := s.roleRepo.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	fakeItems := []*permissionV1.Role{resp}
	_ = s.enrichRelations(ctx, fakeItems)

	return resp, nil
}

func (s *RoleService) Create(ctx context.Context, req *permissionV1.CreateRoleRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	// 父角色绑定校验（权限继承）。租户操作者的归属租户以操作者上下文为准。
	roleTenantID := req.Data.GetTenantId()
	if operator, opErr := auth.FromContext(ctx); opErr == nil && operator.GetTenantId() > 0 {
		roleTenantID = operator.GetTenantId()
	}
	if err := s.validateParentBinding(ctx, 0, req.Data.GetParentId(), roleTenantID, req.Data.GetPermissions()); err != nil {
		return nil, err
	}

	if err := s.roleRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *RoleService) Update(ctx context.Context, req *permissionV1.UpdateRoleRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	r, err := s.roleRepo.Get(ctx, &permissionV1.GetRoleRequest{
		QueryBy: &permissionV1.GetRoleRequest_Id{
			Id: req.Data.GetId(),
		},
	})
	if err != nil {
		return nil, err
	}

	// 保护角色字段不可修改
	if r.GetIsProtected() {
		if len(req.GetUpdateMask().Paths) > 0 {
			req.GetUpdateMask().Paths = utils.FilterBlacklist(req.GetUpdateMask().Paths, []string{
				"is_protected",
				"type",
				"status",
				"code",
			})
		} else {
			req.Data.IsProtected = nil
			req.Data.Type = nil
			req.Data.Status = nil
			req.Data.Code = nil
		}
	}

	// 父角色绑定校验（权限继承，编辑时校验语义）：生效父/权限集按 mask 声明取舍。
	parentIDChanged := hasMaskPath(req.UpdateMask, "parent_id", "parentId")
	permsChanged := hasMaskPath(req.UpdateMask, "permissions")
	var effectiveParentID int64
	var effectivePerms []int64
	if parentIDChanged {
		effectiveParentID = req.Data.GetParentId()
	} else {
		effectiveParentID = r.GetParentId()
	}
	if permsChanged {
		effectivePerms = req.Data.GetPermissions()
	} else {
		effectivePerms = r.GetPermissions()
	}
	if effectiveParentID > 0 {
		if parentIDChanged {
			if err := s.ensureNoRoleCycle(ctx, req.Data.GetId(), effectiveParentID); err != nil {
				return nil, err
			}
		}
		if err := s.validateParentBinding(ctx, req.Data.GetId(), effectiveParentID, r.GetTenantId(), effectivePerms); err != nil {
			return nil, err
		}
	}

	if err = s.roleRepo.Update(ctx, req); err != nil {
		s.log.Errorf("update role error: %v", err)
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *RoleService) Delete(ctx context.Context, req *permissionV1.DeleteRoleRequest) (*emptypb.Empty, error) {
	var err error

	if err = s.roleRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *RoleService) ListRoleCodesByIds(ctx context.Context, req *permissionV1.ListRoleCodesByIdsRequest) (*permissionV1.ListRoleCodesByIdsResponse, error) {
	ids, err := s.roleRepo.ListRoleCodesByRoleIds(ctx, req.GetRoleIds())
	if err != nil {
		return nil, err
	}

	return &permissionV1.ListRoleCodesByIdsResponse{
		RoleCodes: ids,
	}, nil
}

func (s *RoleService) ListRoleIdsByCodes(ctx context.Context, req *permissionV1.ListRoleIdsByCodesRequest) (*permissionV1.ListRoleIdsByCodesResponse, error) {
	ids, err := s.roleRepo.ListRoleIDsByRoleCodes(ctx, req.GetRoleCodes())
	if err != nil {
		return nil, err
	}

	return &permissionV1.ListRoleIdsByCodesResponse{
		RoleIds: ids,
	}, nil
}

func (s *RoleService) ListPermissionIds(ctx context.Context, req *permissionV1.ListPermissionIdsRequest) (*permissionV1.ListPermissionIdsResponse, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid request")
	}
	var permissionIDs []int64
	var err error

	switch req.QueryBy.(type) {
	case *permissionV1.ListPermissionIdsRequest_RoleId:
		permissionIDs, err = s.roleRepo.ListPermissionIDsByRoleIDs(ctx, []int64{req.GetRoleId()})
		if err != nil {
			return nil, err
		}

	case *permissionV1.ListPermissionIdsRequest_RoleCode:
		permissionIDs, err = s.roleRepo.ListPermissionIDsByRoleCodes(ctx, []string{req.GetRoleCode()})
		if err != nil {
			return nil, err
		}

	case *permissionV1.ListPermissionIdsRequest_UserId:
		// 校验目标用户归属当前调用者租户，避免跨租户枚举他人权限 ID
		if err := s.validateTargetUserTenant(ctx, req.GetUserId()); err != nil {
			return nil, err
		}
		permissionIDs, err = s.roleRepo.ListPermissionIDsByUserID(ctx, req.GetUserId())
		if err != nil {
			return nil, err
		}

	default:
		if len(req.RoleIds) > 0 {
			permissionIDs, err = s.roleRepo.ListPermissionIDsByRoleIDs(ctx, req.GetRoleIds())
			if err != nil {
				return nil, err
			}
		}

		if len(req.RoleCodes) > 0 {
			permissionIDs, err = s.roleRepo.ListPermissionIDsByRoleCodes(ctx, req.GetRoleCodes())
			if err != nil {
				return nil, err
			}
		}
	}

	return &permissionV1.ListPermissionIdsResponse{
		PermissionIds: permissionIDs,
	}, nil
}

func (s *RoleService) ListUserRoleIDs(ctx context.Context, req *permissionV1.ListUserRoleIDsRequest) (*permissionV1.ListUserRoleIDsResponse, error) {
	// 校验目标用户归属当前调用者租户，避免跨租户枚举他人角色 ID
	if err := s.validateTargetUserTenant(ctx, req.GetUserId()); err != nil {
		return nil, err
	}
	roleIDs, err := s.userRoleRepo.ListRoleIDs(ctx, req.GetUserId(), false)
	if err != nil {
		return nil, err
	}

	return &permissionV1.ListUserRoleIDsResponse{
		RoleIds: roleIDs,
	}, nil
}

// AssignRolesToUser 为用户分配角色（替换旧关联）
func (s *RoleService) AssignRolesToUser(ctx context.Context, req *permissionV1.AssignRolesToUserRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetUserId() == 0 || len(req.GetRoleIds()) == 0 {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	// 操作人身份与租户必须从 viewer context 推导，忽略客户端传入的 operator_id/tenant_id，
	// 防止越权将角色绑定写入他租户或伪造审计归属
	callerUserID, hasUser := viewerUserIDFromContext(ctx)
	if !hasUser {
		return nil, permissionV1.ErrorBadRequest("operator identity is required")
	}
	var callerTenantID int64
	isPlatform := false
	if vc, exist := viewer.FromContext(ctx); exist && vc != nil {
		callerTenantID = int64(vc.TenantID())
		isPlatform = vc.IsPlatformContext() || vc.IsSystemContext()
	}

	// 校验目标用户归属当前调用者租户：平台上下文放行；租户用户仅可向本租户用户授权。
	// (userRepo.Get 经 EvalQuery 注入 tenant 谓词，租户用户取不到他租户用户即自动拒绝)
	targetUser, err := s.userRepo.Get(ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{Id: req.GetUserId()},
	})
	if err != nil || targetUser == nil {
		return nil, permissionV1.ErrorBadRequest("target user not found")
	}
	if !isPlatform && targetUser.GetTenantId() != callerTenantID {
		return nil, permissionV1.ErrorForbidden("cannot assign roles to a user in another tenant")
	}

	// 逐个校验角色可分配性（启用/非模板/非阻断/非平台管理员模板），CanAssignRole 内部 Get 经
	// EvalQuery 注入 tenant 谓词，租户用户取不到他租户角色即自动拒绝
	for _, roleID := range req.GetRoleIds() {
		ok, err := s.roleRepo.CanAssignRole(ctx, roleID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, permissionV1.ErrorForbidden("role %d is not assignable", roleID)
		}
	}

	now := time.Now()

	userRoles := make([]*permissionV1.UserRole, 0, len(req.GetRoleIds()))
	for _, roleID := range req.GetRoleIds() {
		userRoles = append(userRoles, &permissionV1.UserRole{
			TenantId:   trans.Ptr(callerTenantID),
			UserId:     trans.Ptr(req.GetUserId()),
			RoleId:     trans.Ptr(roleID),
			Status:     permissionV1.UserRole_ACTIVE.Enum(),
			AssignedBy: trans.Ptr(callerUserID),
			AssignedAt: timeutil.TimeToTimestamppb(&now),
			StartAt:    timeutil.TimeToTimestamppb(&now),
			CreatedBy:  trans.Ptr(callerUserID),
		})
	}

	if err := s.userRoleRepo.AssignRolesToUser(ctx, req.GetUserId(), userRoles); err != nil {
		s.log.Errorf("assign roles to user failed: %v", err)
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// GetUserRoles 获取用户的角色绑定列表
func (s *RoleService) GetUserRoles(ctx context.Context, req *permissionV1.GetUserRolesRequest) (*permissionV1.GetUserRolesResponse, error) {
	if req == nil || req.GetUserId() == 0 {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	// 校验目标用户归属当前调用者租户，避免跨租户枚举他人角色绑定
	if err := s.validateTargetUserTenant(ctx, req.GetUserId()); err != nil {
		return nil, err
	}

	bindings, err := s.userRoleRepo.ListByUserID(ctx, req.GetUserId(), req.GetIncludeExpired())
	if err != nil {
		return nil, err
	}

	return &permissionV1.GetUserRolesResponse{
		Bindings: bindings,
	}, nil
}

// UnassignRolesFromUser 从用户移除指定角色
func (s *RoleService) UnassignRolesFromUser(ctx context.Context, req *permissionV1.UnassignRolesFromUserRequest) (*permissionV1.UnassignRolesFromUserResponse, error) {
	if req == nil || req.GetUserId() == 0 || len(req.GetRoleIds()) == 0 {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	if err := s.userRoleRepo.RemoveRolesFromUser(ctx, req.GetUserId(), req.GetRoleIds()); err != nil {
		s.log.Errorf("unassign roles from user failed: %v", err)
		return nil, err
	}

	return &permissionV1.UnassignRolesFromUserResponse{
		RemovedRoleIds: req.GetRoleIds(),
	}, nil
}

// validateTargetUserTenant 校验目标用户归属当前调用者租户：
// 平台/系统上下文放行；租户用户仅可查询本租户用户的数据。
// userRepo.Get 经 EvalQuery 注入 tenant 谓词，租户用户取不到他租户用户即自动拒绝。
// 返回 nil 表示放行，非 nil 表示拒绝（调用方应直接返回该错误）。
func (s *RoleService) validateTargetUserTenant(ctx context.Context, targetUserID int64) error {
	var callerTenantID int64
	isPlatform := false
	if vc, exist := viewer.FromContext(ctx); exist && vc != nil {
		callerTenantID = int64(vc.TenantID())
		isPlatform = vc.IsPlatformContext() || vc.IsSystemContext()
	}
	targetUser, err := s.userRepo.Get(ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{Id: targetUserID},
	})
	if err != nil || targetUser == nil {
		return permissionV1.ErrorBadRequest("target user not found")
	}
	if !isPlatform && targetUser.GetTenantId() != callerTenantID {
		return permissionV1.ErrorForbidden("cannot access a user in another tenant")
	}
	return nil
}

// createDefaultRoles 创建默认角色(包括超级管理员)
func (s *RoleService) createDefaultRoles(ctx context.Context) error {
	var err error

	for _, d := range constants.DefaultRoles {
		err = s.roleRepo.Create(ctx, &permissionV1.CreateRoleRequest{
			Data: d,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// GetRoleCodesByRoleIds 根据角色ID列表获取角色值列表
func (s *RoleService) GetRoleCodesByRoleIds(ctx context.Context, req *permissionV1.GetRoleCodesByRoleIdsRequest) (*permissionV1.GetRoleCodesByRoleIdsResponse, error) {
	ids, err := s.roleRepo.ListRoleCodesByRoleIds(ctx, req.GetRoleIds())
	if err != nil {
		return nil, err
	}

	return &permissionV1.GetRoleCodesByRoleIdsResponse{
		RoleCodes: ids,
	}, nil
}

// GetRolesByRoleCodes 根据角色值列表获取角色列表
func (s *RoleService) GetRolesByRoleCodes(ctx context.Context, req *permissionV1.GetRolesByRoleCodesRequest) (*permissionV1.ListRoleResponse, error) {
	roles, err := s.roleRepo.ListRolesByRoleCodes(ctx, req.GetRoleCodes())
	if err != nil {
		return nil, err
	}

	return &permissionV1.ListRoleResponse{
		Items: roles,
		Total: int64(len(roles)),
	}, nil
}

// GetRolesByRoleIds 根据角色ID列表获取角色列表
func (s *RoleService) GetRolesByRoleIds(ctx context.Context, req *permissionV1.GetRolesByRoleIdsRequest) (*permissionV1.ListRoleResponse, error) {
	roles, err := s.roleRepo.ListRolesByRoleIds(ctx, req.GetRoleIds())
	if err != nil {
		return nil, err
	}

	return &permissionV1.ListRoleResponse{
		Items: roles,
		Total: int64(len(roles)),
	}, nil
}
