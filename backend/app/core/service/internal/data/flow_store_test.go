package data

import (
	"testing"

	"github.com/tx7do/go-wind-flow/flow"
	"github.com/tx7do/go-wind-flow/format/windjson"
)

// parseForTest 走与 FlowStore.definitionToFlow 相同的解析 + POSITION 修正链路。
func parseForTest(t *testing.T, nodeConfig string) *flow.Definition {
	t.Helper()
	def, err := windjson.Parse([]byte(nodeConfig))
	if err != nil {
		t.Fatalf("windjson.Parse: %v", err)
	}
	patchPositionApproverValues(def, nodeConfig)
	return def
}

// POSITION 审批人在 OA 设计器契约里把职位 ID 写在 approvers[].id，
// windjson 按 code 读取（go-wind-im 约定）——patchPositionApproverValues
// 负责回填。此处验证 v2 图格式、旧线性数组与无 POSITION 场景。
func TestPatchPositionApproverValues(t *testing.T) {
	// v2 图格式：POSITION 的 id 被回填为规则 Value。
	v2 := `{"version":2,"nodes":[
		{"id":"start","type":"START"},
		{"id":"node_0","type":"TASK","approvers":[{"type":"LEADER"},{"type":"POSITION","id":7},{"type":"USER","id":3}],"strategy":"ALL"},
		{"id":"end","type":"END"}
	],"edges":[
		{"from":"start","to":"node_0"},{"from":"node_0","to":"end"}
	]}`
	def := parseForTest(t, v2)
	rules := def.Nodes["node_0"].Approver.Rules
	if len(rules) != 3 {
		t.Fatalf("rules = %v", rules)
	}
	if rules[0].Kind != flow.ApproverLeader {
		t.Fatalf("rule0 = %+v", rules[0])
	}
	if rules[1].Kind != flow.ApproverPosition || rules[1].Value != "7" {
		t.Fatalf("POSITION rule not patched: %+v", rules[1])
	}
	if rules[2].Kind != flow.ApproverUser || rules[2].Value != "3" {
		t.Fatalf("USER rule = %+v", rules[2])
	}

	// 旧线性数组：合成节点同样回填。
	legacy := `[{"approvers":[{"type":"POSITION","id":9}],"strategy":"ANY"}]`
	def = parseForTest(t, legacy)
	node := def.Nodes["__legacy_node_0"]
	if node == nil || node.Approver == nil || len(node.Approver.Rules) != 1 {
		t.Fatalf("legacy node = %+v", node)
	}
	if node.Approver.Rules[0].Kind != flow.ApproverPosition || node.Approver.Rules[0].Value != "9" {
		t.Fatalf("legacy POSITION rule not patched: %+v", node.Approver.Rules[0])
	}
	// 旧格式的 strategy 也应映射为或签。
	if node.Approval.Strategy != flow.StrategyAny {
		t.Fatalf("legacy strategy = %s, want ANY", node.Approval.Strategy)
	}

	// 无 POSITION 场景不误伤其他规则。
	plain := `{"version":2,"nodes":[
		{"id":"start","type":"START"},
		{"id":"node_0","type":"TASK","approvers":[{"type":"LEADER"}],"strategy":"ALL"},
		{"id":"end","type":"END"}
	],"edges":[{"from":"start","to":"node_0"},{"from":"node_0","to":"end"}]}`
	def = parseForTest(t, plain)
	if got := def.Nodes["node_0"].Approver.Rules[0]; got.Kind != flow.ApproverLeader || got.Value != "" {
		t.Fatalf("plain rule mutated: %+v", got)
	}
}

// stripFlowReservedKeys 引擎保留键（flow:* 前缀）从表单 JSON 展示中剔除。
func TestStripFlowReservedKeys(t *testing.T) {
	in := `{"days":3,"flow:joinExpected:join_0":2,"reason":"family"}`
	got := stripFlowReservedKeys(in)
	if got == in {
		t.Fatalf("flow:* key not stripped: %s", got)
	}
	want := `{"days":3,"reason":"family"}`
	if got != want {
		t.Fatalf("strip = %s, want %s", got, want)
	}
	// 无保留键时原样返回（避免无谓的重序列化）。
	plain := `{"days":3,"reason":"family"}`
	if stripFlowReservedKeys(plain) != plain {
		t.Fatal("plain form data should pass through unchanged")
	}
}
