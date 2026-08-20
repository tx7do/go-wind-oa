// smokeseed 端到端冒烟测试的数据种子工具（仅限本地开发环境使用）。
//
// 直连本地 Postgres（citus 容器映射的 5432），执行 ent 自动建表后种入一套
// 最小可用的冒烟数据：租户 + 角色 + 组织（含 leader）+ 两名用户（主管/员工，
// 密码见 -password）+ 请假类型与员工额度。幂等：租户已存在则跳过写入。
//
// 运行：go run ./app/core/service/cmd/smokeseed -dsn "host=localhost port=5432 user=postgres password=*Abcd123456 dbname=gwc sslmode=disable" -password 12345678
//
// 输出的 loginPassword 是 LoginRequest.password 所需的 base64(AES(明文)) 串。
package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	"github.com/tx7do/go-utils/crypto"

	"go-wind-oa/app/core/service/internal/data/ent"
	"go-wind-oa/app/core/service/internal/data/ent/migrate"
	_ "go-wind-oa/app/core/service/internal/data/ent/runtime"
	"go-wind-oa/app/core/service/internal/data/ent/tenant"
	"go-wind-oa/app/core/service/internal/data/ent/user"
	"go-wind-oa/app/core/service/internal/data/ent/usercredential"
	"go-wind-oa/app/core/service/internal/data/ent/userorgunit"
	"go-wind-oa/app/core/service/internal/data/ent/userrole"
	oaViewer "go-wind-oa/pkg/entgo/viewer"
)

const (
	tenantCode = "smoke"
	roleCode   = "smoker"
	orgCode    = "SMOKE-DEPT"
)

