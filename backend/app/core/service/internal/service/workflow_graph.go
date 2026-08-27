package service

import (
	"context"
	"encoding/json"
	"fmt"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

// ===================== 图解析 =====================

// parseWorkflowGraph 解析 node_config JSON 文本为图内存表示。
// 兼容旧格式（JSON 节点数组）：自动转换为等价线性图
// start→node0→…→nodeN→end，所有 TASK 节点的 approvers/strategy 原样保留。
func parseWorkflowGraph(nodeConfig string) (*workflowGraph, error) {
	if nodeConfig == "" {
		return nil, oaV1.ErrorBadRequest("empty node config")
	}

	// 尝试解析为图格式（version:2）。
	var graphJSON workflowGraphJSON
	if err := json.Unmarshal([]byte(nodeConfig), &graphJSON); err == nil && graphJSON.Version == 2 {
		return buildGraphFromJSON(&graphJSON)
	}

	// 尝试解析为旧格式（节点数组）。
	var legacyNodes []taskNodeApproverConfig
	if err := json.Unmarshal([]byte(nodeConfig), &legacyNodes); err == nil {
		return buildGraphFromLegacy(legacyNodes)
	}

	return nil, oaV1.ErrorBadRequest("invalid node config json")
}

// buildGraphFromJSON 从图格式 JSON 构建内存图。
func buildGraphFromJSON(j *workflowGraphJSON) (*workflowGraph, error) {
	graph := &workflowGraph{Nodes: make(map[string]*graphNode)}

	// 构建节点。
	for _, nj := range j.Nodes {
		if nj.ID == "" || nj.Type == "" {
			return nil, oaV1.ErrorBadRequest("node missing id or type")
		}
		if _, exists := graph.Nodes[nj.ID]; exists {
			return nil, oaV1.ErrorBadRequest(fmt.Sprintf("duplicate node id: %s", nj.ID))
		}
		node := &graphNode{ID: nj.ID, Type: nj.Type}
		if nj.Type == nodeTypeTask {
			node.Task = &taskNodeApproverConfig{
				Approvers: nj.Approvers,
				Strategy:  nj.Strategy,
			}
			// 旧格式兼容字段。
			node.Task.ApproverType = nj.ApproverType
			node.Task.Approver = nj.Approver
		}
		if nj.Type == nodeTypeSubprocess {
			node.SubprocessDefinition = nj.SubprocessDefinition
		}
		graph.Nodes[nj.ID] = node
	}

	// 构建边 + 入度。
	for _, ej := range j.Edges {
		from, okFrom := graph.Nodes[ej.From]
		to, okTo := graph.Nodes[ej.To]
		if !okFrom || !okTo {
			return nil, oaV1.ErrorBadRequest("edge references unknown node")
		}
		edge := graphEdge{From: ej.From, To: ej.To, Condition: ej.Condition}
		from.OutEdges = append(from.OutEdges, edge)
		to.InDegree++
	}

	return graph, nil
}

// buildGraphFromLegacy 从旧格式节点数组构建等价线性图：
// start → node_0 → node_1 → … → node_N → end
func buildGraphFromLegacy(legacy []taskNodeApproverConfig) (*workflowGraph, error) {
	if len(legacy) == 0 {
		return nil, oaV1.ErrorBadRequest("empty legacy node config")
	}

	graph := &workflowGraph{Nodes: make(map[string]*graphNode)}

	// 极其特殊保留字，避免与用户定义 id 冲突（图格式中用户不可能用这些 id）。
	start := &graphNode{ID: "__legacy_start", Type: nodeTypeStart}
	end := &graphNode{ID: "__legacy_end", Type: nodeTypeEnd}
	graph.Nodes[start.ID] = start
	graph.Nodes[end.ID] = end

	prev := start
	for i, ln := range legacy {
		nodeID := fmt.Sprintf("__legacy_node_%d", i)
		node := &graphNode{
			ID:   nodeID,
			Type: nodeTypeTask,
			Task: &taskNodeApproverConfig{
				Approvers: ln.Approvers,
				Strategy:  ln.Strategy,
			},
		}
		node.Task.ApproverType = ln.ApproverType
		node.Task.Approver = ln.Approver
		graph.Nodes[nodeID] = node
		// prev → node
		prev.OutEdges = append(prev.OutEdges, graphEdge{From: prev.ID, To: nodeID})
		node.InDegree++
		prev = node
	}
	// last → end
	prev.OutEdges = append(prev.OutEdges, graphEdge{From: prev.ID, To: end.ID})
	end.InDegree++

	return graph, nil
}

// startNode 返回图的 START 节点（恰好一个）。
func (g *workflowGraph) startNode() *graphNode {
	for _, n := range g.Nodes {
		if n.Type == nodeTypeStart {
			return n
		}
	}
	return nil
}

// ===================== 图校验 =====================

// validateGraph 校验图结构合法性。校验失败返回 BadRequest。
func validateGraph(graph *workflowGraph) error {
	// 恰一个 START、一个 END。
	startCount, endCount := 0, 0
	for _, n := range graph.Nodes {
		switch n.Type {
		case nodeTypeStart:
			startCount++
		case nodeTypeEnd:
			endCount++
		case nodeTypeTask:
		case nodeTypeExclusiveGateway:
		case nodeTypeParallelGatewayFork:
		case nodeTypeParallelGatewayJoin:
		case nodeTypeSubprocess:
		default:
			return oaV1.ErrorBadRequest(fmt.Sprintf("unknown node type: %s", n.Type))
		}
	}
	if startCount != 1 {
		return oaV1.ErrorBadRequest("graph must have exactly one START node")
	}
	if endCount != 1 {
		return oaV1.ErrorBadRequest("graph must have exactly one END node")
	}

	// 节点级出入度约束。
	for _, n := range graph.Nodes {
		switch n.Type {
		case nodeTypeStart:
			if len(n.OutEdges) != 1 {
				return oaV1.ErrorBadRequest("START must have exactly one out edge")
			}
			if n.InDegree != 0 {
				return oaV1.ErrorBadRequest("START must have zero in edges")
			}
		case nodeTypeEnd:
			if n.InDegree < 1 {
				return oaV1.ErrorBadRequest("END must have at least one in edge")
			}
			if len(n.OutEdges) != 0 {
				return oaV1.ErrorBadRequest("END must have zero out edges")
			}
		case nodeTypeTask:
			if len(n.OutEdges) != 1 {
				return oaV1.ErrorBadRequest("TASK must have exactly one out edge")
			}
			if n.InDegree != 1 {
				return oaV1.ErrorBadRequest("TASK must have exactly one in edge")
			}
		case nodeTypeSubprocess:
			if len(n.OutEdges) != 1 {
				return oaV1.ErrorBadRequest("SUBPROCESS must have exactly one out edge")
			}
			if n.InDegree != 1 {
				return oaV1.ErrorBadRequest("SUBPROCESS must have exactly one in edge")
			}
		case nodeTypeExclusiveGateway:
			// 出边全有 condition（表达式或 "default"），且恰一条 default。
			if len(n.OutEdges) < 2 {
				return oaV1.ErrorBadRequest("EXCLUSIVE_GATEWAY must have at least 2 out edges")
			}
			defaultCount := 0
			for _, e := range n.OutEdges {
				if e.Condition == "" {
					return oaV1.ErrorBadRequest("EXCLUSIVE_GATEWAY out edge missing condition")
				}
				if e.Condition == conditionDefault {
					defaultCount++
				}
			}
			if defaultCount != 1 {
				return oaV1.ErrorBadRequest("EXCLUSIVE_GATEWAY must have exactly one default out edge")
			}
		case nodeTypeParallelGatewayFork:
			if len(n.OutEdges) < 2 {
				return oaV1.ErrorBadRequest("FORK must have at least 2 out edges")
			}
			for _, e := range n.OutEdges {
				if e.Condition != "" {
					return oaV1.ErrorBadRequest("FORK out edge must not have condition")
				}
			}
		case nodeTypeParallelGatewayJoin:
			if n.InDegree < 2 {
				return oaV1.ErrorBadRequest("JOIN must have at least 2 in edges")
			}
			if len(n.OutEdges) != 1 {
				return oaV1.ErrorBadRequest("JOIN must have exactly one out edge")
			}
		}
	}

	// 从 START 可达所有节点；所有节点可达 END（活性保证：即使有环，每个节点都有出口路径到 END）。
	if err := checkReachability(graph); err != nil {
		return err
	}

	return nil
}

// checkReachability 从 START BFS，验证所有节点可达且所有节点可达 END。
func checkReachability(graph *workflowGraph) error {
	start := graph.startNode()
	if start == nil {
		return oaV1.ErrorBadRequest("missing START node")
	}

	// 从 START 正向可达性。
	reachable := make(map[string]bool)
	queue := []*graphNode{start}
	reachable[start.ID] = true
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, e := range cur.OutEdges {
			next := graph.Nodes[e.To]
			if next == nil {
				return oaV1.ErrorBadRequest("edge to unknown node: " + e.To)
			}
			if !reachable[next.ID] {
				reachable[next.ID] = true
				queue = append(queue, next)
			}
		}
	}
	for id, node := range graph.Nodes {
		if !reachable[id] {
			return oaV1.ErrorBadRequest("node not reachable from START: " + id)
		}
		if node.Type == nodeTypeEnd {
			// END 必须可达。
		}
	}

	// 从 END 逆向可达性（所有节点必须能到达 END）。
	// 构建逆向邻接表。
	reverseEdges := make(map[string][]string)
	for _, n := range graph.Nodes {
		for _, e := range n.OutEdges {
			reverseEdges[e.To] = append(reverseEdges[e.To], n.ID)
		}
	}
	var endNode *graphNode
	for _, n := range graph.Nodes {
		if n.Type == nodeTypeEnd {
			endNode = n
			break
		}
	}
	canReachEnd := make(map[string]bool)
	queue2 := []*graphNode{endNode}
	canReachEnd[endNode.ID] = true
	for len(queue2) > 0 {
		cur := queue2[0]
		queue2 = queue2[1:]
		for _, predID := range reverseEdges[cur.ID] {
			if !canReachEnd[predID] {
				canReachEnd[predID] = true
				queue2 = append(queue2, graph.Nodes[predID])
			}
		}
	}
	for id := range graph.Nodes {
		if !canReachEnd[id] {
			return oaV1.ErrorBadRequest("node cannot reach END: " + id)
		}
	}

	return nil
}

