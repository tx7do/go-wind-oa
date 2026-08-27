package service

import (
	"testing"
)

// ===================== parseWorkflowGraph =====================

func TestParseGraph_NewFormat(t *testing.T) {
	jsonStr := `{"version":2,"nodes":[{"id":"start","type":"START"},{"id":"end","type":"END"}],"edges":[{"from":"start","to":"end"}]}`
	graph, err := parseWorkflowGraph(jsonStr)
	if err != nil {
		t.Fatalf("parseWorkflowGraph failed: %v", err)
	}
	if len(graph.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(graph.Nodes))
	}
	if graph.startNode() == nil {
		t.Fatal("missing start node")
	}
}

func TestParseGraph_LegacyFormat(t *testing.T) {
	// 旧格式：节点数组
	jsonStr := `[{"approvers":[{"type":"USER","id":7}],"strategy":"ALL"}]`
	graph, err := parseWorkflowGraph(jsonStr)
	if err != nil {
		t.Fatalf("parseWorkflowGraph legacy failed: %v", err)
	}
	// 应转换为 start→node_0→end（3 节点）
	if len(graph.Nodes) != 3 {
		t.Fatalf("expected 3 nodes (start+task+end), got %d", len(graph.Nodes))
	}
	start := graph.startNode()
	if start == nil {
		t.Fatal("missing start node in legacy conversion")
	}
	if len(start.OutEdges) != 1 {
		t.Fatalf("legacy start should have 1 out edge, got %d", len(start.OutEdges))
	}
}

func TestParseGraph_LegacyMultiNode(t *testing.T) {
	jsonStr := `[{"approvers":[{"type":"USER","id":1}],"strategy":"ALL"},{"approvers":[{"type":"USER","id":2}],"strategy":"ANY"}]`
	graph, err := parseWorkflowGraph(jsonStr)
	if err != nil {
		t.Fatalf("parseWorkflowGraph legacy multi failed: %v", err)
	}
	// start → node_0 → node_1 → end（4 节点）
	if len(graph.Nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(graph.Nodes))
	}
}

func TestParseGraph_Empty(t *testing.T) {
	_, err := parseWorkflowGraph("")
	if err == nil {
		t.Fatal("expected error for empty config")
	}
}

func TestParseGraph_InvalidJSON(t *testing.T) {
	_, err := parseWorkflowGraph("not json at all")
	if err == nil {
		t.Fatal("expected error for invalid json")
	}
}

func TestParseGraph_DuplicateNodeID(t *testing.T) {
	jsonStr := `{"version":2,"nodes":[{"id":"a","type":"START"},{"id":"a","type":"END"}],"edges":[]}`
	_, err := parseWorkflowGraph(jsonStr)
	if err == nil {
		t.Fatal("expected error for duplicate node id")
	}
}

// ===================== validateGraph =====================

func validLinearGraph() *workflowGraph {
	return &workflowGraph{
		Nodes: map[string]*graphNode{
			"start": {ID: "start", Type: nodeTypeStart, OutEdges: []graphEdge{{From: "start", To: "t1"}}},
			"t1":    {ID: "t1", Type: nodeTypeTask, InDegree: 1, OutEdges: []graphEdge{{From: "t1", To: "end"}}, Task: &taskNodeApproverConfig{}},
			"end":   {ID: "end", Type: nodeTypeEnd, InDegree: 1},
		},
	}
}

func TestValidateGraph_ValidLinear(t *testing.T) {
	graph := validLinearGraph()
	if err := validateGraph(graph); err != nil {
		t.Fatalf("valid linear graph rejected: %v", err)
	}
}

func TestValidateGraph_NoStart(t *testing.T) {
	graph := validLinearGraph()
	graph.Nodes["start"].Type = nodeTypeTask
	graph.Nodes["start"].Task = &taskNodeApproverConfig{}
	graph.Nodes["start"].InDegree = 1
	if err := validateGraph(graph); err == nil {
		t.Fatal("expected error for missing START")
	}
}

func TestValidateGraph_MultipleStart(t *testing.T) {
	graph := validLinearGraph()
	graph.Nodes["t1"].Type = nodeTypeStart
	graph.Nodes["t1"].Task = nil
	if err := validateGraph(graph); err == nil {
		t.Fatal("expected error for multiple START nodes")
	}
}

func TestValidateGraph_NoEnd(t *testing.T) {
	graph := validLinearGraph()
	graph.Nodes["end"].Type = nodeTypeTask
	graph.Nodes["end"].Task = &taskNodeApproverConfig{}
	graph.Nodes["end"].OutEdges = []graphEdge{{From: "end", To: "start"}}
	if err := validateGraph(graph); err == nil {
		t.Fatal("expected error for missing END")
	}
}

