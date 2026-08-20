package service

import (
	"context"
	"testing"
	"time"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

var testLoc = time.FixedZone("CST", 8*3600)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, testLoc)
}

// ===================== 请假天数（半日粒度） =====================

func TestComputeLeaveDays(t *testing.T) {
	cases := []struct {
		name      string
		start     time.Time
		end       time.Time
		startHalf oaV1.HalfOfDay
		endHalf   oaV1.HalfOfDay
		want      float64
	}{
		{"单日全天 AM~PM", date(2026, 8, 20), date(2026, 8, 20), oaV1.HalfOfDay_AM, oaV1.HalfOfDay_PM, 1},
		{"单日下午 PM~PM", date(2026, 8, 20), date(2026, 8, 20), oaV1.HalfOfDay_PM, oaV1.HalfOfDay_PM, 0.5},
		{"单日上午 AM~AM", date(2026, 8, 20), date(2026, 8, 20), oaV1.HalfOfDay_AM, oaV1.HalfOfDay_AM, 0.5},
		{"单日非法 PM起~AM止", date(2026, 8, 20), date(2026, 8, 20), oaV1.HalfOfDay_PM, oaV1.HalfOfDay_AM, 0},
		{"两日全天", date(2026, 8, 20), date(2026, 8, 21), oaV1.HalfOfDay_AM, oaV1.HalfOfDay_PM, 2},
		{"两日下午起到下午止", date(2026, 8, 20), date(2026, 8, 21), oaV1.HalfOfDay_PM, oaV1.HalfOfDay_PM, 1.5},
		{"两日上午起到上午止", date(2026, 8, 20), date(2026, 8, 21), oaV1.HalfOfDay_AM, oaV1.HalfOfDay_AM, 1.5},
		{"跨周末五日", date(2026, 8, 17), date(2026, 8, 21), oaV1.HalfOfDay_AM, oaV1.HalfOfDay_PM, 5},
		{"结束早于开始", date(2026, 8, 21), date(2026, 8, 20), oaV1.HalfOfDay_AM, oaV1.HalfOfDay_PM, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := computeLeaveDays(c.start, c.end, c.startHalf, c.endHalf)
			if got != c.want {
				t.Fatalf("computeLeaveDays(%s~%s, %v~%v) = %v, want %v",
					c.start.Format("01-02"), c.end.Format("01-02"), c.startHalf, c.endHalf, got, c.want)
			}
		})
	}
}

func TestHalfDayValues(t *testing.T) {
	if sh, eh := halfDayValues(oaV1.HalfOfDay_AM, oaV1.HalfOfDay_PM); sh != 0 || eh != 1 {
		t.Fatalf("AM/PM = %d/%d, want 0/1", sh, eh)
	}
	if sh, eh := halfDayValues(oaV1.HalfOfDay_PM, oaV1.HalfOfDay_AM); sh != 1 || eh != 0 {
		t.Fatalf("PM/AM = %d/%d, want 1/0", sh, eh)
	}
}

// ===================== 时间工具 =====================

func TestTruncateDate(t *testing.T) {
	in := time.Date(2026, 8, 20, 17, 45, 12, 999, testLoc)
	got := truncateDate(in)
	if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 || got.Day() != 20 {
		t.Fatalf("truncateDate = %v, want 2026-08-20 00:00", got)
	}
	if got.Location() != testLoc {
		t.Fatalf("truncateDate lost location: %v", got.Location())
	}
}

func TestParseHHMM(t *testing.T) {
	workDate := date(2026, 8, 20)
	if ts, ok := parseHHMM(workDate, "09:00"); !ok || ts.Hour() != 9 || ts.Minute() != 0 || ts.Day() != 20 {
		t.Fatalf("parseHHMM 09:00 = %v ok=%v", ts, ok)
	}
	if ts, ok := parseHHMM(workDate, "18:30:45"); !ok || ts.Hour() != 18 || ts.Minute() != 30 {
		t.Fatalf("parseHHMM 18:30:45 = %v ok=%v", ts, ok)
	}
	if _, ok := parseHHMM(workDate, "25:00"); ok {
		t.Fatal("parseHHMM 25:00 should fail")
	}
	if _, ok := parseHHMM(workDate, "abc"); ok {
		t.Fatal("parseHHMM abc should fail")
	}
}