func main() {
	dsn := flag.String("dsn", "host=localhost port=5432 user=postgres password=*Abcd123456 dbname=gwc sslmode=disable", "postgres DSN")
	password := flag.String("password", "12345678", "冒烟用户明文密码")
	flag.Parse()

	// 隐私层要求 ViewerContext；种子工具以系统视图执行（不受租户隔离过滤约束）。
	ctx := oaViewer.NewSystemViewerContext(context.Background())

	db, err := entsql.Open(dialect.Postgres, *dsn)
	if err != nil {
		log.Fatalf("open db failed: %v", err)
	}
	defer db.Close()
	client := ent.NewClient(ent.Driver(db))
	defer client.Close()

	// 建表（与服务启动时的自动迁移等价）。
	if err := client.Schema.Create(ctx, migrate.WithForeignKeys(true)); err != nil {
		log.Fatalf("migrate failed: %v", err)
	}

	// 幂等：租户已存在则跳过。
	existing, _ := client.Tenant.Query().Where().Count(ctx)
	if existing > 0 {
		if t, err := client.Tenant.Query().Limit(1).Only(ctx); err == nil && t.Code != nil && *t.Code == tenantCode {
			printPassword(*password)
			return
		}
	}

	now := time.Now()

	tenant, err := client.Tenant.Create().
		SetCode(tenantCode).
		SetName("冒烟租户").
		SetStatus(tenant.StatusOn).
		SetType(tenant.TypeTrial).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		log.Fatalf("create tenant failed: %v", err)
	}
	tid := tenant.ID

	role, err := client.Role.Create().
		SetCode(roleCode).
		SetName("冒烟角色").
		SetTenantID(tid).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		log.Fatalf("create role failed: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("bcrypt failed: %v", err)
	}

	leader := createUser(ctx, client, tid, "leader", "李主管", "leader@smoke.local", string(hash), now)
	employee := createUser(ctx, client, tid, "employee", "王员工", "employee@smoke.local", string(hash), now)

	org, err := client.OrgUnit.Create().
		SetName("冒烟部").
		SetCode(orgCode).
		SetLeaderID(leader.ID).
		SetTenantID(tid).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		log.Fatalf("create org failed: %v", err)
	}

	// 用户-角色（登录要求用户至少有一个角色码）。
	for _, uid := range []uint32{leader.ID, employee.ID} {
		if _, err := client.UserRole.Create().
			SetUserID(uid).
			SetRoleID(role.ID).
			SetStatus(userrole.StatusActive).
			SetIsPrimary(true).
			SetAssignedAt(now).
			SetTenantID(tid).
			SetCreatedAt(now).
			Save(ctx); err != nil {
			log.Fatalf("create user_role failed: %v", err)
		}
	}

	// 用户-组织（LEADER 寻人链路：user_org_unit → org_unit.leader_id）。
	for _, uid := range []uint32{leader.ID, employee.ID} {
		if _, err := client.UserOrgUnit.Create().
			SetUserID(uid).
			SetOrgUnitID(org.ID).
			SetStatus(userorgunit.StatusActive).
			SetIsPrimary(true).
			SetAssignedAt(now).
			SetTenantID(tid).
			SetCreatedAt(now).
			Save(ctx); err != nil {
			log.Fatalf("create user_org_unit failed: %v", err)
		}
	}

	// 权限 + 角色-权限关联（登录要求角色具有后台访问权限码）。
	perm, err := client.Permission.Create().
		SetCode("sys:access_backend").
		SetName("后台访问").
		SetTenantID(tid).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		log.Fatalf("create permission failed: %v", err)
	}
	if _, err := client.RolePermission.Create().
		SetRoleID(role.ID).
		SetPermissionID(perm.ID).
		SetTenantID(tid).
		SetCreatedAt(now).
		Save(ctx); err != nil {
		log.Fatalf("create role_permission failed: %v", err)
	}

	leaveType, err := client.LeaveType.Create().
		SetCode("ANNUAL").
		SetName("年假").
		SetTenantID(tid).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		log.Fatalf("create leave type failed: %v", err)
	}

	if _, err := client.LeaveBalance.Create().
		SetUserID(employee.ID).
		SetLeaveTypeID(leaveType.ID).
		SetYear(now.Year()).
		SetTotalDays(10).
		SetUsedDays(0).
		SetTenantID(tid).
		SetCreatedAt(now).
		Save(ctx); err != nil {
		log.Fatalf("create leave balance failed: %v", err)
	}

	fmt.Printf("seeded: tenant=%s(%d) leader=%s(%d) employee=%s(%d) org=%s(%d, leader=%d) leaveType=%s(%d) balance=10天\n",
		tenantCode, tid, "leader", leader.ID, "employee", employee.ID, orgCode, org.ID, leader.ID, "ANNUAL", leaveType.ID)
	printPassword(*password)
}

func createUser(ctx context.Context, client *ent.Client, tid uint32, username, realname, email, passwordHash string, now time.Time) *ent.User {
	u, err := client.User.Create().
		SetUsername(username).
		SetNickname(realname).
		SetRealname(realname).
		SetEmail(email).
		SetStatus(user.StatusNormal).
		SetTenantID(tid).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		log.Fatalf("create user %s failed: %v", username, err)
	}
	if _, err := client.UserCredential.Create().
		SetUserID(u.ID).
		SetIdentityType(usercredential.IdentityTypeUsername).
		SetIdentifier(username).
		SetCredentialType(usercredential.CredentialTypePasswordHash).
		SetCredential(passwordHash).
		SetStatus(usercredential.StatusEnabled).
		SetIsPrimary(true).
		SetTenantID(tid).
		SetCreatedAt(now).
		Save(ctx); err != nil {
		log.Fatalf("create credential %s failed: %v", username, err)
	}
	return u
}

func printPassword(plain string) {
	enc, err := crypto.AesEncrypt([]byte(plain), crypto.DefaultAESKey, nil)
	if err != nil {
		log.Fatalf("aes encrypt failed: %v", err)
	}
	fmt.Printf("loginPassword: %s\n", base64.StdEncoding.EncodeToString(enc))
}
