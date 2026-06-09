<template>
  <div class="page">
    <h1 class="page-title">💰 花費統計</h1>

    <template v-if="loaded">
    <ErrorState v-if="loadFailed" />
    <EmptyState v-else-if="isEmpty" />
    <template v-else>
    <!-- 總覽 -->
    <div class="stats-row">
      <el-card class="stat-card">
        <div class="stat-num">¥{{ overall.total_amount?.toLocaleString() ?? '—' }}</div>
        <div class="stat-label">總花費</div>
      </el-card>
      <el-card class="stat-card">
        <div class="stat-num">{{ overall.total_quantity?.toLocaleString() ?? '—' }}</div>
        <div class="stat-label">總張數</div>
      </el-card>
      <el-card class="stat-card">
        <div class="stat-num">{{ overall.purchase_count?.toLocaleString() ?? '—' }}</div>
        <div class="stat-label">筆數</div>
      </el-card>
    </div>

    <!-- 圓餅圖 -->
    <div class="charts-row">
      <el-card class="chart-card">
        <template #header>
          <div class="chart-header">
            <span>依單曲佔比</span>
            <el-button-group v-if="singleChartData.length > TOP_N" size="small">
              <el-button :type="!singleShowAll ? 'primary' : ''" @click="singleShowAll = false">前5名</el-button>
              <el-button :type="singleShowAll ? 'primary' : ''" @click="singleShowAll = true">全部</el-button>
            </el-button-group>
          </div>
        </template>
        <VChart :option="singleOption" style="height: 260px" autoresize />
      </el-card>
      <el-card class="chart-card">
        <template #header>
          <div class="chart-header">
            <span>依成員佔比</span>
            <el-button-group v-if="memberChartData.length > TOP_N" size="small">
              <el-button :type="!memberShowAll ? 'primary' : ''" @click="memberShowAll = false">前5名</el-button>
              <el-button :type="memberShowAll ? 'primary' : ''" @click="memberShowAll = true">全部</el-button>
            </el-button-group>
          </div>
        </template>
        <VChart :option="memberOption" style="height: 260px" autoresize />
      </el-card>
    </div>

    <!-- 依單曲（樹狀） -->
    <el-card class="section">
      <template #header><span>依單曲</span></template>

      <div v-if="tree.length === 0" class="empty">尚無資料</div>

      <el-collapse v-else accordion>
        <el-collapse-item v-for="s in tree" :key="`${s.single_number}:${s.single_name}`" :name="`${s.single_number}:${s.single_name}`">
          <template #title>
            <div class="tree-title">
              <span class="tree-name">{{ formatSingle(s.single_name) }}</span>
              <span class="tree-meta">¥{{ s.total_amount.toLocaleString() }} &nbsp;/&nbsp; {{ s.total_quantity }}張</span>
            </div>
          </template>

          <!-- 抽次層 -->
          <el-collapse accordion class="inner-collapse">
            <el-collapse-item v-for="r in s.rounds" :key="r.lottery_round" :name="r.lottery_round">
              <template #title>
                <div class="tree-title round-title">
                  <span class="tree-name">{{ formatRound(r.lottery_round) }}</span>
                  <span class="tree-meta">¥{{ r.total_amount.toLocaleString() }} &nbsp;/&nbsp; {{ r.total_quantity }}張</span>
                </div>
              </template>

              <!-- 成員層 -->
              <div v-for="m in r.members" :key="m.member_name" class="member-row">
                <span class="member-name">{{ m.member_name }}</span>
                <span class="member-meta">¥{{ m.total_amount.toLocaleString() }} &nbsp;/&nbsp; {{ m.total_quantity }}張</span>
              </div>
            </el-collapse-item>
          </el-collapse>
        </el-collapse-item>
      </el-collapse>
    </el-card>

    <!-- 依成員 -->
    <el-card class="section">
      <template #header><span>依成員</span></template>
      <div v-if="byMember.length === 0" class="empty">尚無資料</div>
      <el-table v-else :data="byMember" stripe>
        <el-table-column prop="member_name" label="成員" />
        <el-table-column prop="total_quantity" label="張數" width="80" align="right" />
        <el-table-column label="金額" width="130" align="right">
          <template #default="{ row }">¥{{ row.total_amount?.toLocaleString() }}</template>
        </el-table-column>
      </el-table>
    </el-card>
    </template>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { use } from 'echarts/core'
