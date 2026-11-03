/**
 * node_config v2（OA 图格式）↔ designer-json-v1（@go-wind/flow-designer 文档）
 * 双向转换。OA 支持的节点子集：START/END/TASK/EXCLUSIVE_GATEWAY/
 * PARALLEL_GATEWAY_FORK/JOIN/SUBPROCESS；旧线性数组格式自动合成为等价线性图。
 *
 * 转换在纯前端完成——后端 RPC 契约（CreateWorkflowDefinition 的 node_config）
 * 与存储格式保持 version:2 不变。
 */
import { computeLayout } from "@go-wind/flow-designer-core";
import type {
  ApproverRule,
  DesignerDocument,
  DesignerEdge,
  DesignerNode,
} from "@go-wind/flow-designer-core";

// ============ v2（OA）侧形状（与后端 graphNodeJSON 对齐） ============

export interface V2Approver {
  type: "USER" | "LEADER" | "POSITION";
  id?: number;
}

export interface V2Node {
  id: string;
  type:
    | "START"
    | "END"
    | "TASK"
    | "EXCLUSIVE_GATEWAY"
    | "PARALLEL_GATEWAY_FORK"
    | "PARALLEL_GATEWAY_JOIN"
    | "SUBPROCESS";
  approvers?: V2Approver[];
  strategy?: string;
  subprocessDefinition?: string;
}

export interface V2Edge {
  from: string;
  to: string;
  condition?: string;
}

export interface V2Graph {
  version: 2;
  nodes: V2Node[];
  edges: V2Edge[];
}

// ============ 解析入口：兼容 v2 图与旧线性数组 ============

/** 解析 node_config 文本为 v2 图；旧线性数组自动合成 START/END 线性图。 */
export function parseNodeConfig(text: string): V2Graph {
  const parsed = JSON.parse(text) as unknown;
  if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
    const graph = parsed as { version?: number; nodes?: V2Node[]; edges?: V2Edge[] };
    if (graph.version === 2 && Array.isArray(graph.nodes)) {
      return { version: 2, nodes: graph.nodes, edges: Array.isArray(graph.edges) ? graph.edges : [] };
    }
    throw new Error("node_config 不是 version:2 图格式");
  }
  if (Array.isArray(parsed) && parsed.length > 0) {
    return legacyToGraph(parsed as LegacyNode[]);
  }
  throw new Error("node_config 为空或格式不识别");
}

/** 旧线性节点数组（ensureWorkflowDefinition 引导格式）→ 等价线性 v2 图。 */
export function legacyToGraph(legacy: LegacyNode[]): V2Graph {
  const nodes: V2Node[] = [{ id: "__legacy_start", type: "START" }];
  const edges: V2Edge[] = [];
  let prev = "__legacy_start";
  legacy.forEach((ln, i) => {
    const id = `__legacy_node_${i}`;
    const node: V2Node = { id, type: "TASK" };
    const approvers = Array.isArray(ln.approvers) ? ln.approvers : [];
    if (approvers.length > 0) {
      node.approvers = approvers;
    } else if (ln.approver_type === "USER" && ln.approver) {
      node.approvers = [{ type: "USER", id: ln.approver }];
    } else {
      node.approvers = [{ type: "LEADER" }];
    }
    node.strategy = String(ln.strategy ?? "ALL").toUpperCase() === "ANY" ? "ANY" : "ALL";
    nodes.push(node);
    edges.push({ from: prev, to: id });
    prev = id;
  });
  nodes.push({ id: "__legacy_end", type: "END" });
  edges.push({ from: prev, to: "__legacy_end" });
  return { version: 2, nodes, edges };
}

export interface LegacyNode {
  id?: string;
  approvers?: V2Approver[];
  strategy?: string;
  approver_type?: string;
  approver?: number;
}

// ============ v2 → 设计器文档 ============

const V2_TO_DESIGNER_TYPE: Record<V2Node["type"], DesignerNode["type"]> = {
  START: "startEvent",
  END: "endEvent",
  TASK: "userTask",
  EXCLUSIVE_GATEWAY: "exclusiveGateway",
  PARALLEL_GATEWAY_FORK: "parallelGateway",
  PARALLEL_GATEWAY_JOIN: "parallelGateway",
  SUBPROCESS: "callActivity",
};

/** v2 图 → 设计器文档。坐标系交给 computeLayout（v2 不存布局）。
 * key 为空时兜底 "new_process"——documentToXML 用它作 process id 与
 * BPMNDI 引用，空串会导致 bpmn-js 导入报 rootElement required。 */
export function graphToDesigner(graph: V2Graph, key = "", name = ""): DesignerDocument {
  const safeKey = key || "new_process";
  const layout = computeLayout(toDesignerDocWithoutLayout(graph, safeKey, name));
  const doc = toDesignerDocWithoutLayout(graph, safeKey, name);
  for (const node of doc.nodes) {
    const pos = layout.nodes[node.id];
    if (pos) node.ui = { x: pos.x, y: pos.y };
  }
  return doc;
}