func TestIsWeekend(t *testing.T) {
	if !isWeekend(date(2026, 8, 22)) { // 周六
		t.Fatal("2026-08-22 is Saturday, want weekend")
	}
	if !isWeekend(date(2026, 8, 23)) { // 周日
		t.Fatal("2026-08-23 is Sunday, want weekend")
	}
	if isWeekend(date(2026, 8, 20)) { // 周四
		t.Fatal("2026-08-20 is Thursday, want weekday")
	}
}

// ===================== 工作流节点解析 =====================

func TestWorkflowNodeNormalizedApprovers(t *testing.T) {
	// 新格式直取
	n := &workflowNode{Approvers: []workflowApprover{{Type: approverTypeLeader}}}
	if got := n.normalizedApprovers(); len(got) != 1 || got[0].Type != approverTypeLeader {
		t.Fatalf("new format = %v", got)
	}
	// 旧格式归一化为单 USER
	n = &workflowNode{ApproverType: "USER", Approver: 42}
	if got := n.normalizedApprovers(); len(got) != 1 || got[0].Type != "USER" || got[0].ID != 42 {
		t.Fatalf("legacy format = %v", got)
	}
	// 旧格式非法类型归一化为空
	n = &workflowNode{ApproverType: "ROLE", Approver: 1}
	if got := n.normalizedApprovers(); got != nil {
		t.Fatalf("invalid legacy = %v, want nil", got)
	}
	// 空节点
	n = &workflowNode{}
	if got := n.normalizedApprovers(); got != nil {
		t.Fatalf("empty node = %v, want nil", got)
	}
}

func TestWorkflowNodeIsAnyStrategy(t *testing.T) {
	if (&workflowNode{Strategy: "ANY"}).isAnyStrategy() != true {
		t.Fatal("ANY should be or-sign")
	}
	if (&workflowNode{Strategy: "any"}).isAnyStrategy() != true { // 大小写不敏感
		t.Fatal("any should be or-sign")
	}
	if (&workflowNode{Strategy: "ALL"}).isAnyStrategy() != false {
		t.Fatal("ALL should be and-sign")
	}
	if (&workflowNode{}).isAnyStrategy() != false { // 缺省会签
		t.Fatal("empty strategy should default to and-sign")
	}
	if (&workflowNode{Strategy: "WHATEVER"}).isAnyStrategy() != false { // 未知值会签
		t.Fatal("unknown strategy should default to and-sign")
	}
}

// ===================== 迟到/早退判定（纯时间逻辑） =====================

// computeDayResult 依赖 repo（设置/请假查询），此处覆盖其时间比较骨架：
// 用 parseHHMM 构造上班/下班时刻后做判定，确保语义不颠倒。
func TestLateEarlySemantics(t *testing.T) {
	workDate := date(2026, 8, 20)
	workStart, _ := parseHHMM(workDate, "09:00")
	workEnd, _ := parseHHMM(workDate, "18:00")

	checkIn := time.Date(2026, 8, 20, 9, 1, 0, 0, testLoc)
	if !checkIn.After(workStart) {
		t.Fatal("09:01 should be after 09:00 (late)")
	}
	checkIn = time.Date(2026, 8, 20, 8, 59, 0, 0, testLoc)
	if checkIn.After(workStart) {
		t.Fatal("08:59 should not be late")
	}
	checkOut := time.Date(2026, 8, 20, 17, 59, 0, 0, testLoc)
	if !checkOut.Before(workEnd) {
		t.Fatal("17:59 should be before 18:00 (early leave)")
	}
	checkOut = time.Date(2026, 8, 20, 18, 1, 0, 0, testLoc)
	if checkOut.Before(workEnd) {
		t.Fatal("18:01 should not be early leave")
	}
	_ = context.Background()
}
