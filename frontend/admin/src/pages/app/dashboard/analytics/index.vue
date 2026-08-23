<template>
  <div class="analytics-page">
    <!-- Overview Cards -->
    <el-row :gutter="16" class="mb-5">
      <el-col v-for="(item, index) in overviewItems" :key="index" :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="overview-card">
          <div class="overview-header">
            <div class="overview-header__text">
              <div class="title">{{ item.title }}</div>
              <div class="value-row">
                <span class="value">{{ item.value.toLocaleString() }}</span>
              </div>
            </div>
            <div class="overview-header__icon">
              <SvgIcon :icon="item.icon" :size="32" />
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- OA Instance Trend Chart -->
    <el-card shadow="hover" class="mb-5">
      <template #header>
        <div class="card-header-tabs">
          <span class="card-title">{{ $t("pages.dashboard.oaTrend") }}</span>
        </div>
      </template>
      <div class="chart-container chart-container-trend">
        <AnalyticsTrends :data="trendQuery.data.value" />
      </div>
    </el-card>

    <!-- Distribution Cards Grid -->
    <el-row :gutter="16">
      <el-col :xs="24" :sm="24" :md="12">
        <el-card shadow="hover">
          <template #header>
            <span class="card-title">{{ $t("pages.dashboard.instanceStatusDistribution") }}</span>
          </template>
          <div class="chart-container chart-container-small">
            <AnalyticsDistribution
              :data="instanceStatusDistQuery.data.value"
              title-key="pages.dashboard.instanceStatusDistribution"
              enum-key-prefix="enum.workflowInstance.instanceStatus"
            />
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="24" :md="12">
        <el-card shadow="hover">
          <template #header>
            <span class="card-title">{{ $t("pages.dashboard.attendanceDayResultDistribution") }}</span>
          </template>
          <div class="chart-container chart-container-small">
            <AnalyticsDistribution
              :data="attendanceDistQuery.data.value"
              title-key="pages.dashboard.attendanceDayResultDistribution"
              enum-key-prefix="enum.attendance.dayResult"
            />
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script lang="ts" setup>
import SvgIcon from "@/components/SvgIcon/index.vue";
import { $t } from "@/core/i18n";
import {
  useOaAttendanceDayResultDistribution,
  useOaDashboardOverview,
  useOaInstanceStatusDistribution,
  useOaTrend,
} from "@/api/composables/dashboard";
import AnalyticsTrends from "./analytics-trends.vue";
import AnalyticsDistribution from "./analytics-distribution.vue";

// 概览卡：每张卡由描述符声明其数据字段、标题 i18n 键与图标，
// 数值从后端 GetOverview 对应字段读取。字段名与响应类型严格对齐，
// 避免之前按下标硬绑图标/数据的脆弱写法。
type OverviewField =
  | "workflowInstanceCount"
  | "pendingTaskCount"
  | "todayNewInstanceCount"
  | "todayWorkflowActionCount";

interface OverviewDescriptor {
  field: OverviewField;
  titleKey: string;
  icon: string;
}

const overviewDescriptors: OverviewDescriptor[] = [
  {
    field: "workflowInstanceCount",
    titleKey: "pages.dashboard.workflowInstanceCount",
    icon: "svg:color_card",
  },
  {
    field: "pendingTaskCount",
    titleKey: "pages.dashboard.pendingTaskCount",
    icon: "svg:color_cake",
  },
  {
    field: "todayNewInstanceCount",
    titleKey: "pages.dashboard.todayNewInstanceCount",
    icon: "svg:color_download",
  },
  {
    field: "todayWorkflowActionCount",
    titleKey: "pages.dashboard.todayWorkflowActionCount",
    icon: "svg:color_bell",
  },
];

// 数据未就绪（加载/出错）时返回空数组，待数据到达后由 computed 自动填充。
const overviewQuery = useOaDashboardOverview();

const overviewItems = computed(() => {
  const d = overviewQuery.data.value;
  if (!d) {
    return [];
  }
  return overviewDescriptors.map((desc) => ({
    icon: desc.icon,
    title: $t(desc.titleKey),
    value: d[desc.field] ?? 0,
  }));
});

const trendQuery = useOaTrend(7);
const instanceStatusDistQuery = useOaInstanceStatusDistribution();
const attendanceDistQuery = useOaAttendanceDayResultDistribution();
</script>

<style lang="scss" scoped>
.analytics-page {
  padding: 20px;
}

.overview-card {
  border-radius: 12px;
  transition: all 0.3s ease;
  border: 1px solid var(--el-border-color-lighter);

  &:hover {
    border-color: var(--el-color-primary-light-5);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.03);
  }

  // 暗黑模式 hover 阴影
  html.dark & {
    &:hover {
      border-color: var(--el-color-primary-light-3);
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
    }

    .overview-header__icon {
      background: rgba(64, 128, 255, 0.15);
    }
  }

  :deep(.el-card__body) {
    padding: 20px;
    height: 100%;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    gap: 12px;
  }

  .overview-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;

    &__text {
      flex: 1;
      min-width: 0;
    }

    &__icon {
      flex-shrink: 0;
      display: flex;
      align-items: center;
      justify-content: center;
      width: 48px;
      height: 48px;
      border-radius: 12px;
      background: var(--el-color-primary-light-9);
    }
  }

  .title {
    font-size: 14px;
    font-weight: 500;
    color: var(--el-text-color-regular);
    margin-bottom: 8px;
  }

  .value-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .value {
    font-size: 28px;
    font-weight: 700;
    color: var(--el-text-color-primary);
    line-height: 1;
    letter-spacing: -0.5px;
  }
}

.card-header-tabs {
  display: flex;
  align-items: center;
}

.card-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  display: block;
  padding-top: 2px;
}

.chart-container {
  width: 100%;
  height: 100%;
}

.chart-container-trend {
  height: 380px;
}

.chart-container-small {
  height: 300px;
}
</style>
