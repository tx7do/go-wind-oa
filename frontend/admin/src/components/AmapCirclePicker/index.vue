<template>
  <div class="amap-picker">
    <template v-if="state === 'error'">
      <div class="amap-picker__fallback">
        <div class="amap-picker__fallback-notice">{{ errorMsg }}</div>
        <ElFormItem label="纬度">
          <ElInputNumber
            v-model="latInput"
            :precision="6"
            :step="0.000001"
            :controls="false"
            style="width: 100%"
          />
        </ElFormItem>
        <ElFormItem label="经度">
          <ElInputNumber
            v-model="lngInput"
            :precision="6"
            :step="0.000001"
            :controls="false"
            style="width: 100%"
          />
        </ElFormItem>
        <ElFormItem label="半径(米)">
          <ElInputNumber
            v-model="radiusInput"
            :min="1"
            :max="50000"
            :precision="0"
            :step="100"
            :controls="false"
            style="width: 100%"
          />
        </ElFormItem>
      </div>
    </template>
    <template v-else>
      <div ref="containerRef" class="amap-picker__map">
        <div v-if="state === 'loading'" class="amap-picker__status">
          <ElIcon class="is-loading"><Loading /></ElIcon>
          <span>地图加载中…</span>
        </div>
      </div>
      <div v-if="state === 'ready'" class="amap-picker__controls">
        <ElFormItem label="半径(米)">
          <ElInputNumber
            v-model="radiusInput"
            :min="1"
            :max="50000"
            :precision="0"
            :step="100"
            :controls="false"
            style="width: 100%"
          />
        </ElFormItem>
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, onUnmounted, watch } from "vue";
import { ElIcon, ElInputNumber, ElFormItem } from "element-plus";
import { Loading } from "@element-plus/icons-vue";
import { loadAmap } from "@/composables/use-amap";

defineOptions({ name: "AmapCirclePicker" });

const props = defineProps<{
  modelValue: { center: [number, number] | null; radius: number };
}>();
const emit = defineEmits<{
  (e: "update:modelValue", v: { center: [number, number] | null; radius: number }): void;
}>();

type LoadState = "loading" | "ready" | "error";
const state = ref<LoadState>("loading");
const errorMsg = ref("地图服务不可用");
const containerRef = ref<HTMLElement | null>(null);
const radiusInput = ref<number>(props.modelValue.radius || 100);
// 退化模式（地图不可用）下的手填坐标输入。
const latInput = ref<number>(props.modelValue.center ? props.modelValue.center[1] : 0);
const lngInput = ref<number>(props.modelValue.center ? props.modelValue.center[0] : 0);

// 高德实例引用，卸载时清理用。
let AMap: any = null;
let map: any = null;
let circle: any = null;
let editor: any = null;

// 默认地图中心（无初始 center 时）：上海。
const DEFAULT_CENTER: [number, number] = [121.473701, 31.230416];

function emitUpdate(center: [number, number] | null, radius: number) {
  emit("update:modelValue", { center, radius });
}

function createOrUpdateCircle(center: [number, number], radius: number) {
  if (!map || !AMap) return;
  if (!circle) {
    circle = new AMap.Circle({
      center,
      radius,
      strokeColor: "#1791fc",
      strokeWeight: 2,
      strokeOpacity: 0.9,
      fillColor: "#1791fc",
      fillOpacity: 0.2,
      zIndex: 50,
    });
    map.add(circle);
    // CircleEditor 是插件，脚本加载时已通过 &plugin= 一并加载。
    editor = new AMap.CircleEditor(map, circle);
    editor.on("adjust", (e: any) => {
      const r = Number(e.radius);
      radiusInput.value = r;
      emitUpdate(circle.getCenter() ? [circle.getCenter().getLng(), circle.getCenter().getLat()] : null, r);
    });
    editor.on("move", (e: any) => {
      const c = e.lnglat;
      const center: [number, number] = [c.getLng(), c.getLat()];
      emitUpdate(center, radiusInput.value);
    });
    editor.on("end", () => {
      // close 后最终状态；同步一次。
      if (circle) {
        const c = circle.getCenter();
        const r = circle.getRadius();
        if (c) emitUpdate([c.getLng(), c.getLat()], Number(r));
      }
    });
    editor.open();
  } else {
    circle.setCenter(center);
    circle.setRadius(radius);
  }
}

