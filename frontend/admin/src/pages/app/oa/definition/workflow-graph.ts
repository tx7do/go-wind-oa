/**
 * 流程图编辑器的纯逻辑模块：图草稿类型、节点/边操作与双向序列化。
 * 除 nodeTypeLabel 需要翻译外均为纯函数，与组件状态解耦，便于复用与单测。
 */
import { $t, $te } from "@/core/i18n";

/** 审批人草稿。LEADER 无需 id（按提交人动态解析）。 */
export interface ApproverDraft {
  type: "USER" | "LEADER" | "POSITION";
  id?: number;
}

/** 图节点草稿。TASK 节点额外携带审批人配置。 */
export interface GraphNodeDraft {
  id: string;
  type:
    | "START"
    | "END"
    | "TASK"
    | "EXCLUSIVE_GATEWAY"
    | "PARALLEL_GATEWAY_FORK"
    | "PARALLEL_GATEWAY_JOIN"
    | "SUBPROCESS";
  strategy?: "ALL" | "ANY";
  approvers?: ApproverDraft[];
  subprocessDefinition?: string;
  x?: number;
  y?: number;
}

/** 图边草稿。EXCLUSIVE_GATEWAY 出边的 condition 为表达式或 "default"。 */
export interface GraphEdgeDraft {
  from: string;
  to: string;
  condition?: string;
}

/** 表单字段草稿（options 为逗号分隔字符串，仅 select 用）。 */
export interface FieldDraft {
  key: string;
  label: string;
  type: "text" | "textarea" | "number" | "date" | "select";
  required: boolean;
  options: string;
}

/** 合法节点类型集合：反序列化时用于过滤未知类型。 */
export const KNOWN_NODE_TYPES: ReadonlySet<string> = new Set([
  "START",
  "END",
  "TASK",
  "EXCLUSIVE_GATEWAY",
  "PARALLEL_GATEWAY_FORK",
  "PARALLEL_GATEWAY_JOIN",
  "SUBPROCESS",
]);

export const NODE_VISUAL_SIZE: Record<string, { w: number; h: number }> = {
  START: { w: 50, h: 50 },
  END: { w: 50, h: 50 },
  TASK: { w: 90, h: 40 },
  EXCLUSIVE_GATEWAY: { w: 60, h: 60 },
  PARALLEL_GATEWAY_FORK: { w: 60, h: 60 },
  PARALLEL_GATEWAY_JOIN: { w: 60, h: 60 },
  SUBPROCESS: { w: 90, h: 40 },
};

/** 节点类型展示名（词条缺失时回退原始值）。 */
export function nodeTypeLabel(type: string): string {
  const key = `pages.oa.definition.nodeType.${type}`;
  return $te(key) ? $t(key) : type;
}

let nodeCounter = 0;

/** 新建图节点草稿（含类型默认字段）。 */
export function makeGraphNode(type: GraphNodeDraft["type"]): GraphNodeDraft {
  const prefix =
    type === "START"
      ? "start"
      : type === "END"
        ? "end"
        : type === "EXCLUSIVE_GATEWAY"
          ? "gw"
          : type === "PARALLEL_GATEWAY_FORK"
            ? "fork"
            : type === "PARALLEL_GATEWAY_JOIN"
              ? "join"
              : type === "SUBPROCESS"
                ? "sub"
                : "node";
  const node: GraphNodeDraft = { id: `${prefix}_${nodeCounter++}`, type };
  if (type === "TASK") {
    node.strategy = "ALL";
    node.approvers = [{ type: "LEADER" }];
  }
  if (type === "SUBPROCESS") {
    node.subprocessDefinition = "";
  }
  return node;
}

/** 切换节点类型时调整字段（补齐/清除类型专属配置）。 */
export function applyNodeTypeChange(node: GraphNodeDraft) {
  if (node.type === "TASK") {
    if (!node.approvers) node.approvers = [{ type: "LEADER" }];
    if (!node.strategy) node.strategy = "ALL";
    node.subprocessDefinition = undefined;
  } else if (node.type === "SUBPROCESS") {
    node.approvers = undefined;
    node.strategy = undefined;
    if (!node.subprocessDefinition) node.subprocessDefinition = "";
  } else {
    node.approvers = undefined;
    node.strategy = undefined;
    node.subprocessDefinition = undefined;
  }
}

