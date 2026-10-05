<template>
  <div class="graph-canvas-container">
    <svg class="graph-canvas" width="100%" height="300" @mousedown="onCanvasMouseDown">
      <!-- 边 -->
      <g v-for="(edge, ei) in renderedEdges" :key="'se' + ei">
        <line
          :x1="edge.x1"
          :y1="edge.y1"
          :x2="edge.x2"
          :y2="edge.y2"
          :class="['graph-edge-line', edge.hasCondition ? 'conditional' : '']"
          @click="selectEdge(edge.index)"
        />
        <text
          v-if="edge.hasCondition"
          :x="(edge.x1 + edge.x2) / 2"
          :y="(edge.y1 + edge.y2) / 2 - 5"
          class="graph-edge-label"
          @click="selectEdge(edge.index)"
        >
          {{ edge.condition }}
        </text>
      </g>
      <!-- 节点 -->
      <g
        v-for="(node, ni) in nodes"
        :key="'sn' + ni"
        :transform="`translate(${node.x ?? 50}, ${node.y ?? 50})`"
        @mousedown.stop="onNodeDragStart($event, ni)"
        @click="selectNode(ni)"
      >
        <rect
          :width="nodeVisualWidth(node)"
          :height="nodeVisualHeight(node)"
          :rx="node.type === 'START' || node.type === 'END' ? 25 : 6"
          :class="[
            'graph-node-rect',
            `node-type-${node.type.toLowerCase()}`,
            selectedNodeIndex === ni ? 'selected' : '',
          ]"
        />
        <text
          :x="nodeVisualWidth(node) / 2"
          :y="nodeVisualHeight(node) / 2"
          class="graph-node-label"
        >
          {{ nodeVisualLabel(node) }}
        </text>
      </g>
    </svg>
  </div>
</template>

<script lang="ts" setup>
import { computed, onBeforeUnmount, ref } from "vue";

import {
  NODE_VISUAL_SIZE,
  nodeTypeLabel,
  type GraphEdgeDraft,
  type GraphNodeDraft,
} from "./workflow-graph";

const props = defineProps<{
  nodes: GraphNodeDraft[];
  edges: GraphEdgeDraft[];
}>();

const emit = defineEmits<{
  (e: "node-move", ni: number, x: number, y: number): void;
}>();

const selectedNodeIndex = ref<number | null>(null);
const selectedEdgeIndex = ref<number | null>(null);

function nodeVisualWidth(node: GraphNodeDraft): number {
  return NODE_VISUAL_SIZE[node.type]?.w ?? 60;
}

function nodeVisualHeight(node: GraphNodeDraft): number {
  return NODE_VISUAL_SIZE[node.type]?.h ?? 60;
}

function nodeVisualLabel(node: GraphNodeDraft): string {
  return nodeTypeLabel(node.type);
}

const renderedEdges = computed(() => {
  const result: {
    x1: number;
    y1: number;
    x2: number;
    y2: number;
    hasCondition: boolean;
    condition: string;
    index: number;
  }[] = [];
  for (let i = 0; i < props.edges.length; i++) {
    const edge = props.edges[i];
    const fromNode = props.nodes.find((n) => n.id === edge.from);
    const toNode = props.nodes.find((n) => n.id === edge.to);
    if (!fromNode || !toNode) continue;
    result.push({
      x1: fromNode.x ?? 50,
      y1: fromNode.y ?? 50,
      x2: toNode.x ?? 50,
      y2: toNode.y ?? 50,
      hasCondition: typeof edge.condition === "string" && edge.condition.length > 0,
      condition: typeof edge.condition === "string" ? edge.condition : "",
      index: i,
    });
  }
  return result;
});

function selectNode(ni: number) {
  selectedNodeIndex.value = ni;
  selectedEdgeIndex.value = null;
}

function selectEdge(ei: number) {
  selectedEdgeIndex.value = ei;
  selectedNodeIndex.value = null;
}

function onCanvasMouseDown() {
  selectedNodeIndex.value = null;
  selectedEdgeIndex.value = null;
}

