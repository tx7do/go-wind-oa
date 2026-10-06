package service

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	stdsql "database/sql"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	entCrud "github.com/tx7do/go-crud/entgo"
	gcrudViewer "github.com/tx7do/go-crud/viewer"
	entsql "entgo.io/ent/dialect/sql"

	_ "modernc.org/sqlite"

	identityV1 "go-wind-oa/api/gen/go/identity/service/v1"
	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
	"go-wind-oa/app/core/service/internal/data"
	"go-wind-oa/app/core/service/internal/data/ent"
	_ "go-wind-oa/app/core/service/internal/data/ent/runtime"
	"go-wind-oa/app/core/service/internal/data/ent/user"
	entMigrate "go-wind-oa/app/core/service/internal/data/ent/migrate"
	oaViewer "go-wind-oa/pkg/entgo/viewer"
)

// 审批动作目标校验测试：转办/加签的目标用户必须同租户且在职（NORMAL）。
// 审批路径此前零测试覆盖；本文件用独立 SQLite 库走真实 repo 栈。

type auditFixture struct {
	svc    *WorkflowService
	client *ent.Client
	tid    uint32
	uid    uint32 // 审批人（任务 assignee）
}

func newAuditFixture(t *testing.T) *auditFixture {
	t.Helper()

	dsn := filepath.Join(t.TempDir(), "test.db")
	db, err := stdsql.Open("sqlite", dsn+"?_fk=1")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	drv := entsql.OpenDB("sqlite3", db)
	client := ent.NewClient(ent.Driver(drv))

	ctx := oaViewer.NewSystemViewerContext(context.Background())
	if err := client.Schema.Create(ctx, entMigrate.WithForeignKeys(true)); err != nil {
		t.Fatalf("schema migrate: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	const tid, uid = uint32(9100), uint32(1)
	// 审批人本人（消息发送者身份需要）
	if _, err := client.User.Create().
		SetTenantID(tid).SetUsername("approver").SetStatus(entUserStatusNormal()).
		Save(ctx); err != nil {
		t.Fatalf("seed approver user: %v", err)
	}

	kratosLog := log.DefaultLogger
	bctx := bootstrap.NewContextWithParam(context.Background(), nil, nil, kratosLog)
	l := log.NewHelper(kratosLog)
	ec := entCrud.NewEntClient[*ent.Client](client, drv)
	svc := &WorkflowService{
		log:            l,
		instanceRepo:   data.NewWorkflowInstanceRepo(bctx, ec),
		taskRepo:       data.NewWorkflowTaskRepo(bctx, ec),
		logRepo:        data.NewWorkflowLogRepo(bctx, ec),
		resolverRepo:   data.NewWorkflowResolverRepo(bctx, ec),
		joinRepo:       data.NewWorkflowInstanceJoinRepo(bctx, ec),
		parentLinkRepo: data.NewWorkflowInstanceParentLinkRepo(bctx, ec),
		delegationRepo: data.NewWorkflowDelegationRepo(bctx, ec),
	}
	return &auditFixture{svc: svc, client: client, tid: tid, uid: uid}
}

// seedPendingTask 造一条 assignee=uid 的待办任务 + PENDING 实例，返回任务 ID。
func (f *auditFixture) seedPendingTask(t *testing.T, ctx context.Context) uint32 {
	t.Helper()
	inst, err := f.client.WorkflowInstance.Create().
		SetTenantID(f.tid).SetCreatedBy(f.uid).
		Save(ctx)
	if err != nil {
		t.Fatalf("seed instance: %v", err)
	}
	task, err := f.client.WorkflowTask.Create().
		SetTenantID(f.tid).SetInstanceID(inst.ID).
		SetNodeID("node_1").SetAssigneeUserID(f.uid).
		Save(ctx)
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}
	return task.ID
}

// asViewer 返回带指定操作者身份的上下文
func (f *auditFixture) asViewer(uid uint32) context.Context {
	vc := oaViewer.NewUserViewer(uint64(uid), uint64(f.tid), 0, "test", identityV1.DataScope_ALL)
	return gcrudViewer.WithContext(context.Background(), vc)
}

func TestAuditForwardTargetValidation(t *testing.T) {
	f := newAuditFixture(t)
	ctx := f.asViewer(f.uid)
	taskID := f.seedPendingTask(t, ctx)

	cases := []struct {
		name      string
		forwardTo uint32
		wantErr   string
	}{
		{"nonexistent user", 999, "unavailable"},
		{"zero target", 0, "invalid forward target"},
		{"self target", f.uid, "invalid forward target"},
	}
	for _, tc := range cases {
		_, err := f.svc.AuditTask(ctx, &oaV1.AuditTaskRequest{
			TaskId:    taskID,
			Action:    oaV1.AuditAction_FORWARD,
			ForwardTo: tc.forwardTo,
		})
		if err == nil {
			t.Fatalf("%s: expected error", tc.name)
		}
		if !strings.Contains(err.Error(), tc.wantErr) {
			t.Fatalf("%s: error %q does not contain %q", tc.name, err.Error(), tc.wantErr)
		}
	}
}

func TestAuditForwardTargetMustBeActive(t *testing.T) {
	f := newAuditFixture(t)
	ctx := f.asViewer(f.uid)
	taskID := f.seedPendingTask(t, ctx)

	// 同租户但已停用的用户
	disabledID, err := f.client.User.Create().
		SetTenantID(f.tid).SetUsername("disabled-user").
		SetStatus(entUserStatusDisabled()).
		Save(ctx)
	if err != nil {
		t.Fatalf("seed disabled user: %v", err)
	}

	_, err = f.svc.AuditTask(ctx, &oaV1.AuditTaskRequest{
		TaskId:    taskID,
		Action:    oaV1.AuditAction_FORWARD,
		ForwardTo: uint32(disabledID.ID),
	})
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("disabled target should be rejected, got %v", err)
	}

	// 跨租户存在的同 ID 用户同样拒绝。
	// 注意：TenantPrivacy 在普通用户上下文 Create 时会强制覆盖 tenant_id 为
	// viewer 的租户（防越权），必须用 system viewer 才能落到别的租户。
	otherTenantID, err := f.client.User.Create().
		SetTenantID(f.tid + 1).SetUsername("other-tenant-user").
		SetStatus(entUserStatusNormal()).
		Save(oaViewer.NewSystemViewerContext(context.Background()))
	if err != nil {
		t.Fatalf("seed cross-tenant user: %v", err)
	}
	_, err = f.svc.AuditTask(ctx, &oaV1.AuditTaskRequest{
		TaskId:    taskID,
		Action:    oaV1.AuditAction_FORWARD,
		ForwardTo: uint32(otherTenantID.ID),
	})
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("cross-tenant target should be rejected, got %v", err)
	}
}

