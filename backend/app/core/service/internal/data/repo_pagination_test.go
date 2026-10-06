package data

import (
	"context"
	"path/filepath"
	"testing"

	stdsql "database/sql"

	entCrud "github.com/tx7do/go-crud/entgo"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/go-kratos/kratos/v2/log"
	entsql "entgo.io/ent/dialect/sql"

	_ "modernc.org/sqlite"

	"go-wind-oa/app/core/service/internal/data/ent"
	_ "go-wind-oa/app/core/service/internal/data/ent/runtime"
	entMigrate "go-wind-oa/app/core/service/internal/data/ent/migrate"
	oaViewer "go-wind-oa/pkg/entgo/viewer"
)

// newPagingTestClient 为 repo 分页测试构建独立的文件型 SQLite ent 客户端
// （纯 Go 驱动，无 cgo；每个测试独立临时库，互不串扰）。
func newPagingTestClient(t *testing.T) *entCrud.EntClient[*ent.Client] {
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

	return entCrud.NewEntClient[*ent.Client](client, drv)
}

func newPagingTestLogger() *log.Helper {
	return log.NewHelper(log.DefaultLogger)
}

// paginationV1PagingRequestForTest 构造带页码的分页请求
func paginationV1PagingRequestForTest(page, pageSize uint32) *paginationV1.PagingRequest {
	return &paginationV1.PagingRequest{Page: &page, PageSize: &pageSize}
}

// ============ LeaveTypeRepo ============

func TestLeaveTypeRepoListPaging(t *testing.T) {
	ec := newPagingTestClient(t)
	repo := &LeaveTypeRepo{entClient: ec, log: newPagingTestLogger()}
	ctx := oaViewer.NewSystemViewerContext(context.Background())
	const tid = 9001

	for _, code := range []string{"T1", "T2", "T3"} {
		if _, err := ec.Client().LeaveType.Create().
			SetTenantID(tid).SetCode(code).SetName("n-" + code).
			Save(ctx); err != nil {
			t.Fatalf("seed leave type %s: %v", code, err)
		}
	}

	// 不限页：全量返回，按 ID 倒序
	resp, err := repo.List(ctx, tid, nil)
	if err != nil {
		t.Fatalf("list no-paging: %v", err)
	}
	items := resp.GetItems()
	if resp.GetTotal() != 3 || len(items) != 3 {
		t.Fatalf("no-paging total=%d len=%d, want 3/3", resp.GetTotal(), len(items))
	}
	if items[0].GetCode() != "T3" || items[2].GetCode() != "T1" {
		t.Fatalf("no-paging order desc broken: %v,%v", items[0].GetCode(), items[2].GetCode())
	}

	// 第 1 页（2 条/页）
	page1Resp, err := repo.List(ctx, tid, paginationV1PagingRequestForTest(1, 2))
	if err != nil {
		t.Fatalf("list page1: %v", err)
	}
	page1 := page1Resp.GetItems()
	if page1Resp.GetTotal() != 3 || len(page1) != 2 || page1[0].GetCode() != "T3" || page1[1].GetCode() != "T2" {
		t.Fatalf("page1 wrong: total=%d n=%d first=%s", page1Resp.GetTotal(), len(page1), page1[0].GetCode())
	}

	// 第 2 页：剩余 1 条
	page2Resp, err := repo.List(ctx, tid, paginationV1PagingRequestForTest(2, 2))
	if err != nil {
		t.Fatalf("list page2: %v", err)
	}
	page2 := page2Resp.GetItems()
	if len(page2) != 1 || page2[0].GetCode() != "T1" {
		t.Fatalf("page2 wrong: n=%d code=%s", len(page2), page2[0].GetCode())
	}

	// 越界页：空
	page9Resp, err := repo.List(ctx, tid, paginationV1PagingRequestForTest(9, 2))
	if err != nil {
		t.Fatalf("list page9: %v", err)
	}
	if len(page9Resp.GetItems()) != 0 {
		t.Fatalf("page9 should be empty, got %d", len(page9Resp.GetItems()))
	}

	// 租户隔离
	otherResp, err := repo.List(ctx, tid+1, nil)
	if err != nil {
		t.Fatalf("list other tenant: %v", err)
	}
	if len(otherResp.GetItems()) != 0 {
		t.Fatalf("tenant isolation broken, got %d", len(otherResp.GetItems()))
	}
}