// graphOfInstance 读取实例所属定义并解析图配置。
func (s *WorkflowService) graphOfInstance(ctx context.Context, instanceID, tenantID uint32) (*workflowGraph, error) {
	nodeConfig, err := s.instanceRepo.GetDefinitionNodeConfig(ctx, instanceID, tenantID)
	if err != nil {
		return nil, err
	}
	return parseWorkflowGraph(nodeConfig)
}

// resolveApprovers 解析 TASK 节点审批人列表。USER 直取；LEADER 解析申请人主组织负责人；
// POSITION 解析职位在职持有者（可展开多人）。结果去重且保序。
func (s *WorkflowService) resolveApprovers(
	ctx context.Context, tenantID, applicantUserID uint32, node *taskNodeApproverConfig,
) ([]uint32, error) {
	specs := node.normalizedApprovers()
	if len(specs) == 0 {
		return nil, oaV1.ErrorBadRequest("node has no approver")
	}

	seen := make(map[uint32]struct{}, len(specs))
	approvers := make([]uint32, 0, len(specs))
	appendUserID := func(id uint32) {
		if id == 0 {
			return
		}
		if _, dup := seen[id]; dup {
			return
		}
		seen[id] = struct{}{}
		approvers = append(approvers, id)
	}

	for _, spec := range specs {
		switch spec.Type {
		case approverTypeUser:
			appendUserID(s.applyDelegation(ctx, tenantID, spec.ID))
		case approverTypeLeader:
			leaderID, err := s.resolverRepo.ResolveOrgLeader(ctx, tenantID, applicantUserID)
			if err != nil {
				return nil, err
			}
			appendUserID(s.applyDelegation(ctx, tenantID, leaderID))
		case approverTypePosition:
			holders, err := s.resolverRepo.ResolvePositionHolders(ctx, tenantID, spec.ID)
			if err != nil {
				return nil, err
			}
			for _, holder := range holders {
				appendUserID(s.applyDelegation(ctx, tenantID, holder))
			}
		default:
			return nil, oaV1.ErrorBadRequest("unsupported approver type")
		}
	}
	if len(approvers) == 0 {
		return nil, oaV1.ErrorBadRequest("node has no approver")
	}
	return approvers, nil
}

// applyDelegation 返回 userID 的代理人 ID（若已设委托），否则返回 userID 本身。
func (s *WorkflowService) applyDelegation(ctx context.Context, tenantID, userID uint32) uint32 {
	delegate, err := s.delegationRepo.ResolveDelegate(ctx, tenantID, userID)
	if err != nil {
		s.log.Errorf("resolve delegation for user %d failed: %s", userID, err.Error())
		return userID
	}
	if delegate != 0 {
		return delegate
	}
	return userID
}