/** 切换审批人类型时调整 id（LEADER 无 id，其余给默认值）。 */
export function applyApproverTypeChange(ap: ApproverDraft) {
  if (ap.type === "LEADER") ap.id = undefined;
  else if (!ap.id) ap.id = 1;
}

/** 删除引用了不存在节点 id 的边。 */
export function filterEdges(nodes: GraphNodeDraft[], edges: GraphEdgeDraft[]): GraphEdgeDraft[] {
  const validIds = new Set(nodes.map((n) => n.id));
  return edges.filter((e) => validIds.has(e.from) && validIds.has(e.to));
}

export function isExclusiveEdge(nodes: GraphNodeDraft[], edge: GraphEdgeDraft): boolean {
  const fromNode = nodes.find((n) => n.id === edge.from);
  return fromNode?.type === "EXCLUSIVE_GATEWAY";
}

/** 边的起点变更后维护 condition：条件分支出边默认 default，其余清空。 */
export function applyEdgeFromChange(nodes: GraphNodeDraft[], edge: GraphEdgeDraft) {
  if (isExclusiveEdge(nodes, edge) && !edge.condition) {
    edge.condition = "default";
  } else if (!isExclusiveEdge(nodes, edge)) {
    edge.condition = undefined;
  }
}

/** 为缺少坐标的节点分配默认位置（环形布局）。已有坐标的节点不动。 */
export function ensureLayout(nodes: GraphNodeDraft[]) {
  const n = nodes.length;
  if (n === 0) return;
  const cx = 400;
  const cy = 160;
  const radius = Math.min(280, 50 + n * 35);
  let unassigned = 0;
  for (let i = 0; i < n; i++) {
    if (nodes[i].x === undefined || nodes[i].y === undefined) {
      unassigned++;
    }
  }
  if (unassigned === 0) return;
  let placed = 0;
  for (let i = 0; i < n; i++) {
    if (nodes[i].x === undefined || nodes[i].y === undefined) {
      const angle = (2 * Math.PI * placed) / unassigned;
      nodes[i].x = Math.round(cx + radius * Math.cos(angle));
      nodes[i].y = Math.round(cy + radius * Math.sin(angle));
      placed++;
    }
  }
}

/** 图节点+边草稿 → 流程图 JSON（version:2 图格式）。 */
export function nodesToConfig(nodes: GraphNodeDraft[], edges: GraphEdgeDraft[]): string {
  const validNodes = nodes.filter((n) => n.id);
  if (!validNodes.length) return "";

  const outNodes = validNodes.map((n) => {
    const node: Record<string, unknown> = { id: n.id, type: n.type };
    if (n.type === "TASK" && n.approvers && n.approvers.length > 0) {
      node.approvers = n.approvers.map((a) =>
        a.type === "LEADER" ? { type: a.type } : { type: a.type, id: a.id ?? 0 }
      );
      node.strategy = n.strategy;
    }
    if (n.type === "SUBPROCESS" && n.subprocessDefinition) {
      node.subprocessDefinition = n.subprocessDefinition;
    }
    return node;
  });

  const validIds = new Set(validNodes.map((n) => n.id));
  const outEdges = edges
    .filter((e) => validIds.has(e.from) && validIds.has(e.to))
    .map((e) => {
      const edge: Record<string, unknown> = { from: e.from, to: e.to };
      if (e.condition) edge.condition = e.condition;
      return edge;
    });

  return JSON.stringify({ version: 2, nodes: outNodes, edges: outEdges });
}

/** 流程图 JSON → 图节点+边草稿。
 *  支持图格式（version:2）和旧数组格式（自动转线性图）。 */