function toDesignerDocWithoutLayout(graph: V2Graph, key: string, name: string): DesignerDocument {
  const nodes: DesignerNode[] = graph.nodes.map((n) => {
    const base: DesignerNode = {
      id: n.id,
      type: V2_TO_DESIGNER_TYPE[n.type] ?? "userTask",
      name: n.type === "TASK" || n.type === "SUBPROCESS" ? n.id : undefined,
      ui: undefined,
    };
    if (n.type === "TASK") {
      base.approver = {
        rules: (n.approvers ?? []).map(approverToRule),
        strategy: String(n.strategy ?? "ALL").toUpperCase() === "ANY" ? "ANY" : "ALL",
      };
    }
    if (n.type === "SUBPROCESS" && n.subprocessDefinition) {
      const [subKey, subVer] = splitSubprocessRef(n.subprocessDefinition);
      base.subprocessKey = subKey;
      base.subprocessVersion = subVer;
    }
    return base;
  });
  const edges: DesignerEdge[] = graph.edges.map((e, i) => ({
    id: `e_${i + 1}`,
    source: e.from,
    target: e.to,
    ...(e.condition && e.condition !== "default"
      ? { condition: e.condition }
      : e.condition === "default"
        ? { isDefault: true }
        : {}),
  }));
  return { version: 1, key, name, nodes, edges };
}

function approverToRule(a: V2Approver): ApproverRule {
  if (a.type === "USER") return { kind: "USER", value: String(a.id ?? "") };
  if (a.type === "POSITION") return { kind: "POSITION", value: String(a.id ?? "") };
  return { kind: "LEADER" };
}

function splitSubprocessRef(ref: string): [string, number] {
  const idx = ref.lastIndexOf(":");
  if (idx > 0) {
    const ver = Number(ref.slice(idx + 1).replace(/^v/i, ""));
    if (Number.isFinite(ver) && ver > 0) return [ref.slice(0, idx), ver];
  }
  return [ref, 0];
}

// ============ 设计器文档 → v2 ============

const DESIGNER_TO_V2_TYPE: Partial<Record<DesignerNode["type"], V2Node["type"]>> = {
  startEvent: "START",
  endEvent: "END",
  userTask: "TASK",
  exclusiveGateway: "EXCLUSIVE_GATEWAY",
  callActivity: "SUBPROCESS",
};

/**
 * 设计器文档 → v2 图 JSON 文本。
 * 设计器画布允许添加 OA 子集之外的节点（脚本/定时器/消息等），转换时
 * fail-fast 列出——这些能力 v2 格式暂不承载。
 */
export function designerToGraph(doc: DesignerDocument): V2Graph {
  const nodes: V2Node[] = [];
  const unsupported: string[] = [];

  for (const n of doc.nodes) {
    const v2Type = DESIGNER_TO_V2_TYPE[n.type];
    if (!v2Type) {
      unsupported.push(`${n.name || n.id}(${n.type})`);
      continue;
    }
    const node: V2Node = { id: n.id, type: v2Type };
    if (v2Type === "TASK") {
      const rules = n.approver?.rules ?? [];
      const approvers: V2Approver[] = [];
      for (const rule of rules) {
        if (rule.kind === "LEADER") {
          approvers.push({ type: "LEADER" });
          continue;
        }
        if (rule.kind === "USER" || rule.kind === "POSITION") {
          const id = Number(rule.value);
          if (!rule.value || !Number.isInteger(id) || id <= 0) {
            throw new Error(
              `节点「${n.name || n.id}」的 ${rule.kind} 审批人需填写数字 ID${rule.kind === "POSITION" ? "（职位ID）" : "（用户ID）"}`,
            );
          }
          approvers.push(rule.kind === "USER" ? { type: "USER", id } : { type: "POSITION", id });
          continue;
        }
        throw new Error(`节点「${n.name || n.id}」的审批人类型 ${rule.kind} 暂不支持保存`);
      }
      if (approvers.length === 0) {
        throw new Error(`审批节点「${n.name || n.id}」未配置审批人`);
      }
      node.approvers = approvers;
      node.strategy = n.approver?.strategy === "ANY" ? "ANY" : "ALL";
    }
    if (v2Type === "SUBPROCESS") {
      if (!n.subprocessKey) {
        throw new Error(`子流程节点「${n.name || n.id}」未配置子流程定义`);
      }
      node.subprocessDefinition =
        n.subprocessVersion && n.subprocessVersion > 0
          ? `${n.subprocessKey}:${n.subprocessVersion}`
          : n.subprocessKey;
    }
    nodes.push(node);
  }
  if (unsupported.length > 0) {
    throw new Error(`存在 OA 格式暂不支持保存的节点：${unsupported.join("、")}`);
  }

  const edges: V2Edge[] = doc.edges.map((e) => {
    const edge: V2Edge = { from: e.source, to: e.target };
    if (e.condition) edge.condition = e.condition;
    else if (e.isDefault) edge.condition = "default";
    return edge;
  });

  return { version: 2, nodes, edges };
}

/** 设计器文档 → node_config 文本（pretty JSON）。 */
export function designerToNodeConfig(doc: DesignerDocument): string {
  return JSON.stringify(designerToGraph(doc), null, 2);
}