// ============ LeaveBalanceRepo ============

func TestLeaveBalanceRepoListPaging(t *testing.T) {
	ec := newPagingTestClient(t)
	repo := &LeaveBalanceRepo{entClient: ec, log: newPagingTestLogger()}
	ctx := oaViewer.NewSystemViewerContext(context.Background())
	const tid = 9002
	const year = 2026

	for _, u := range []uint32{1, 2, 3} {
		if _, err := ec.Client().LeaveBalance.Create().
			SetTenantID(tid).SetUserID(u).SetLeaveTypeID(1).
			SetYear(year).SetTotalDays(5).SetUsedDays(0).
			Save(ctx); err != nil {
			t.Fatalf("seed balance user %d: %v", u, err)
		}
	}
	// 不同年度的数据不应被查出
	if _, err := ec.Client().LeaveBalance.Create().
		SetTenantID(tid).SetUserID(1).SetLeaveTypeID(1).
		SetYear(year - 1).SetTotalDays(5).SetUsedDays(0).
		Save(ctx); err != nil {
		t.Fatalf("seed prev-year balance: %v", err)
	}

	// 不限页 + 年度过滤
	items, total, err := repo.List(ctx, tid, 0, year, 0, 0)
	if err != nil {
		t.Fatalf("list no-paging: %v", err)
	}
	if total != 3 || len(items) != 3 {
		t.Fatalf("year filter broken: total=%d n=%d", total, len(items))
	}

	// 用户过滤
	items, total, err = repo.List(ctx, tid, 2, year, 0, 0)
	if err != nil {
		t.Fatalf("list user filter: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].GetUserId() != 2 {
		t.Fatalf("user filter broken: total=%d n=%d", total, len(items))
	}

	// 分页切片
	page1, total, err := repo.List(ctx, tid, 0, year, 1, 2)
	if err != nil {
		t.Fatalf("list page1: %v", err)
	}
	if total != 3 || len(page1) != 2 {
		t.Fatalf("page1 wrong: total=%d n=%d", total, len(page1))
	}
	page2, _, err := repo.List(ctx, tid, 0, year, 2, 2)
	if err != nil {
		t.Fatalf("list page2: %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("page2 wrong: n=%d", len(page2))
	}
}

// ============ WorkflowDelegationRepo ============

func TestWorkflowDelegationRepoListPaging(t *testing.T) {
	ec := newPagingTestClient(t)
	repo := &WorkflowDelegationRepo{entClient: ec, log: newPagingTestLogger()}
	ctx := oaViewer.NewSystemViewerContext(context.Background())
	const tid = 9003

	for i := uint32(0); i < 3; i++ {
		if _, err := ec.Client().WorkflowDelegation.Create().
			SetTenantID(tid).
			SetDelegatorUserID(10 + i).
			SetDelegateUserID(20 + i).
			Save(ctx); err != nil {
			t.Fatalf("seed delegation %d: %v", i, err)
		}
	}

	resp, err := repo.List(ctx, tid, nil)
	if err != nil {
		t.Fatalf("list no-paging: %v", err)
	}
	items := resp.GetItems()
	if resp.GetTotal() != 3 || len(items) != 3 {
		t.Fatalf("no-paging broken: total=%d n=%d", resp.GetTotal(), len(items))
	}

	page1Resp, err := repo.List(ctx, tid, paginationV1PagingRequestForTest(1, 2))
	if err != nil {
		t.Fatalf("list page1: %v", err)
	}
	page1 := page1Resp.GetItems()
	if page1Resp.GetTotal() != 3 || len(page1) != 2 {
		t.Fatalf("page1 wrong: total=%d n=%d", page1Resp.GetTotal(), len(page1))
	}
	if page1[0].GetDelegatorUserId() != 12 {
		t.Fatalf("page1 order desc broken: delegator=%d", page1[0].GetDelegatorUserId())
	}
}