func TestAuditTaskOwnershipGuard(t *testing.T) {
	f := newAuditFixture(t)
	taskID := f.seedPendingTask(t, f.asViewer(f.uid))

	// 非 assignee 调用（uid=7）应被拒绝，且不区分动作
	_, err := f.svc.AuditTask(f.asViewer(7), &oaV1.AuditTaskRequest{
		TaskId:    taskID,
		Action:    oaV1.AuditAction_FORWARD,
		ForwardTo: 2,
	})
	if err == nil || !strings.Contains(err.Error(), "not your pending task") {
		t.Fatalf("non-assignee should be rejected, got %v", err)
	}

	_, err = f.svc.AuditTask(f.asViewer(7), &oaV1.AuditTaskRequest{
		TaskId:             taskID,
		Action:             oaV1.AuditAction_ADD_APPROVER,
		AdditionalApprover: 2,
	})
	if err == nil || !strings.Contains(err.Error(), "not your pending task") {
		t.Fatalf("non-assignee add-approver should be rejected, got %v", err)
	}
}

func TestAuditAddApproverTargetValidation(t *testing.T) {
	f := newAuditFixture(t)
	ctx := f.asViewer(f.uid)
	taskID := f.seedPendingTask(t, ctx)

	_, err := f.svc.AuditTask(ctx, &oaV1.AuditTaskRequest{
		TaskId:             taskID,
		Action:             oaV1.AuditAction_ADD_APPROVER,
		AdditionalApprover: 999,
	})
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("nonexistent add-approver target should be rejected, got %v", err)
	}
}

func entUserStatusNormal() user.Status { return user.StatusNormal }
func entUserStatusDisabled() user.Status { return user.StatusDisabled }
