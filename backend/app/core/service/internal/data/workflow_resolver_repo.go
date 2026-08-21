package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-oa/app/core/service/internal/data/ent"
	"go-wind-oa/app/core/service/internal/data/ent/orgunit"
	"go-wind-oa/app/core/service/internal/data/ent/user"
	"go-wind-oa/app/core/service/internal/data/ent/userorgunit"
	"go-wind-oa/app/core/service/internal/data/ent/userposition"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// WorkflowResolverRepo 工作流审批人动态解析数据源。LEADER/POSITION 类型的寻人
// 经 user_org_unit / org_unit / user_position 多跳外键查询完成（这些表之间无 ent
// edge，应用层拼接查询链）。
type WorkflowResolverRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewWorkflowResolverRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *WorkflowResolverRepo {
	return &WorkflowResolverRepo{
		log:       ctx.NewLoggerHelper("workflow-resolver/repo/core-service"),
		entClient: entClient,
	}
}

// ResolveOrgLeader 解析用户主组织单元的负责人（LEADER 审批人类型）。
// 链路：user_org_unit（ACTIVE，is_primary 优先）→ org_unit.leader_id（指向 user）。
// 无组织归属或组织未设负责人时返回明确错误——在提交/推进时即失败，避免流程挂起。
func (r *WorkflowResolverRepo) ResolveOrgLeader(ctx context.Context, tenantID uint32, userID uint32) (uint32, error) {
	links, err := r.entClient.Client().UserOrgUnit.Query().
		Where(
			userorgunit.UserIDEQ(userID),
			userorgunit.StatusEQ(userorgunit.StatusActive),
			userorgunit.TenantIDEQ(tenantID),
		).
		Order(ent.Desc(userorgunit.FieldIsPrimary)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query user org links failed: %s", err.Error())
		return 0, oaV1.ErrorInternalServerError("query user org links failed")
	}
	for _, link := range links {
		if link.OrgUnitID == nil || *link.OrgUnitID == 0 {
			continue
		}
		unit, err := r.entClient.Client().OrgUnit.Query().
			Where(
				orgunit.IDEQ(*link.OrgUnitID),
				orgunit.TenantIDEQ(tenantID),
			).
			Only(ctx)
		if err != nil {
			continue // 无效组织归属，尝试下一条
		}
		if unit.LeaderID != nil && *unit.LeaderID != 0 {
			return *unit.LeaderID, nil
		}
	}
	return 0, oaV1.ErrorBadRequest("no org leader found for user")
}

// ResolveUsernames 批量解析用户姓名（sys_users.username），供列表展示回填。
// 查不到的用户不出现在结果里。
func (r *WorkflowResolverRepo) ResolveUsernames(ctx context.Context, tenantID uint32, userIDs []uint32) (map[uint32]string, error) {
	names := make(map[uint32]string, len(userIDs))
	if len(userIDs) == 0 {
		return names, nil
	}
	entities, err := r.entClient.Client().User.Query().
		Where(
			user.IDIn(userIDs...),
			user.TenantIDEQ(tenantID),
		).
		All(ctx)
	if err != nil {
		r.log.Errorf("query user names failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query user names failed")
	}
	for _, u := range entities {
		if u.Username != nil && *u.Username != "" {
			names[u.ID] = *u.Username
		}
	}
	return names, nil
}

// UserIsActive 校验目标用户存在且处于 Normal 状态（非禁用/过期/关闭）。
// 用于转办目标校验，防止把待办转给不存在或不可用账号。
func (r *WorkflowResolverRepo) UserIsActive(ctx context.Context, tenantID uint32, userID uint32) (bool, error) {
	if userID == 0 {
		return false, nil
	}
	count, err := r.entClient.Client().User.Query().
		Where(
			user.IDEQ(userID),
			user.TenantIDEQ(tenantID),
			user.StatusEQ(user.StatusNormal),
		).
		Count(ctx)
	if err != nil {
		r.log.Errorf("check user active failed: %s", err.Error())
		return false, oaV1.ErrorInternalServerError("check user active failed")
	}
	return count > 0, nil
}

// ResolvePositionHolders 解析指定职位的在职持有者（POSITION 审批人类型，可解析出多人，
// 用于会签/或签的多人节点）。
func (r *WorkflowResolverRepo) ResolvePositionHolders(ctx context.Context, tenantID uint32, positionID uint32) ([]uint32, error) {
	links, err := r.entClient.Client().UserPosition.Query().
		Where(
			userposition.PositionIDEQ(positionID),
			userposition.StatusEQ(userposition.StatusActive),
			userposition.TenantIDEQ(tenantID),
		).
		All(ctx)
	if err != nil {
		r.log.Errorf("query position holders failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query position holders failed")
	}
	userIDs := make([]uint32, 0, len(links))
	for _, link := range links {
		if link.UserID != nil && *link.UserID != 0 {
			userIDs = append(userIDs, *link.UserID)
		}
	}
	if len(userIDs) == 0 {
		return nil, oaV1.ErrorBadRequest("no active holder found for position")
	}
	return userIDs, nil
}