import { PieChart } from 'echarts/charts'
import { TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { getPurchaseOverallStats, getPurchaseTree, getPurchaseStatsByMember } from '../api/index'
import { useDataStore } from '../stores/data'
import EmptyState from '../components/EmptyState.vue'
import ErrorState from '../components/ErrorState.vue'

use([PieChart, TooltipComponent, LegendComponent, CanvasRenderer])

const dataStore  = useDataStore()
const overall    = ref({})
const tree       = ref([])
const byMember   = ref([])
const loaded     = ref(false)
const loadFailed = ref(false)

const TOP_N = 5
const singleShowAll = ref(false)
const memberShowAll = ref(false)

const singleChartData = computed(() =>
  [...tree.value]
    .sort((a, b) => b.total_amount - a.total_amount)
    .map(s => ({ name: formatSingle(s.single_name) || `第${s.single_number}單`, value: s.total_amount }))
)

const memberChartData = computed(() =>
  byMember.value.map(m => ({ name: m.member_name, value: m.total_amount }))
)

function sliceData(data, showAll) {
  if (showAll || data.length <= TOP_N) return data
  const top = data.slice(0, TOP_N)
  const otherValue = data.slice(TOP_N).reduce((sum, d) => sum + d.value, 0)
  return otherValue > 0 ? [...top, { name: '其他', value: otherValue }] : top
}

function makePieOption(data) {
  return {
    tooltip: {
      trigger: 'item',
      formatter: p => `${p.name}<br/>¥${p.value.toLocaleString()} (${p.percent}%)`,
    },
    legend: {
      orient: 'horizontal',
      bottom: 0,
      type: 'scroll',
      textStyle: { fontSize: 11 },
      formatter: name => name.length > 9 ? name.slice(0, 9) + '…' : name,
    },
    series: [{
      type: 'pie',
      radius: ['38%', '65%'],
      center: ['50%', '44%'],
      data,
      label: { show: false },
      labelLine: { show: false },
      emphasis: {
        itemStyle: { shadowBlur: 10, shadowOffsetX: 0, shadowColor: 'rgba(0,0,0,0.4)' },
      },
    }],
  }
}

const singleOption = computed(() => makePieOption(sliceData(singleChartData.value, singleShowAll.value)))
const memberOption = computed(() => makePieOption(sliceData(memberChartData.value, memberShowAll.value)))

const isEmpty = computed(() => loaded.value && tree.value.length === 0)

function formatSingle(name) {
  return (name ?? '')
    .replace(/(\d+)(?:st|nd|rd|th)シングル/, (_, n) => `${n}單`)
    .replace(/(\d+)(?:st|nd|rd|th)アルバム/, (_, n) => `${n}專`)
    .replace(/^アルバム/, '專輯')
}

function formatRound(round) {
  return round ? `${round}抽` : '—'
}

async function load() {
  if (dataStore.hasData === false) {
    loaded.value = true
    return
  }
  try {
    const [o, t, mb] = await Promise.all([
      getPurchaseOverallStats(),
      getPurchaseTree(),
      getPurchaseStatsByMember(),
    ])
    overall.value  = o.data
    tree.value     = t.data ?? []
    byMember.value = mb.data ?? []
  } catch {
    loadFailed.value = true
  } finally {
    loaded.value = true
  }
}

onMounted(load)
</script>

<style scoped>
.stats-row  { display: flex; gap: 16px; margin-bottom: 24px; flex-wrap: wrap; }
.charts-row { display: flex; gap: 16px; margin-bottom: 24px; flex-wrap: wrap; }
.chart-card { flex: 1; min-width: 280px; }
.chart-header { display: flex; justify-content: space-between; align-items: center; }
.stat-card { flex: 1; min-width: 140px; text-align: center; }
.stat-num  { font-size: 26px; font-weight: bold; color: var(--el-color-primary); }
.stat-label { font-size: 13px; color: #888; margin-top: 4px; }
.section { margin-bottom: 24px; }
.empty { color: #999; text-align: center; padding: 32px 0; }

.tree-title {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  padding-right: 12px;
}
.tree-name { font-weight: 500; }
.tree-meta { color: #888; font-size: 13px; white-space: nowrap; }

.round-title .tree-name { font-weight: normal; color: #555; }

.inner-collapse { margin: 0; }

.member-row {
  display: flex;
  justify-content: space-between;
  padding: 6px 16px;
  font-size: 13px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.member-row:last-child { border-bottom: none; }
.member-name { color: #333; }
.member-meta { color: #888; }
</style>
