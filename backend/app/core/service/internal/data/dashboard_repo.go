package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-oa/app/core/service/internal/data/ent"
	"go-wind-oa/app/core/service/internal/data/ent/attendancerecord"
	"go-wind-oa/app/core/service/internal/data/ent/workflowinstance"
	"go-wind-oa/app/core/service/internal/data/ent/workflowlog"
	"go-wind-oa/app/core/service/internal/data/ent/workflowtask"
)

// DistributionRow 是按字段分组聚合后的扫描结构。
// ent sql.Scan 按列名匹配 struct 字段（见 ent scan.go columnName）：
// 优先匹配 `sql` tag，否则字段名小写。因此分组列必须用 sql tag 显式标注列名。
// 两个分布接口分别返回 instance_status / day_result 列，故此 struct 同时声明两个字段，
// 每次扫描只会填充其中之一，由 service 层归并到 Label。
type DistributionRow struct {
	InstanceStatus string `sql:"instance_status"`
	DayResult      string `sql:"day_result"`
	Count          int    `sql:"count"`
}

// TrendRow 是工单创建趋势按日分桶后的结果项（日期 + 次数），由 service 层使用。
type TrendRow struct {
	Date  string
	Count int
}

// DashboardRepo 聚合多张 OA 表做只读统计（admin 分析页数据源）。
// 多租户隔离由 ent Policy 的 EvalQuery 在 prepareQuery 阶段自动注入，
// admin BFF 转发时已携带 viewer 元数据，因此这里不需要手动 where tenant_id。
type DashboardRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewDashboardRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *DashboardRepo {
	return &DashboardRepo{
		log:       ctx.NewLoggerHelper("dashboard/repo/core-service"),
		entClient: entClient,
	}
}

// startOfToday 返回今日零点（本地时区）。
func startOfToday() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// CountWorkflowInstances 统计工作流实例总数（软删/租户隔离由 ent Policy 自动处理）。
func (r *DashboardRepo) CountWorkflowInstances(ctx context.Context) (int, error) {
	count, err := r.entClient.Client().WorkflowInstance.Query().Count(ctx)
	if err != nil {
		r.log.Errorf("count workflow instances failed: %s", err.Error())
		return 0, err
	}
	return count, nil
}

// CountPendingTasks 统计处于 PENDING 状态的工作流任务数（待办积压）。
func (r *DashboardRepo) CountPendingTasks(ctx context.Context) (int, error) {
	count, err := r.entClient.Client().WorkflowTask.Query().
		Where(workflowtask.TaskStatusEQ(workflowtask.TaskStatusPending)).
		Count(ctx)
	if err != nil {
		r.log.Errorf("count pending tasks failed: %s", err.Error())
		return 0, err
	}
	return count, nil
}

// CountTodayNewInstances 统计今日创建的工作流实例数（created_at >= 今日零点）。
func (r *DashboardRepo) CountTodayNewInstances(ctx context.Context) (int, error) {
	today := startOfToday()
	count, err := r.entClient.Client().WorkflowInstance.Query().
		Where(workflowinstance.CreatedAtGTE(today)).
		Count(ctx)
	if err != nil {
		r.log.Errorf("count today new instances failed: %s", err.Error())
		return 0, err
	}
	return count, nil
}

// CountTodayWorkflowActions 统计今日工作流日志记录的审批动作数（created_at >= 今日零点）。
func (r *DashboardRepo) CountTodayWorkflowActions(ctx context.Context) (int, error) {
	today := startOfToday()
	count, err := r.entClient.Client().WorkflowLog.Query().
		Where(workflowlog.CreatedAtGTE(today)).
		Count(ctx)
	if err != nil {
		r.log.Errorf("count today workflow actions failed: %s", err.Error())
		return 0, err
	}
	return count, nil
}

// InstanceCreationTrend 统计近 days 天每日新增工单数。
// ent 的 GroupBy 只支持已有列、不支持 DATE() 等原生表达式，且 prepareQuery 会校验列名，
// 故这里只 Select 出 created_at 单列，在内存按天分桶补零。
// 近 N 天工单数据量可控，单列查询开销可接受。
func (r *DashboardRepo) InstanceCreationTrend(ctx context.Context, days int) ([]TrendRow, error) {
	if days <= 0 {
		days = 7
	}
	today := startOfToday()
	start := today.AddDate(0, 0, -(days - 1))

	type onlyCreated struct {
		CreatedAt time.Time `sql:"created_at"`
	}
	var logs []onlyCreated
	err := r.entClient.Client().WorkflowInstance.Query().
		Where(workflowinstance.CreatedAtGTE(start)).
		Select(workflowinstance.FieldCreatedAt).
		Scan(ctx, &logs)
	if err != nil {
		r.log.Errorf("instance creation trend select failed: %s", err.Error())
		return nil, err
	}

	// 按日期初始化桶，保证无记录的日期也有零值，结果按日期升序。
	loc := today.Location()
	buckets := make([]TrendRow, 0, days)
	idx := make(map[string]int, days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i).Format("2006-01-02")
		idx[d] = i
		buckets = append(buckets, TrendRow{Date: d, Count: 0})
	}
	for _, lg := range logs {
		d := lg.CreatedAt.In(loc).Format("2006-01-02")
		if i, ok := idx[d]; ok {
			buckets[i].Count++
		}
	}
	return buckets, nil
}

// InstanceStatusDistribution 按工作流实例状态（instance_status）分组统计。
// 返回的 InstanceStatus 是枚举值字符串（PENDING/APPROVED/REJECTED/WITHDRAWN），
// 由 service 层透传，前端做 i18n 映射。
// GroupBy 的字段会被自动 select；用 ent.As(ent.Count(),"count") 给计数列起别名，
// 配合 struct 的 sql tag 映射；分组列名即字段名 "instance_status"。
func (r *DashboardRepo) InstanceStatusDistribution(ctx context.Context) ([]DistributionRow, error) {
	var rows []DistributionRow
	err := r.entClient.Client().WorkflowInstance.Query().
		GroupBy(workflowinstance.FieldInstanceStatus).
		Aggregate(ent.As(ent.Count(), "count")).
		Scan(ctx, &rows)
	if err != nil {
		r.log.Errorf("instance status distribution failed: %s", err.Error())
		return nil, err
	}
	return rows, nil
}

// AttendanceDayResultDistribution 按考勤日结果（day_result）分组统计。
// DayResult 为枚举值字符串（NORMAL/LATE/EARLY_LEAVE/ABSENT/...）。
func (r *DashboardRepo) AttendanceDayResultDistribution(ctx context.Context) ([]DistributionRow, error) {
	var rows []DistributionRow
	err := r.entClient.Client().AttendanceRecord.Query().
		GroupBy(attendancerecord.FieldDayResult).
		Aggregate(ent.As(ent.Count(), "count")).
		Scan(ctx, &rows)
	if err != nil {
		r.log.Errorf("attendance day result distribution failed: %s", err.Error())
		return nil, err
	}
	return rows, nil
}
