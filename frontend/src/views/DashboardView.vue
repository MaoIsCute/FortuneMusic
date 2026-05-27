<template>
  <div class="page">
    <h1 class="page-title">📊 總覽</h1>

    <!-- 全體統計 -->
    <div class="stats-grid">
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

    <!-- 應募次數別中選率折線圖 -->
    <div v-if="chartOption.series.length" class="chart-card">
      <div class="chart-header">
        <div class="chart-title">各次應募中選率比較</div>
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
      </div>
      <v-chart :option="chartOption" autoresize style="height: 320px;" @legendselectchanged="onLegendChange" />
    </div>

    <!-- 各部中選率長條圖 -->
    <div class="chart-card">
      <div class="chart-title">各部中選率</div>
      <div class="chart-filters">
        <el-select v-model="barFilterMember" placeholder="選擇成員" clearable size="small">
          <el-option v-for="m in allMembers" :key="m" :label="m" :value="m" />
        </el-select>
        <el-select v-model="barFilterRound" placeholder="選擇抽次" clearable size="small">
          <el-option v-for="r in allRounds" :key="r" :label="formatRound(r)" :value="r" />
        </el-select>
      </div>
      <v-chart v-if="sessionChartOption.series?.length" :option="sessionChartOption" autoresize style="height: 300px;" />
      <div v-else class="chart-empty">請選擇篩選條件</div>
    </div>

    <!-- 訂單序號 vs 中選率長條圖 -->
    <div class="chart-card">
      <div class="chart-title">各張應募中選率</div>
      <div class="chart-filters">
        <el-select v-model="seqFilterMember" placeholder="選擇成員" clearable size="small" @change="fetchSeqChart">
          <el-option v-for="m in allMembers" :key="m" :label="m" :value="m" />
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
    </div>

    <!-- 成員列表控制列 -->
    <div class="member-list-header">
      <button
        :class="['range-btn', { active: showActiveOnly }]"
        @click="showActiveOnly = !showActiveOnly"
      >在籍成員</button>
    </div>

    <!-- 成員手風琴（第一層） -->
    <div class="member-list">
      <div
        v-for="[memberName, member] in sortedMembers"
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
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { getStats, getDetailStats, getOrderSequenceStats } from '../api/index'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { LineChart, BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

use([LineChart, BarChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

const MEMBERS = {
  // 1期
  '秋元真夏':             { gen: 1, active: false },
  '安藤美雲':             { gen: 1, active: false },
  '生田絵梨花':           { gen: 1, active: false },
  '生駒里奈':             { gen: 1, active: false },
  '市來玲奈':             { gen: 1, active: false },
  '伊藤かりん':           { gen: 1, active: false },
  '伊藤寧寧':             { gen: 1, active: false },
  '伊藤萬理華':           { gen: 1, active: false },
  '岩瀬佑美子':           { gen: 1, active: false },
  '衛藤美彩':             { gen: 1, active: false },
  '大和里菜':             { gen: 1, active: false },
  '川後陽菜':             { gen: 1, active: false },
  '川村真洋':             { gen: 1, active: false },
  '柏幸奈':               { gen: 1, active: false },
  '斉藤優里':             { gen: 1, active: false },
  '齋藤飛鳥':             { gen: 1, active: false },
  '桜井玲香':             { gen: 1, active: false },
  '相楽伊織':             { gen: 1, active: false },
  '白石麻衣':             { gen: 1, active: false },
  '高山一実':             { gen: 1, active: false },
  '中元日芽香':           { gen: 1, active: false },
  '能條愛未':             { gen: 1, active: false },
  '永島聖羅':             { gen: 1, active: false },
  '西野七瀬':             { gen: 1, active: false },
  '橋本奈々未':           { gen: 1, active: false },
  '畠中清羅':             { gen: 1, active: false },
  '樋口日奈':             { gen: 1, active: false },
  '深川麻衣':             { gen: 1, active: false },
  '星野みなみ':           { gen: 1, active: false },
  '松井玲奈':             { gen: 1, active: false },
  '松村沙友理':           { gen: 1, active: false },
  '宮澤成良':             { gen: 1, active: false },
  '山本穂乃香':           { gen: 1, active: false },
  '若月佑美':             { gen: 1, active: false },
  '和田まあや':           { gen: 1, active: false },
  '吉本彩華':             { gen: 1, active: false },
  // 2期
  '井上小百合':           { gen: 2, active: false },
  '伊藤純奈':             { gen: 2, active: false },
  '斎藤ちはる':           { gen: 2, active: false },
  '佐々木琴子':           { gen: 2, active: false },
  '新内眞衣':             { gen: 2, active: false },
  '鈴木絢音':             { gen: 2, active: false },
  '寺田蘭世':             { gen: 2, active: false },
  '中田花奈':             { gen: 2, active: false },
  '堀未央奈':             { gen: 2, active: false },
  '北野日奈子':           { gen: 2, active: false },
  '渡辺みり愛':           { gen: 2, active: false },
  // 3期
  '伊藤理々杏':           { gen: 3, active: true },
  '岩本蓮加':             { gen: 3, active: true },
  '吉田綾乃クリスティー': { gen: 3, active: true },
  // 4期
  '遠藤さくら':           { gen: 4, active: true },
  '賀喜遥香':             { gen: 4, active: true },
  '金川紗耶':             { gen: 4, active: true },
  '黒見明香':             { gen: 4, active: true },
  '柴田柚菜':             { gen: 4, active: true },
  '田村真佑':             { gen: 4, active: true },
  '筒井あやめ':           { gen: 4, active: true },
  '林瑠奈':               { gen: 4, active: true },
  '弓木奈於':             { gen: 4, active: true },
  // 5期
  '五百城茉央':           { gen: 5, active: true },
  '池田瑛紗':             { gen: 5, active: true },
  '一ノ瀬美空':           { gen: 5, active: true },
  '井上和':               { gen: 5, active: true },
  '岡本姫奈':             { gen: 5, active: true },
  '小川彩':               { gen: 5, active: true },
  '奥田いろは':           { gen: 5, active: true },
  '川﨑桜':               { gen: 5, active: true },
  '菅原咲月':             { gen: 5, active: true },
  '冨里奈央':             { gen: 5, active: true },
  '中西アルノ':           { gen: 5, active: true },
  // 3期（卒業）
  '梅澤美波':             { gen: 3, active: false },
  '大園桃子':             { gen: 3, active: false },
  '久保史緒里':           { gen: 3, active: false },
  '阪口珠美':             { gen: 3, active: false },
  '佐藤楓':               { gen: 3, active: false },
  '中村麗乃':             { gen: 3, active: false },
  '向井葉月':             { gen: 3, active: false },
  '山崎怜奈':             { gen: 3, active: false },
  '山下美月':             { gen: 3, active: false },
  '与田祐希':             { gen: 3, active: false },
  // 4期（卒業）
  '掛橋沙耶香':           { gen: 4, active: false },
  '北川悠理':             { gen: 4, active: false },
  '清宮レイ':             { gen: 4, active: false },
  '早川聖来':             { gen: 4, active: false },
  '松尾美佑':             { gen: 4, active: false },
  '矢久保美緒':           { gen: 4, active: false },
  '佐藤璃果':             { gen: 4, active: false },
  // 6期
  '愛宕心響':             { gen: 6, active: true },
  '大越ひなの':           { gen: 6, active: true },
  '小津玲奈':             { gen: 6, active: true },
  '海邉朱莉':             { gen: 6, active: true },
  '川端晃菜':             { gen: 6, active: true },
  '鈴木佑捺':             { gen: 6, active: true },
  '瀬戸口心月':           { gen: 6, active: true },
  '長嶋凛桜':             { gen: 6, active: true },
  '増田三莉音':           { gen: 6, active: true },
  '森平麗心':             { gen: 6, active: true },
  '矢田萌華':             { gen: 6, active: true },
}

const overall  = ref({ total_applied: 0, total_won: 0, win_rate: 0 })
const rows     = ref([])
const expandedMembers = ref({})
const expandedSingles = ref({}) // key: "memberName::singleName"
const expandedRounds  = ref({}) // key: "memberName::singleName::round"

onMounted(async () => {
  try {
    const [s, d] = await Promise.all([getStats(), getDetailStats()])
    overall.value = s.data
    rows.value    = d.data ?? []
  } catch {
    // 保持預設 0 值
  }
})

// flat rows → member → singleKey → round → rows
// singleKey: numbered singles use single_number (e.g. "41"),
//            albums (single_number=0) use "album::<single_name>" to avoid merging
const memberMap = computed(() => {
  const map = {}
  for (const row of rows.value) {
    if (!map[row.member_name]) {
      map[row.member_name] = { singles: {}, totalApplied: 0, totalWon: 0 }
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

// 成員依期別 → 五十音排序，可過濾只顯示在籍
const sortedMembers = computed(() =>
  Object.entries(memberMap.value)
    .filter(([name]) => !showActiveOnly.value || (MEMBERS[name]?.active ?? true))
    .sort(([a], [b]) => {
      const ga = MEMBERS[a]?.gen ?? 99
      const gb = MEMBERS[b]?.gen ?? 99
      if (ga !== gb) return ga - gb
      return a.localeCompare(b, 'ja')
    })
)

// 單曲依最早 event_date 排序（時間軸升序），讓專輯與單曲按發生順序交錯
function sortedSingles(singles) {
  return Object.entries(singles).sort(([, a], [, b]) =>
    parseDate(a.minEventDate) - parseDate(b.minEventDate)
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
  const n = parseInt(round.match(/\d+/)?.[0] ?? 0)
  return `${n}抽`
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
  return {
    tooltip: {
      trigger: 'axis',
      formatter(params) {
        const d = data[params[0].dataIndex]
        return `${params[0].name}<br/>中選率：${params[0].value}%<br/>應募：${d.applied}　中選：${d.won}`
      },
    },
    grid: { top: 16, right: 24, bottom: 40, left: 54 },
    xAxis: { type: 'category', data: labels },
    yAxis: {
      type: 'value',
      min: 0,
      max: 100,
      axisLabel: { formatter: '{value}%' },
      splitLine: { lineStyle: { type: 'dashed' } },
    },
    series: [{
      type: 'bar',
      data: data.map(d => ({
        value: d.value,
        itemStyle: {
          color: d.value >= 80 ? '#52c41a' : d.value >= 40 ? '#faad14' : '#ff4d4f',
        },
      })),
      label: { show: true, position: 'top', formatter: '{c}%', fontSize: 12 },
    }],
  }
})

// ── 各部長條圖篩選 ───────────────────────────────────────
const barFilterMember = ref('')
const barFilterRound  = ref('')

const allMembers = computed(() =>
  [...new Set(rows.value.map(r => r.member_name))].sort((a, b) => a.localeCompare(b, 'ja'))
)

const allRounds = computed(() => {
  const set = new Set()
  for (const row of rows.value) {
    if (row.lottery_round) set.add(row.lottery_round)
  }
  return [...set].sort((a, b) =>
    parseInt(a.match(/\d+/)?.[0] ?? 0) - parseInt(b.match(/\d+/)?.[0] ?? 0)
  )
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

  return {
    tooltip: {
      trigger: 'axis',
      formatter(params) {
        const d = data[params[0].dataIndex]
        return `${params[0].name}<br/>中選率：${params[0].value}%<br/>應募：${d.applied}　中選：${d.won}`
      },
    },
    grid: { top: 16, right: 24, bottom: 40, left: 54 },
    xAxis: { type: 'category', data: sessions },
    yAxis: {
      type: 'value',
      min: 0,
      max: 100,
      axisLabel: { formatter: '{value}%' },
      splitLine: { lineStyle: { type: 'dashed' } },
    },
    series: [{
      type: 'bar',
      data: data.map(d => ({
        value: d.value,
        itemStyle: {
          color: d.value >= 80 ? '#52c41a' : d.value >= 40 ? '#faad14' : '#ff4d4f',
        },
      })),
      label: { show: true, position: 'top', formatter: '{c}%', fontSize: 12 },
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

// rows 載入後初始化所有 legend 為選取狀態
watch(memberMap, (map) => {
  const sel = {}
  for (const name of Object.keys(map)) sel[name] = true
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
  const rounds = [...roundSet].sort((a, b) =>
    parseInt(a.match(/\d+/)?.[0] ?? 0) - parseInt(b.match(/\d+/)?.[0] ?? 0)
  )

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

  const members = Object.keys(agg).sort((a, b) => a.localeCompare(b, 'ja'))
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

  return {
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
    legend: { data: [...members, '全部'], bottom: 0, type: 'scroll', selected: legendSelected.value },
    grid: { top: 16, right: 24, bottom: 56, left: 54 },
    xAxis: { type: 'category', data: xLabels },
    yAxis: {
      type: 'value',
      min: 0,
      max: 100,
      axisLabel: { formatter: '{value}%' },
      splitLine: { lineStyle: { type: 'dashed' } },
    },
    series,
  }
})
</script>

<style scoped>
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

/* 圖表共用 */
.chart-card {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
  padding: 20px 20px 12px;
  margin-bottom: 32px;
}
.chart-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}
.chart-title {
  font-size: 15px;
  font-weight: 600;
  color: #444;
}
.chart-range-btns {
  display: flex;
  gap: 6px;
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
  justify-content: flex-end;
  margin-bottom: 8px;
}
.member-list { display: flex; flex-direction: column; gap: 12px; }

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
</style>
