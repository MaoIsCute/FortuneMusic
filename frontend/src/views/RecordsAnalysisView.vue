<template>
  <div class="page">
    <h1 class="page-title">📊 個握分析</h1>

    <template v-if="pageLoaded">
    <!-- 錯誤提示 -->
    <ErrorState v-if="loadFailed" />
    <!-- 尚無資料提示 -->
    <EmptyState v-else-if="!hasData" />

    <!-- 全體統計 -->
    <div v-if="hasData && !loadFailed" class="stats-grid">
      <div class="stat-card">
        <div class="stat-label">總應募數</div>
        <div class="stat-value">{{ overall.total_applied }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">總中選數</div>
        <div class="stat-value">{{ overall.total_won }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">總中選率</div>
        <div class="stat-value highlight">{{ overall.win_rate.toFixed(1) }}%</div>
      </div>
    </div>

    <template v-if="hasData && !loadFailed">
    <el-collapse v-model="openContentSections">

      <!-- 應募次數別中選率折線圖 -->
      <el-collapse-item v-if="chartOption.series.length" name="trend">
        <template #title><span class="collapse-title">各次應募中選率比較</span></template>
        <div class="chart-range-btns">
          <button
            :class="['range-btn', { active: isAllSelected }]"
            @click="toggleAllLegend"
          >成員全選</button>
          <span class="range-divider">|</span>
          <button
            v-for="opt in rangeOptions"
            :key="opt.value"
            :class="['range-btn', { active: chartRange === opt.value }]"
            @click="chartRange = opt.value"
          >{{ opt.label }}</button>
        </div>
        <div class="chart-filters">
          <el-select v-model="chartFilterGroup" placeholder="團體（全部）" clearable multiple collapse-tags size="small" style="width:200px" @change="onChartGroupChange">
            <el-option label="乃木坂46" value="nogizaka46">
              <span :style="{ color: GROUP_COLORS.nogizaka46, fontWeight: 500 }">乃木坂46</span>
            </el-option>
            <el-option label="櫻坂46" value="sakurazaka46">
              <span :style="{ color: GROUP_COLORS.sakurazaka46, fontWeight: 500 }">櫻坂46</span>
            </el-option>
            <el-option label="日向坂46" value="hinatazaka46">
              <span :style="{ color: GROUP_COLORS.hinatazaka46, fontWeight: 500 }">日向坂46</span>
            </el-option>
          </el-select>
          <el-select v-model="chartFilterMembers" placeholder="成員（不選 = 全部顯示）" clearable multiple filterable collapse-tags collapse-tags-tooltip size="small" style="width:280px" @change="applyChartFilter">
            <el-option v-for="m in chartMemberOptions" :key="m.name" :label="m.name" :value="m.name">
              <span :style="{ color: GROUP_COLORS[m.group] }">{{ m.name }}</span>
            </el-option>
          </el-select>
        </div>
        <v-chart :option="chartOption" autoresize style="height: 320px;" @legendselectchanged="onLegendChange" />
      </el-collapse-item>

      <!-- 各部中選率長條圖 -->
      <el-collapse-item name="session">
        <template #title><span class="collapse-title">各部中選率</span></template>
        <div class="chart-filters">
          <el-select v-model="barFilterMember" placeholder="選擇成員" clearable size="small">
            <el-option v-for="m in allMembers" :key="m.name" :label="m.name" :value="m.name">
              <span :style="{ color: GROUP_COLORS[m.group] }">{{ m.name }}</span>
            </el-option>
          </el-select>
          <el-select v-model="barFilterRound" placeholder="選擇抽次" clearable size="small">
            <el-option v-for="r in allRounds" :key="r" :label="formatRound(r)" :value="r" />
          </el-select>
        </div>
        <v-chart v-if="sessionChartOption.series?.length" :option="sessionChartOption" autoresize style="height: 300px;" />
        <div v-else class="chart-empty">請選擇篩選條件</div>
      </el-collapse-item>

      <!-- 訂單序號 vs 中選率長條圖 -->
      <el-collapse-item name="sequence">
        <template #title><span class="collapse-title">各筆應募中選率</span></template>
        <div class="chart-filters">
          <el-select v-model="seqFilterMember" placeholder="選擇成員" clearable size="small" @change="fetchSeqChart">
            <el-option v-for="m in allMembers" :key="m.name" :label="m.name" :value="m.name">
              <span :style="{ color: GROUP_COLORS[m.group] }">{{ m.name }}</span>
            </el-option>
          </el-select>
          <el-select v-model="seqFilterSession" placeholder="選擇部數" clearable size="small" @change="fetchSeqChart">
            <el-option v-for="s in allSessions" :key="s" :label="s" :value="s" />
          </el-select>
          <el-select v-model="seqFilterRound" placeholder="選擇抽次" clearable size="small" @change="fetchSeqChart">
            <el-option v-for="r in allRounds" :key="r" :label="formatRound(r)" :value="r" />
          </el-select>
        </div>
        <v-chart v-if="seqChartOption.series?.length" :option="seqChartOption" autoresize style="height: 300px;" />
        <div v-else class="chart-empty">請選擇篩選條件</div>
      </el-collapse-item>

      <!-- 成員手風琴列表 -->
      <el-collapse-item name="members">
        <template #title><span class="collapse-title">成員列表</span></template>
        <div class="member-list-header">
          <el-select
            v-model="filterMembers"
            multiple
            clearable
            collapse-tags
            collapse-tags-tooltip
            placeholder="顯示特定成員（不選 = 全部）"
            size="small"
            class="member-filter-select"
          >
            <el-option v-for="m in allMembers" :key="m.name" :label="m.name" :value="m.name">
              <span :style="{ color: GROUP_COLORS[m.group] }">{{ m.name }}</span>
            </el-option>
          </el-select>
          <button
            :class="['range-btn', { active: showActiveOnly }]"
            @click="showActiveOnly = !showActiveOnly"
          >在籍成員</button>
        </div>

        <div class="member-list">
          <div
            v-for="[group, data] in groupedMembers"
            :key="group"
            class="group-card"
          >
            <!-- 團體標頭 -->
            <div class="group-header" @click="toggleGroupSection(group)">
              <span class="group-name" :style="{ color: GROUP_COLORS[group] }">{{ groupLabel(group) }}</span>
              <span class="member-summary">
                {{ data.totalApplied }} 應 / {{ data.totalWon }} 中
                <span class="rate">{{ calcRate(data.totalWon, data.totalApplied) }}%</span>
              </span>
              <span class="chevron">{{ isGroupExpanded(group) ? '▲' : '▼' }}</span>
            </div>

            <!-- 團體展開內容：成員手風琴 -->
            <div v-if="isGroupExpanded(group)" class="group-body">
              <div
                v-for="[memberName, member] in data.members"
                :key="memberName"
                class="member-card"
              >
                <!-- 成員標頭 -->
                <div class="member-header" @click="toggleMember(memberName)">
                  <span class="member-name">{{ memberName }}</span>
                  <span class="member-summary">
                    {{ member.totalApplied }} 應 / {{ member.totalWon }} 中
                    <span class="rate">{{ calcRate(member.totalWon, member.totalApplied) }}%</span>
                  </span>
                  <span class="chevron">{{ expandedMembers[memberName] ? '▲' : '▼' }}</span>
                </div>

                <!-- 成員展開內容 -->
                <div v-if="expandedMembers[memberName]" class="member-body">

                  <!-- 單曲手風琴（第二層） -->
                  <div
                    v-for="[singleNum, single] in sortedSingles(member.singles)"
                    :key="singleNum"
                    class="single-card"
                  >
                    <!-- 單曲標頭 -->
                    <div class="single-header" @click="toggleSingle(memberName, singleNum)">
                      <span class="single-name">{{ formatSingle(single.singleName) }}</span>
                      <span class="single-summary">
                        {{ single.totalApplied }} 應 / {{ single.totalWon }} 中
                        <span class="rate">{{ calcRate(single.totalWon, single.totalApplied) }}%</span>
                      </span>
                      <span class="chevron">{{ isSingleExpanded(memberName, singleNum) ? '▲' : '▼' }}</span>
                    </div>

                    <!-- 單曲展開：依次數分組（第三層手風琴） -->
                    <div v-if="isSingleExpanded(memberName, singleNum)" class="single-body">
                      <div
                        v-for="[round, roundData] in sortedRounds(single.rounds)"
                        :key="round"
                        class="round-card"
                      >
                        <div class="round-header" @click="toggleRound(memberName, singleNum, round)">
                          <span class="round-label">{{ formatRound(round) }}</span>
                          <span class="round-summary">
                            {{ roundData.totalApplied }} 應 / {{ roundData.totalWon }} 中
                            <span class="rate">{{ calcRate(roundData.totalWon, roundData.totalApplied) }}%</span>
                          </span>
                          <span class="chevron">{{ isRoundExpanded(memberName, singleNum, round) ? '▲' : '▼' }}</span>
                        </div>
                        <table v-if="isRoundExpanded(memberName, singleNum, round)" class="detail-table">
                          <thead>
                            <tr>
                              <th>日期</th>
                              <th>部數</th>
                              <th>應募</th>
                              <th>中選</th>
                              <th>中選率</th>
                            </tr>
                          </thead>
                          <tbody>
                            <tr
                              v-for="row in sortedRows(roundData.rows)"
                              :key="row.event_date + row.session"
                            >
                              <td>{{ row.event_date }}</td>
                              <td>{{ row.session }}</td>
                              <td>{{ row.total_applied }}</td>
                              <td>{{ row.total_won }}</td>
                              <td>
                                <span :class="rateClass(row.win_rate)">
                                  {{ row.win_rate.toFixed(1) }}%
                                </span>
                              </td>
                            </tr>
                          </tbody>
                        </table>
                      </div>
                    </div>
                  </div>

                </div>
              </div>
            </div>
          </div>
        </div>
      </el-collapse-item>

    </el-collapse>
    </template>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { getStats, getDetailStats, getOrderSequenceStats } from '../api/index'
import { useThemeStore } from '../stores/theme'
import { useDataStore } from '../stores/data'
import { detectExtension } from '../utils/extension'
import EmptyState from '../components/EmptyState.vue'
import ErrorState from '../components/ErrorState.vue'
import { getMemberInfo, sortMembersByGroupAndGen, memberOrderIndex } from '../utils/members'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { LineChart, BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

use([LineChart, BarChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

const GROUP_LABELS = { nogizaka46: '乃木坂46', sakurazaka46: '櫻坂46', hinatazaka46: '日向坂46' }
const GROUP_COLORS = { nogizaka46: '#9333ea', sakurazaka46: '#ec4899', hinatazaka46: '#0ea5e9' }
function groupLabel(g) {
  return GROUP_LABELS[g] || g || '—'
}

const router = useRouter()
const themeStore = useThemeStore()
const dataStore  = useDataStore()
const ct = computed(() => themeStore.isDark
  ? { text: '#d4d8e3', sub: '#9aa3b5', line: '#3a3f5c' }
  : { text: '#555',    sub: '#888',    line: '#e8e8e8' }
)

const overall     = ref({ total_applied: 0, total_won: 0, win_rate: 0 })
const rows        = ref([])
const hasData     = ref(false)
const pageLoaded  = ref(false)
const loadFailed  = ref(false)
const openContentSections = ref(['trend', 'session', 'sequence', 'members'])
const expandedMembers = ref({})
const expandedSingles = ref({})
const expandedRounds  = ref({})

onMounted(async () => {
  try {
    if (dataStore.hasData === false) {
      const installed = await detectExtension()
      if (!installed) { router.replace('/setup'); return }
      pageLoaded.value = true
      return
    }
    const [s, d] = await Promise.all([getStats(), getDetailStats()])
    overall.value = s.data
    rows.value    = d.data ?? []
    if (overall.value.total_applied === 0) {
      dataStore.hasData = false
      const installed = await detectExtension()
      if (!installed) { router.replace('/setup'); return }
    } else {
      dataStore.hasData = true
      hasData.value = true
    }
  } catch {
    loadFailed.value = true
  }
  pageLoaded.value = true
})

// flat rows → member → singleKey → round → rows
// singleKey: numbered singles use single_number (e.g. "41"),
//            albums (single_number=0) use "album::<single_name>" to avoid merging
const memberMap = computed(() => {
  const map = {}
  for (const row of rows.value) {
    if (!map[row.member_name]) {
      map[row.member_name] = {
        singles: {}, totalApplied: 0, totalWon: 0,
        group: getMemberInfo(row.member_name).group || row.group || '',
      }
    }
    const m = map[row.member_name]
    m.totalApplied += row.total_applied
    m.totalWon     += row.total_won

    const singleKey = row.single_number > 0
      ? String(row.single_number)
      : `album::${row.single_name}`

    if (!m.singles[singleKey]) {
      m.singles[singleKey] = {
        singleName:   row.single_name,
        singleNumber: row.single_number,
        minEventDate: row.event_date,
        rounds:       {},
        totalApplied: 0,
        totalWon:     0,
      }
    } else {
      // 取最新的 single_name（title 可能從「未定」更新）
      m.singles[singleKey].singleName = row.single_name
      if (row.event_date < m.singles[singleKey].minEventDate) {
        m.singles[singleKey].minEventDate = row.event_date
      }
    }
    const s = m.singles[singleKey]
    s.totalApplied += row.total_applied
    s.totalWon     += row.total_won

    const roundKey = row.lottery_round || '—'
    if (!s.rounds[roundKey]) s.rounds[roundKey] = { rows: [], totalApplied: 0, totalWon: 0 }
    s.rounds[roundKey].rows.push(row)
    s.rounds[roundKey].totalApplied += row.total_applied
    s.rounds[roundKey].totalWon     += row.total_won
  }
  return map
})

const showActiveOnly = ref(false)
const filterMembers  = ref([])

// 成員依期別 → 五十音排序，可過濾只顯示在籍 / 指定成員
const sortedMembers = computed(() =>
  Object.entries(memberMap.value)
    .filter(([name]) => {
      if (showActiveOnly.value && !(getMemberInfo(name).active ?? true)) return false
      if (filterMembers.value.length && !filterMembers.value.includes(name)) return false
      return true
    })
    .sort(([a], [b]) => {
      const ga = getMemberInfo(a).gen ?? 99
      const gb = getMemberInfo(b).gen ?? 99
      if (ga !== gb) return ga - gb
      return memberOrderIndex(a) - memberOrderIndex(b)
    })
)

// 團體 → （團體內沿用 sortedMembers 已經排好的期別→五十音順序）
const GROUP_ORDER = { nogizaka46: 0, sakurazaka46: 1, hinatazaka46: 2 }
const groupedMembers = computed(() => {
  const buckets = {}
  for (const entry of sortedMembers.value) {
    const g = entry[1].group || ''
    if (!buckets[g]) buckets[g] = { totalApplied: 0, totalWon: 0, members: [] }
    buckets[g].members.push(entry)
    buckets[g].totalApplied += entry[1].totalApplied
    buckets[g].totalWon     += entry[1].totalWon
  }
  return Object.entries(buckets).sort(([a], [b]) => (GROUP_ORDER[a] ?? 9) - (GROUP_ORDER[b] ?? 9))
})

const expandedGroups = ref({})
function isGroupExpanded(group) {
  return expandedGroups.value[group] !== false
}
function toggleGroupSection(group) {
  expandedGroups.value[group] = !isGroupExpanded(group)
}

// 單曲依最早 event_date 排序（新的在前），讓專輯與單曲按時間軸交錯
function sortedSingles(singles) {
  return Object.entries(singles).sort(([, a], [, b]) =>
    parseDate(b.minEventDate) - parseDate(a.minEventDate)
  )
}

// 次數依數字排序
function sortedRounds(rounds) {
  return Object.entries(rounds).sort(([a], [b]) => {
    const na = parseInt(a.match(/\d+/)?.[0] ?? 0)
    const nb = parseInt(b.match(/\d+/)?.[0] ?? 0)
    return na - nb
  })
}

// 行依日期 → 部數排序
function sortedRows(rowList) {
  return [...rowList].sort((a, b) => {
    const da = parseDate(a.event_date)
    const db = parseDate(b.event_date)
    if (da - db !== 0) return da - db
    return a.session.localeCompare(b.session, 'ja')
  })
}

function parseDate(str) {
  const p = str.split('/')
  if (p.length === 3) return new Date(p[0], p[1] - 1, p[2])
  if (p.length === 2) return new Date(2000, p[0] - 1, p[1])
  return new Date(0)
}

function toggleMember(name) {
  expandedMembers.value[name] = !expandedMembers.value[name]
}

function toggleSingle(memberName, singleName) {
  const key = `${memberName}::${singleName}`
  expandedSingles.value[key] = !expandedSingles.value[key]
}

function isSingleExpanded(memberName, singleName) {
  return !!expandedSingles.value[`${memberName}::${singleName}`]
}

function toggleRound(memberName, singleName, round) {
  const key = `${memberName}::${singleName}::${round}`
  expandedRounds.value[key] = !expandedRounds.value[key]
}

function isRoundExpanded(memberName, singleName, round) {
  return !!expandedRounds.value[`${memberName}::${singleName}::${round}`]
}

function formatRound(round) {
  return round ? `${round}抽` : ''
}

function calcRate(won, applied) {
  if (!applied) return '0.0'
  return (won / applied * 100).toFixed(1)
}

function rateClass(rate) {
  if (rate >= 80) return 'rate high'
  if (rate >= 40) return 'rate mid'
  return 'rate low'
}

// "41stシングル「最後に…」" → "41單「最後に…」"
// "5thアルバム「My respect」" → "5專「My respect」"
// "アルバム「My respect」" → "專輯「My respect」"
function formatSingle(singleName) {
  return singleName
    .replace(/(\d+)(?:st|nd|rd|th)シングル/, (_, n) => `${n}單`)
    .replace(/(\d+)(?:st|nd|rd|th)アルバム/, (_, n) => `${n}專`)
    .replace(/^アルバム/, '專輯')
}

// ── 訂單序號圖篩選 ───────────────────────────────────────
const seqFilterMember  = ref('')
const seqFilterSession = ref('')
const seqFilterRound   = ref('')
const seqData          = ref([])

const allSessions = computed(() => {
  const set = new Set()
  for (const row of rows.value) set.add(row.session)
  return [...set].sort((a, b) =>
    parseInt(a.match(/\d+/)?.[0] ?? 0) - parseInt(b.match(/\d+/)?.[0] ?? 0)
  )
})

async function fetchSeqChart() {
  if (!seqFilterMember.value && !seqFilterSession.value && !seqFilterRound.value) {
    seqData.value = []
    return
  }
  const params = {}
  if (seqFilterMember.value)  params.member  = seqFilterMember.value
  if (seqFilterSession.value) params.session = seqFilterSession.value
  if (seqFilterRound.value)   params.round   = seqFilterRound.value
  const res = await getOrderSequenceStats(params)
  seqData.value = res.data ?? []
}

const seqChartOption = computed(() => {
  if (!seqData.value.length) return {}
  const labels = seqData.value.map(d => d.position)
  const data   = seqData.value.map(d => ({
    value:   d.win_rate,
    applied: d.applied,
    won:     d.won,
  }))
  const c = ct.value
  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      formatter(params) {
        const d = data[params[0].dataIndex]
        return `${params[0].name}<br/>中選率：${params[0].value}%<br/>應募：${d.applied}　中選：${d.won}`
      },
    },
    grid: { top: 16, right: 24, bottom: 40, left: 54 },
    xAxis: { type: 'category', data: labels, axisLabel: { color: c.text }, axisLine: { lineStyle: { color: c.line } } },
    yAxis: {
      type: 'value', min: 0, max: 100,
      axisLabel: { formatter: '{value}%', color: c.text },
      splitLine: { lineStyle: { type: 'dashed', color: c.line } },
    },
    series: [{
      type: 'bar',
      data: data.map(d => ({
        value: d.value,
        itemStyle: { color: d.value >= 80 ? '#52c41a' : d.value >= 40 ? '#faad14' : '#ff4d4f' },
      })),
      label: { show: true, position: 'top', formatter: '{c}%', fontSize: 12, color: c.text },
    }],
  }
})

// ── 各部長條圖篩選 ───────────────────────────────────────
const barFilterMember = ref('')
const barFilterRound  = ref('')

const allMembers = computed(() => {
  const nameGroupMap = new Map()
  rows.value.forEach(r => nameGroupMap.set(r.member_name, r.group || ''))
  return sortMembersByGroupAndGen([...nameGroupMap.entries()].map(([name, group]) => ({ name, group })))
})

const allRounds = computed(() => {
  const set = new Set()
  for (const row of rows.value) {
    if (row.lottery_round) set.add(row.lottery_round)
  }
  return [...set].sort((a, b) => a - b)
})

const sessionChartOption = computed(() => {
  const filtered = rows.value.filter(row => {
    if (barFilterMember.value && row.member_name !== barFilterMember.value) return false
    if (barFilterRound.value && row.lottery_round !== barFilterRound.value) return false
    return true
  })

  if (filtered.length === 0) return {}

  const agg = {}
  for (const row of filtered) {
    if (!agg[row.session]) agg[row.session] = { applied: 0, won: 0 }
    agg[row.session].applied += row.total_applied
    agg[row.session].won     += row.total_won
  }

  const sessions = Object.keys(agg).sort((a, b) =>
    parseInt(a.match(/\d+/)?.[0] ?? 0) - parseInt(b.match(/\d+/)?.[0] ?? 0)
  )

  const data = sessions.map(s => {
    const d = agg[s]
    const rate = d.applied ? parseFloat((d.won / d.applied * 100).toFixed(1)) : 0
    return { value: rate, applied: d.applied, won: d.won }
  })

  const c = ct.value
  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      formatter(params) {
        const d = data[params[0].dataIndex]
        return `${params[0].name}<br/>中選率：${params[0].value}%<br/>應募：${d.applied}　中選：${d.won}`
      },
    },
    grid: { top: 16, right: 24, bottom: 40, left: 54 },
    xAxis: { type: 'category', data: sessions, axisLabel: { color: c.text }, axisLine: { lineStyle: { color: c.line } } },
    yAxis: {
      type: 'value', min: 0, max: 100,
      axisLabel: { formatter: '{value}%', color: c.text },
      splitLine: { lineStyle: { type: 'dashed', color: c.line } },
    },
    series: [{
      type: 'bar',
      data: data.map(d => ({
        value: d.value,
        itemStyle: { color: d.value >= 80 ? '#52c41a' : d.value >= 40 ? '#faad14' : '#ff4d4f' },
      })),
      label: { show: true, position: 'top', formatter: '{c}%', fontSize: 12, color: c.text },
    }],
  }
})

// ── 折線圖 ──────────────────────────────────────────────
const CHART_COLORS = ['#5470c6', '#91cc75', '#fac858', '#ee6666', '#73c0de', '#3ba272', '#fc8452', '#9a60b4', '#ea7ccc']

const rangeOptions = [
  { label: '前3抽', value: 3 },
  { label: '前6抽', value: 6 },
  { label: '全部',  value: 0 },
]
const chartRange    = ref(0)
const legendSelected = ref({})

// 團體/成員下拉：成員數一多，直接在圖表 legend 裡點選很難快速找到人，
// 改用團體先縮小範圍、成員下拉多選（可搜尋）來決定折線圖要顯示誰
const chartFilterGroup   = ref([])
const chartFilterMembers = ref([])

const chartMemberOptions = computed(() =>
  chartFilterGroup.value.length
    ? allMembers.value.filter(m => chartFilterGroup.value.includes(m.group))
    : allMembers.value
)

function onChartGroupChange() {
  const allowed = new Set(chartMemberOptions.value.map(m => m.name))
  chartFilterMembers.value = chartFilterMembers.value.filter(n => allowed.has(n))
  applyChartFilter()
}

// 成員下拉沒選任何人 = 顯示全部（維持原本預設行為），選了才只顯示被選中的成員
function applyChartFilter() {
  const sel = {}
  const hasFilter = chartFilterMembers.value.length > 0
  for (const name of Object.keys(legendSelected.value)) {
    sel[name] = name === '全部' || !hasFilter || chartFilterMembers.value.includes(name)
  }
  legendSelected.value = sel
}

// rows 載入後初始化所有 legend 為選取狀態
watch(memberMap, (map) => {
  const hasFilter = chartFilterMembers.value.length > 0
  const sel = {}
  for (const name of Object.keys(map)) sel[name] = !hasFilter || chartFilterMembers.value.includes(name)
  sel['全部'] = true
  legendSelected.value = sel
}, { immediate: true })

function onLegendChange(e) {
  legendSelected.value = { ...e.selected }
}

const isAllSelected = computed(() =>
  Object.values(legendSelected.value).every(v => v)
)

function toggleAllLegend() {
  const next = !isAllSelected.value
  const sel = {}
  for (const k of Object.keys(legendSelected.value)) sel[k] = next
  legendSelected.value = sel
}

const chartOption = computed(() => {
  // 收集所有 round，排序
  const roundSet = new Set()
  for (const row of rows.value) {
    if (row.lottery_round) roundSet.add(row.lottery_round)
  }
  const rounds = [...roundSet].sort((a, b) => a - b)

  if (rounds.length === 0) return { series: [] }

  // 依範圍裁切
  const visibleRounds = chartRange.value > 0 ? rounds.slice(0, chartRange.value) : rounds

  // 彙整 (member, round) → { applied, won }
  const agg = {}
  const totalByRound = {}
  for (const row of rows.value) {
    const round = row.lottery_round
    if (!round) continue
    if (!agg[row.member_name]) agg[row.member_name] = {}
    if (!agg[row.member_name][round]) agg[row.member_name][round] = { applied: 0, won: 0 }
    agg[row.member_name][round].applied += row.total_applied
    agg[row.member_name][round].won     += row.total_won
    if (!totalByRound[round]) totalByRound[round] = { applied: 0, won: 0 }
    totalByRound[round].applied += row.total_applied
    totalByRound[round].won     += row.total_won
  }

  const members = Object.keys(agg).sort((a, b) => memberOrderIndex(a) - memberOrderIndex(b))
  const xLabels = visibleRounds.map(r => formatRound(r))

  const winRate = (d) => d && d.applied ? parseFloat((d.won / d.applied * 100).toFixed(1)) : null

  const series = [
    ...members.map((member, i) => ({
      name: member,
      type: 'line',
      smooth: true,
      connectNulls: false,
      color: CHART_COLORS[i % CHART_COLORS.length],
      symbol: 'circle',
      symbolSize: 7,
      data: visibleRounds.map(r => winRate(agg[member][r])),
    })),
    {
      name: '全部',
      type: 'line',
      smooth: true,
      lineStyle: { width: 3, type: 'dashed' },
      color: '#333',
      symbol: 'diamond',
      symbolSize: 8,
      data: visibleRounds.map(r => winRate(totalByRound[r])),
    },
  ]

  const c = ct.value
  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      formatter(params) {
        const idx = params[0].dataIndex
        let html = `<b>${xLabels[idx]}</b><br/>`
        params.forEach(p => {
          if (p.value !== null && p.value !== undefined)
            html += `${p.marker}${p.seriesName}：${p.value}%<br/>`
        })
        return html
      },
    },
    legend: { data: [...members, '全部'], bottom: 0, type: 'scroll', selected: legendSelected.value, textStyle: { color: c.text } },
    grid: { top: 16, right: 24, bottom: 56, left: 54 },
    xAxis: { type: 'category', data: xLabels, axisLabel: { color: c.text }, axisLine: { lineStyle: { color: c.line } } },
    yAxis: {
      type: 'value', min: 0, max: 100,
      axisLabel: { formatter: '{value}%', color: c.text },
      splitLine: { lineStyle: { type: 'dashed', color: c.line } },
    },
    series,
  }
})
</script>