func TestValidateGraph_UnreachableNode(t *testing.T) {
	graph := validLinearGraph()
	graph.Nodes["orphan"] = &graphNode{ID: "orphan", Type: nodeTypeTask, Task: &taskNodeApproverConfig{}}
	if err := validateGraph(graph); err == nil {
		t.Fatal("expected error for unreachable node")
	}
}

func TestValidateGraph_CycleAllowed(t *testing.T) {
	// Phase 2: 回退边（环路）允许。所有节点仍须可达 END（活性保证）。
	// start→gw，gw 有两条出边：一条到 task（default），一条回 start（构成环）。
	// 所有节点均可经 gw→task→end 到达 END。
	graph := &workflowGraph{
		Nodes: map[string]*graphNode{
			"start": {ID: "start", Type: nodeTypeStart, OutEdges: []graphEdge{{From: "start", To: "gw"}}},
			"gw":    {ID: "gw", Type: nodeTypeExclusiveGateway, InDegree: 2, OutEdges: []graphEdge{{From: "gw", To: "start", Condition: "1 > 0"}, {From: "gw", To: "task", Condition: "default"}}},
			"task":  {ID: "task", Type: nodeTypeTask, InDegree: 1, Task: &taskNodeApproverConfig{}, OutEdges: []graphEdge{{From: "task", To: "end"}}},
			"end":   {ID: "end", Type: nodeTypeEnd, InDegree: 1},
		},
	}
	if err := validateGraph(graph); err != nil {
		t.Fatalf("cycle with exit should be valid in Phase 2: %v", err)
	}
}

func TestValidateGraph_ExclusiveMissingDefault(t *testing.T) {
	graph := &workflowGraph{
		Nodes: map[string]*graphNode{
			"start": {ID: "start", Type: nodeTypeStart, OutEdges: []graphEdge{{From: "start", To: "gw"}}},
			"gw":    {ID: "gw", Type: nodeTypeExclusiveGateway, InDegree: 1, OutEdges: []graphEdge{{From: "gw", To: "a", Condition: "x > 1"}, {From: "gw", To: "b", Condition: "y > 1"}}},
			"a":     {ID: "a", Type: nodeTypeTask, InDegree: 1, Task: &taskNodeApproverConfig{}, OutEdges: []graphEdge{{From: "a", To: "end"}}},
			"b":     {ID: "b", Type: nodeTypeTask, InDegree: 1, Task: &taskNodeApproverConfig{}, OutEdges: []graphEdge{{From: "b", To: "end"}}},
			"end":   {ID: "end", Type: nodeTypeEnd, InDegree: 2},
		},
	}
	if err := validateGraph(graph); err == nil {
		t.Fatal("expected error for EXCLUSIVE_GATEWAY without default")
	}
}

func TestValidateGraph_ExcessiveNodeOutDegree(t *testing.T) {
	// TASK with 2 out edges
	graph := &workflowGraph{
		Nodes: map[string]*graphNode{
			"start": {ID: "start", Type: nodeTypeStart, OutEdges: []graphEdge{{From: "start", To: "t"}, {From: "start", To: "t2"}}},
			"t":     {ID: "t", Type: nodeTypeTask, InDegree: 2, Task: &taskNodeApproverConfig{}, OutEdges: []graphEdge{{From: "t", To: "end"}}},
			"t2":    {ID: "t2", Type: nodeTypeTask, InDegree: 1, Task: &taskNodeApproverConfig{}, OutEdges: []graphEdge{{From: "t2", To: "end"}}},
			"end":   {ID: "end", Type: nodeTypeEnd, InDegree: 2},
		},
	}
	if err := validateGraph(graph); err == nil {
		t.Fatal("expected error for START with 2 out edges")
	}
}

func TestValidateGraph_ValidExclusive(t *testing.T) {
	graph := &workflowGraph{
		Nodes: map[string]*graphNode{
			"start": {ID: "start", Type: nodeTypeStart, OutEdges: []graphEdge{{From: "start", To: "gw"}}},
			"gw":    {ID: "gw", Type: nodeTypeExclusiveGateway, InDegree: 1, OutEdges: []graphEdge{{From: "gw", To: "a", Condition: "x > 1"}, {From: "gw", To: "end", Condition: "default"}}},
			"a":     {ID: "a", Type: nodeTypeTask, InDegree: 1, Task: &taskNodeApproverConfig{}, OutEdges: []graphEdge{{From: "a", To: "end"}}},
			"end":   {ID: "end", Type: nodeTypeEnd, InDegree: 2},
		},
	}
	if err := validateGraph(graph); err != nil {
		t.Fatalf("valid exclusive graph rejected: %v", err)
	}
}