const mapClickHandler = (ev: any) => {
  if (!ev || !ev.lnglat) return;
  const center: [number, number] = [ev.lnglat.getLng(), ev.lnglat.getLat()];
  createOrUpdateCircle(center, radiusInput.value);
  emitUpdate(center, radiusInput.value);
};

// 半径输入框 → 圆体（ready 态反向同步）；退化态仅向外发值。
watch(radiusInput, (v) => {
  const clamped = Math.max(1, Math.min(50000, Number(v) || 1));
  if (state.value === "ready" && circle) {
    circle.setRadius(clamped);
    const c = circle.getCenter();
    emitUpdate(c ? [c.getLng(), c.getLat()] : null, clamped);
  } else if (state.value === "error") {
    emitUpdate(lngInput.value && latInput.value ? [lngInput.value, latInput.value] : null, clamped);
  }
});

// 退化态：手填经纬度 → 向外发 center。
watch([lngInput, latInput], ([lng, lat]) => {
  if (state.value !== "error") return;
  if (lng && lat) {
    emitUpdate([Number(lng), Number(lat)], radiusInput.value);
  } else {
    emitUpdate(null, radiusInput.value);
  }
});

onMounted(async () => {
  if (!containerRef.value) {
    state.value = "error";
    errorMsg.value = "地图容器初始化失败";
    return;
  }
  try {
    AMap = await loadAmap();
  } catch (e: any) {
    state.value = "error";
    errorMsg.value = e && e.message === "AMAP_NOT_CONFIGURED" ? "地图服务未配置（缺少高德 Key/安全码）" : "地图服务不可用";
    return;
  }
  try {
    const initialCenter: [number, number] = props.modelValue.center ?? DEFAULT_CENTER;
    map = new AMap.Map(containerRef.value, {
      zoom: 14,
      center: initialCenter,
      viewMode: "2D",
    });
    map.on("click", mapClickHandler);
    // 若已有初始 center（编辑场景），直接画圆并开编辑器。
    if (props.modelValue.center) {
      createOrUpdateCircle(props.modelValue.center, props.modelValue.radius || 100);
    }
    state.value = "ready";
  } catch (e) {
    state.value = "error";
    errorMsg.value = "地图初始化失败";
  }
});

onUnmounted(() => {
  // 严格遵循高德生命周期要求：解绑事件 → 关编辑器 → 移除覆盖物 → 销毁地图。
  try {
    if (editor) {
      editor.close();
      editor.off("adjust");
      editor.off("move");
      editor.off("end");
      editor = null;
    }
  } catch {
    /* ignore */
  }
  try {
    if (map) {
      if (mapClickHandler) map.off("click", mapClickHandler);
      if (circle) {
        map.remove(circle);
        circle = null;
      }
      map.destroy();
      map = null;
    }
  } catch {
    /* ignore */
  }
  AMap = null;
});
</script>

<style lang="scss" scoped>
.amap-picker {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.amap-picker__map {
  width: 100%;
  height: 320px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
}
.amap-picker__status {
  width: 100%;
  height: 320px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  color: var(--el-text-color-secondary);
}
.amap-picker__status--error {
  color: var(--el-color-danger);
}
.amap-picker__controls {
  display: flex;
  flex-direction: column;
}
.amap-picker__fallback {
  display: flex;
  flex-direction: column;
}
.amap-picker__fallback-notice {
  padding: 8px 12px;
  margin-bottom: 12px;
  border: 1px solid var(--el-color-warning);
  border-radius: 4px;
  color: var(--el-color-warning);
  font-size: 12px;
}
</style>