<style scoped>
.empty-card {
  background: white;
  border-radius: 16px;
  padding: 48px 32px;
  text-align: center;
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
  margin-bottom: 32px;
}
.empty-icon { font-size: 48px; margin-bottom: 16px; }
.empty-title { font-size: 20px; font-weight: bold; color: #333; margin: 0 0 12px; }
.empty-sub { color: #888; line-height: 1.7; margin: 0; }
html.dark .empty-card  { background: #1e2030; }
html.dark .empty-title { color: #e8eaf0; }
html.dark .empty-sub   { color: #9aa3b5; }

.stats-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 32px;
}
.stat-card {
  background: white;
  border-radius: 12px;
  padding: 24px;
  text-align: center;
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
}
.stat-label { color: #888; font-size: 14px; margin-bottom: 8px; }
.stat-value { font-size: 32px; font-weight: bold; }
.stat-value.highlight { color: var(--color-primary); }

:deep(.el-collapse) { border: none; background: transparent; }
:deep(.el-collapse-item) {
  margin-bottom: 12px;
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid #e5e7eb;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.07);
  background: white;
}
:deep(.el-collapse-item__header) {
  height: 52px;
  padding: 0 20px;
  font-size: 14px;
  font-weight: 600;
  color: #111827;
  background: white;
  border-bottom: 1px solid transparent;
}
:deep(.el-collapse-item.is-active .el-collapse-item__header) { border-bottom-color: #e5e7eb; }
:deep(.el-collapse-item__arrow) { color: #6b7280; }
:deep(.el-collapse-item__wrap) { background: white; border: none; }
:deep(.el-collapse-item__content) { padding: 16px 20px 20px; }
.collapse-title { font-weight: 600; font-size: 14px; }

/* 圖表共用 */
.chart-range-btns {
  display: flex;
  gap: 6px;
  margin-bottom: 14px;
}
.range-btn {
  padding: 3px 12px;
  border: 1px solid #ddd;
  border-radius: 20px;
  background: white;
  font-size: 13px;
  cursor: pointer;
  color: #666;
  transition: all 0.15s;
}
.range-btn:hover { border-color: var(--color-primary); color: var(--color-primary); }
.range-btn.active { background: var(--color-primary); border-color: var(--color-primary); color: white; }
.range-divider { color: #ddd; align-self: center; }
.chart-filters {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
}
.chart-empty {
  height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #bbb;
  font-size: 14px;
}

/* 成員層 */
.member-list-header {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  margin-bottom: 8px;
}
.member-filter-select {
  width: 260px;
}
.member-list { display: flex; flex-direction: column; gap: 12px; }

/* 團體層 */
.group-card {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
  overflow: hidden;
}
.group-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  cursor: pointer;
  user-select: none;
  transition: background 0.15s;
}
.group-header:hover { background: #f5f5f5; }
.group-name { font-size: 18px; font-weight: bold; flex: 1; }
.group-body { padding: 0 16px 16px; display: flex; flex-direction: column; gap: 12px; }

.member-card {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
  overflow: hidden;
}

.member-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  cursor: pointer;
  user-select: none;
  transition: background 0.15s;
}
.member-header:hover { background: #f5f5f5; }
.member-name { font-size: 18px; font-weight: bold; flex: 1; }
.member-summary { color: #666; font-size: 14px; }
.member-summary .rate,
.single-summary .rate { color: var(--color-primary); font-weight: bold; margin-left: 6px; }
.chevron { color: #bbb; font-size: 11px; }

.member-body { padding: 0 16px 16px; display: flex; flex-direction: column; gap: 8px; }

/* 單曲層 */
.single-card {
  border: 1px solid #eee;
  border-radius: 8px;
  overflow: hidden;
}

.single-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  cursor: pointer;
  user-select: none;
  background: #fafafa;
  transition: background 0.15s;
}
.single-header:hover { background: #f0f0f0; }
.single-name { font-size: 15px; font-weight: 600; color: var(--color-primary); flex: 1; }
.single-summary { color: #888; font-size: 13px; }

.single-body { padding: 0 16px 16px; }

/* 次數層（手風琴） */
.round-card {
  border: 1px solid #e8e8e8;
  border-radius: 6px;
  overflow: hidden;
  margin-top: 8px;
}

.round-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 14px;
  cursor: pointer;
  user-select: none;
  background: #f5f5f5;
  border-left: 3px solid var(--color-primary);
  transition: background 0.15s;
}
.round-header:hover { background: #ececec; }

.round-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-primary);
  flex: 1;
}

.round-summary {
  color: #888;
  font-size: 12px;
}
.round-summary .rate { color: var(--color-primary); font-weight: bold; margin-left: 4px; }

/* 表格 */
.detail-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.detail-table th {
  background: #f7f7f7;
  padding: 7px 12px;
  text-align: left;
  color: #888;
  font-weight: 500;
}
.detail-table td {
  padding: 7px 12px;
  border-bottom: 1px solid #f0f0f0;
}
.detail-table tr:last-child td { border-bottom: none; }

.rate { font-weight: bold; }
.rate.high { color: #52c41a; }
.rate.mid  { color: #faad14; }
.rate.low  { color: #ff4d4f; }

/* ── 深色模式 ── */
html.dark .stat-card  { background: #1e2030; box-shadow: 0 2px 12px rgba(0,0,0,0.4); }
html.dark .stat-label { color: #9aa3b5; }
html.dark .stat-value { color: #e8eaf0; }

html.dark .chart-empty { color: #6b7490; }

html.dark .range-btn         { background: #252840; border-color: #3a3f5c; color: #b8bfcc; }
html.dark .range-btn:hover   { border-color: var(--color-primary); color: var(--color-primary); }
html.dark .range-btn.active  { background: var(--color-primary); border-color: var(--color-primary); color: white; }
html.dark .range-divider     { color: #3a3f5c; }

html.dark .group-card            { background: #1e2030; box-shadow: 0 2px 12px rgba(0,0,0,0.4); }
html.dark .group-header:hover    { background: #252840; }

html.dark .member-card           { background: #1e2030; box-shadow: 0 2px 12px rgba(0,0,0,0.4); }
html.dark .member-header:hover   { background: #252840; }
html.dark .member-name           { color: #e8eaf0; }
html.dark .member-summary        { color: #9aa3b5; }
html.dark .chevron               { color: #4a5270; }

html.dark .single-card           { border-color: #2e3450; }
html.dark .single-header         { background: #252840; }
html.dark .single-header:hover   { background: #2c3154; }
html.dark .single-summary        { color: #9aa3b5; }

html.dark .round-card            { border-color: #2e3450; }
html.dark .round-header          { background: #1a1f3a; }
html.dark .round-header:hover    { background: #20264a; }
html.dark .round-summary         { color: #9aa3b5; }

html.dark .detail-table th       { background: #252840; color: #9aa3b5; }
html.dark .detail-table td       { border-bottom-color: #2e3450; color: #d4d8e3; }
</style>