let dragInfo: {
  idx: number;
  startMX: number;
  startMY: number;
  startNX: number;
  startNY: number;
} | null = null;

function onNodeDragStart(event: MouseEvent, ni: number) {
  event.preventDefault();
  const node = props.nodes[ni];
  if (!node) return;
  selectedNodeIndex.value = ni;
  dragInfo = {
    idx: ni,
    startMX: event.clientX,
    startMY: event.clientY,
    startNX: node.x ?? 0,
    startNY: node.y ?? 0,
  };
  window.addEventListener("mousemove", onNodeDragMove);
  window.addEventListener("mouseup", onNodeDragEnd);
}

function onNodeDragMove(event: MouseEvent) {
  if (!dragInfo) return;
  const node = props.nodes[dragInfo.idx];
  if (!node) return;
  const nx = dragInfo.startNX + (event.clientX - dragInfo.startMX);
  const ny = dragInfo.startNY + (event.clientY - dragInfo.startMY);
  emit("node-move", dragInfo.idx, Math.max(0, Math.min(750, nx)), Math.max(0, Math.min(270, ny)));
}

function onNodeDragEnd() {
  dragInfo = null;
  window.removeEventListener("mousemove", onNodeDragMove);
  window.removeEventListener("mouseup", onNodeDragEnd);
}

onBeforeUnmount(() => {
  window.removeEventListener("mousemove", onNodeDragMove);
  window.removeEventListener("mouseup", onNodeDragEnd);
});
</script>

<style lang="scss" scoped>
.graph-canvas-container {
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  background: var(--el-fill-color-lighter);
  overflow: hidden;
  margin-bottom: 12px;
}

.graph-canvas {
  display: block;
  cursor: default;
  user-select: none;
}

.graph-edge-line {
  stroke: var(--el-border-color);
  stroke-width: 2;
  cursor: pointer;
}

.graph-edge-line.conditional {
  stroke: var(--el-color-warning);
  stroke-dasharray: 6 4;
}

.graph-edge-line:hover {
  stroke-width: 3;
}

.graph-edge-label {
  fill: var(--el-color-warning);
  font-size: 10px;
  text-anchor: middle;
  cursor: pointer;
  pointer-events: none;
}

.graph-node-rect {
  stroke-width: 2;
  cursor: grab;
}

.node-type-start {
  fill: #e8f5e9;
  stroke: #4caf50;
}

.node-type-end {
  fill: #ffebee;
  stroke: #f44336;
}

.node-type-task {
  fill: #e3f2fd;
  stroke: #2196f3;
}

.node-type-exclusive_gateway {
  fill: #fff3e0;
  stroke: #ff9800;
}

.node-type-parallel_gateway_fork {
  fill: #f3e5f5;
  stroke: #9c27b0;
}

.node-type-parallel_gateway_join {
  fill: #f3e5f5;
  stroke: #9c27b0;
}

.node-type-subprocess {
  fill: #efebe9;
  stroke: #795548;
}

/* 暗黑模式：浅色底改为深色底，描边提亮，避免刺眼 */
html.dark .node-type-start {
  fill: #1b3a20;
  stroke: #66bb6a;
}

html.dark .node-type-end {
  fill: #3e1a1d;
  stroke: #ef5350;
}

html.dark .node-type-task {
  fill: #14293f;
  stroke: #42a5f5;
}

html.dark .node-type-exclusive_gateway {
  fill: #3a2a10;
  stroke: #ffa726;
}

html.dark .node-type-parallel_gateway_fork,
html.dark .node-type-parallel_gateway_join {
  fill: #2b1735;
  stroke: #ba68c8;
}

html.dark .node-type-subprocess {
  fill: #2b2420;
  stroke: #8d6e63;
}

.graph-node-rect.selected {
  stroke-width: 4;
  filter: drop-shadow(0 0 4px rgba(0, 0, 0, 0.3));
}

.graph-node-label {
  fill: var(--el-text-color-primary);
  font-size: 11px;
  text-anchor: middle;
  dominant-baseline: middle;
  pointer-events: none;
}
</style>
