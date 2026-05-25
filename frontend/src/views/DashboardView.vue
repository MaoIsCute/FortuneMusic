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
            v-for="[singleName, single] in sortedSingles(member.singles)"
            :key="singleName"
            class="single-card"
          >
            <!-- 單曲標頭 -->
            <div class="single-header" @click="toggleSingle(memberName, singleName)">
              <span class="single-name">{{ formatSingle(singleName) }}</span>
              <span class="single-summary">
                {{ single.totalApplied }} 應 / {{ single.totalWon }} 中
                <span class="rate">{{ calcRate(single.totalWon, single.totalApplied) }}%</span>
              </span>
              <span class="chevron">{{ isSingleExpanded(memberName, singleName) ? '▲' : '▼' }}</span>
            </div>

            <!-- 單曲展開：依次數分組 -->
            <div v-if="isSingleExpanded(memberName, singleName)" class="single-body">
              <div
                v-for="[round, roundData] in sortedRounds(single.rounds)"
                :key="round"
                class="round-section"
              >
                <div class="round-label">{{ round }}</div>
                <table class="detail-table">
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
import { ref, computed, onMounted } from 'vue'
import { getStats, getDetailStats } from '../api/index'

const overall  = ref({ total_applied: 0, total_won: 0, win_rate: 0 })
const rows     = ref([])
const expandedMembers = ref({})
const expandedSingles = ref({}) // key: "memberName::singleName"

onMounted(async () => {
  try {
    const [s, d] = await Promise.all([getStats(), getDetailStats()])
    overall.value = s.data
    rows.value    = d.data ?? []
  } catch {
    // 保持預設 0 值
  }
})

// event_name 格式："41stシングル「歌名」/第3次"
// singleName = "/" 前的部分，round = "/" 後的部分
function parseEventName(eventName) {
  const idx = eventName.lastIndexOf('/')
  if (idx === -1) return { singleName: eventName, round: '' }
  return {
    singleName: eventName.slice(0, idx),
    round:      eventName.slice(idx + 1),
  }
}

// flat rows → member → singleName → round → rows
const memberMap = computed(() => {
  const map = {}
  for (const row of rows.value) {
    if (!map[row.member_name]) {
      map[row.member_name] = { singles: {}, totalApplied: 0, totalWon: 0 }
    }
    const m = map[row.member_name]
    m.totalApplied += row.total_applied
    m.totalWon     += row.total_won

    const { singleName, round } = parseEventName(row.event_name)
    if (!m.singles[singleName]) {
      m.singles[singleName] = { rounds: {}, totalApplied: 0, totalWon: 0 }
    }
    const s = m.singles[singleName]
    s.totalApplied += row.total_applied
    s.totalWon     += row.total_won

    const roundKey = round || '—'
    if (!s.rounds[roundKey]) s.rounds[roundKey] = { rows: [] }
    s.rounds[roundKey].rows.push(row)
  }
  return map
})

// 成員依名稱排序
const sortedMembers = computed(() =>
  Object.entries(memberMap.value).sort(([a], [b]) => a.localeCompare(b, 'ja'))
)

// 單曲依單曲號排序（小 → 大）
function sortedSingles(singles) {
  return Object.entries(singles).sort(([a], [b]) => {
    const na = parseInt(a.match(/^(\d+)/)?.[1] ?? 0)
    const nb = parseInt(b.match(/^(\d+)/)?.[1] ?? 0)
    return na - nb
  })
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
function formatSingle(singleName) {
  return singleName.replace(/(\d+)(?:st|nd|rd|th)シングル/, (_, n) => `${n}單`)
}
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

/* 成員層 */
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

/* 次數層 */
.round-section { margin-top: 12px; }
.round-label {
  font-size: 13px;
  font-weight: 600;
  color: #888;
  margin-bottom: 6px;
  padding-left: 4px;
  border-left: 3px solid var(--color-primary);
  padding-left: 8px;
}

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
