package task

// 内置系统周期任务：由 asynq periodic scheduler 触发，服务启动时注册，
// 不进 oa_task 表（非用户管理任务）。
const (
	// WorkflowTimeoutScanTaskType 工作流待办超时催办/升级扫描，每小时整点。
	WorkflowTimeoutScanTaskType = "oa:workflow:timeout-scan"
	// AttendanceSettleTaskType 考勤昨日结算，每日 00:30。
	AttendanceSettleTaskType = "oa:attendance:settle"
)

type WorkflowTimeoutScanData struct{}

type AttendanceSettleData struct{}