func TestValidateGraph_ValidParallel(t *testing.T) {
	graph := &workflowGraph{
		Nodes: map[string]*graphNode{
			"start": {ID: "start", Type: nodeTypeStart, OutEdges: []graphEdge{{From: "start", To: "t1"}}},
			"t1":    {ID: "t1", Type: nodeTypeTask, InDegree: 1, Task: &taskNodeApproverConfig{}, OutEdges: []graphEdge{{From: "t1", To: "fork"}}},
			"fork":  {ID: "fork", Type: nodeTypeParallelGatewayFork, InDegree: 1, OutEdges: []graphEdge{{From: "fork", To: "ta"}, {From: "fork", To: "tb"}}},
			"ta":    {ID: "ta", Type: nodeTypeTask, InDegree: 1, Task: &taskNodeApproverConfig{}, OutEdges: []graphEdge{{From: "ta", To: "join"}}},
			"tb":    {ID: "tb", Type: nodeTypeTask, InDegree: 1, Task: &taskNodeApproverConfig{}, OutEdges: []graphEdge{{From: "tb", To: "join"}}},
			"join":  {ID: "join", Type: nodeTypeParallelGatewayJoin, InDegree: 2, OutEdges: []graphEdge{{From: "join", To: "end"}}},
			"end":   {ID: "end", Type: nodeTypeEnd, InDegree: 1},
		},
	}
	if err := validateGraph(graph); err != nil {
		t.Fatalf("valid parallel graph rejected: %v", err)
	}
}

func TestValidateGraph_JoinLowInDegree(t *testing.T) {
	// JOIN with in-degree 1 should fail
	graph := &workflowGraph{
		Nodes: map[string]*graphNode{
			"start": {ID: "start", Type: nodeTypeStart, OutEdges: []graphEdge{{From: "start", To: "join"}}},
			"join":  {ID: "join", Type: nodeTypeParallelGatewayJoin, InDegree: 1, OutEdges: []graphEdge{{From: "join", To: "end"}}},
			"end":   {ID: "end", Type: nodeTypeEnd, InDegree: 1},
		},
	}
	if err := validateGraph(graph); err == nil {
		t.Fatal("expected error for JOIN with in-degree < 2")
	}
}

// ===================== evalCondition =====================

func TestEvalCondition_SimpleComparison(t *testing.T) {
	formData := `{"amount": 15000}`
	result, err := evalCondition("amount > 10000", formData)
	if err != nil {
		t.Fatalf("evalCondition failed: %v", err)
	}
	if !result {
		t.Fatal("expected true for amount=15000 > 10000")
	}
}

func TestEvalCondition_FalseComparison(t *testing.T) {
	formData := `{"amount": 5000}`
	result, err := evalCondition("amount > 10000", formData)
	if err != nil {
		t.Fatalf("evalCondition failed: %v", err)
	}
	if result {
		t.Fatal("expected false for amount=5000 > 10000")
	}
}

func TestEvalCondition_MissingField(t *testing.T) {
	formData := `{"other": 1}`
	_, err := evalCondition("amount > 10000", formData)
	if err == nil {
		t.Fatal("expected error for missing field")
	}
}

func TestEvalCondition_SyntaxError(t *testing.T) {
	formData := `{"amount": 1}`
	_, err := evalCondition("amount >>>>", formData)
	if err == nil {
		t.Fatal("expected error for syntax error")
	}
}

func TestEvalCondition_EmptyCondition(t *testing.T) {
	result, err := evalCondition("", `{"a":1}`)
	if err != nil {
		t.Fatalf("evalCondition failed: %v", err)
	}
	if result {
		t.Fatal("expected false for empty condition")
	}
}

func TestEvalCondition_NestedField(t *testing.T) {
	formData := `{"order":{"total":500}}`
	result, err := evalCondition("order.total > 100", formData)
	if err != nil {
		t.Fatalf("evalCondition nested failed: %v", err)
	}
	if !result {
		t.Fatal("expected true for order.total=500 > 100")
	}
}

func TestEvalCondition_NonBoolResult(t *testing.T) {
	formData := `{"amount": 5}`
	// 表达式返回 int 而非 bool → AsBool() 编译期应拒绝
	_, err := evalCondition("amount", formData)
	if err == nil {
		t.Fatal("expected error for non-bool expression")
	}
}