export function configToNodes(json: string): {
  nodes: GraphNodeDraft[];
  edges: GraphEdgeDraft[];
} {
  const nodes: GraphNodeDraft[] = [];
  const edges: GraphEdgeDraft[] = [];
  try {
    const parsed = JSON.parse(json);

    // 图格式 version:2
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed) && parsed.version === 2) {
      if (Array.isArray(parsed.nodes)) {
        for (const n of parsed.nodes) {
          const type = n.type as GraphNodeDraft["type"];
          if (!type || !KNOWN_NODE_TYPES.has(type)) continue;
          const node: GraphNodeDraft = { id: String(n.id), type };
          if (type === "TASK" && Array.isArray(n.approvers) && n.approvers.length > 0) {
            node.approvers = n.approvers
              .filter((a: any) => ["USER", "LEADER", "POSITION"].includes(a.type))
              .map((a: any) => ({ type: a.type as ApproverDraft["type"], id: a.id }));
            node.strategy = n.strategy === "ANY" ? "ANY" : "ALL";
          }
          if (type === "SUBPROCESS" && typeof n.subprocessDefinition === "string") {
            node.subprocessDefinition = n.subprocessDefinition;
          }
          nodes.push(node);
        }
      }
      if (Array.isArray(parsed.edges)) {
        for (const e of parsed.edges) {
          if (!e.from || !e.to) continue;
          const edge: GraphEdgeDraft = { from: String(e.from), to: String(e.to) };
          if (typeof e.condition === "string") edge.condition = e.condition;
          edges.push(edge);
        }
      }
      return { nodes, edges };
    }

    // 旧数组格式 → 自动转线性图
    if (!Array.isArray(parsed)) return { nodes, edges };
    const taskNodes: GraphNodeDraft[] = [];
    for (const node of parsed) {
      const approvers: ApproverDraft[] =
        Array.isArray(node.approvers) && node.approvers.length > 0
          ? node.approvers
              .filter((a: any) => ["USER", "LEADER", "POSITION"].includes(a.type))
              .map((a: any) => ({ type: a.type, id: a.id }))
          : node.approver_type === "USER" && node.approver
            ? [{ type: "USER" as const, id: node.approver }]
            : [];
      if (approvers.length === 0) continue;
      taskNodes.push({
        id: `node_${taskNodes.length}`,
        type: "TASK" as const,
        strategy: node.strategy === "ANY" ? "ANY" : "ALL",
        approvers,
      });
    }
    if (!taskNodes.length) return { nodes, edges };

    // start → task0 → … → taskN → end
    const startNode: GraphNodeDraft = { id: "start", type: "START" };
    const endNode: GraphNodeDraft = { id: "end", type: "END" };
    nodes.push(startNode, ...taskNodes, endNode);
    edges.push({ from: "start", to: taskNodes[0].id });
    for (let i = 0; i < taskNodes.length - 1; i++) {
      edges.push({ from: taskNodes[i].id, to: taskNodes[i + 1].id });
    }
    edges.push({ from: taskNodes[taskNodes.length - 1].id, to: "end" });
  } catch {
    /* 非法 JSON：返回已解析部分 */
  }
  return { nodes, edges };
}

/** 字段草稿 → form_schema JSON。key 为空的行跳过；select 解析逗号选项。 */
export function fieldsToSchema(drafts: FieldDraft[]): string {
  const fields = drafts
    .filter((f) => f.key.trim())
    .map((f) => {
      const item: Record<string, unknown> = {
        key: f.key.trim(),
        label: f.label.trim() || f.key.trim(),
        type: f.type,
        required: f.required,
      };
      if (f.type === "select") {
        item.options = f.options
          .split(/[,，]/)
          .map((o) => o.trim())
          .filter(Boolean);
      }
      return item;
    });
  return fields.length ? JSON.stringify(fields) : "";
}

/** JSON 文本 → 字段草稿。 */
export function schemaToFields(json: string): FieldDraft[] {
  const drafts: FieldDraft[] = [];
  try {
    const parsed = JSON.parse(json);
    const list = Array.isArray(parsed) ? parsed : [parsed];
    for (const f of list) {
      if (!f?.key) continue;
      drafts.push({
        key: String(f.key),
        label: String(f.label ?? f.key),
        type: (["text", "textarea", "number", "date", "select"].includes(f.type)
          ? f.type
          : "text") as FieldDraft["type"],
        required: f.required === true,
        options: Array.isArray(f.options) ? f.options.join(",") : "",
      });
    }
  } catch {
    /* 非法 JSON：返回已解析部分 */
  }
  return drafts;
}
