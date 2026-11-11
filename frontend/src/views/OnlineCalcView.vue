<template>
  <div class="page">
    <h1 class="page-title">🎫 付款模擬</h1>
    <p class="page-subtitle">剛開完一抽、要決定付哪幾張單時用：券面上已持有的張數 ＋ 勾選要付款的最新一抽中選單，即時算出每天每部每個成員的張數</p>

    <template v-if="loaded">
      <ErrorState v-if="loadFailed" />
      <EmptyState v-else-if="singles.length === 0" />
      <template v-else>
        <!-- 多個團體都有資料時才需要切換；每個團體只看最新的那張單曲 -->
        <el-radio-group v-if="singles.length > 1" v-model="selectedKey" class="group-switch" @change="loadSummary">
          <el-radio-button v-for="s in singles" :key="singleKey(s)" :value="singleKey(s)">
            {{ groupLabel(s.group) }}
          </el-radio-button>
        </el-radio-group>

        <div v-if="summaryLoading" v-loading="true" class="summary-loading"></div>
        <ErrorState v-else-if="summaryFailed" />

        <template v-else-if="summary">
          <h2 class="event-title">
            <span class="round-badge">{{ summary.latest_round ? `第${summary.latest_round}次` : '最新' }}抽選</span>
            {{ groupLabel(summary.group) }}　{{ summary.single_name || `${summary.single_number}單` }}
          </h2>

          <el-alert
            v-if="!summary.has_ticket"
            type="warning"
            :closable="false"
            show-icon
            class="section"
            title="還沒有同步過券面（保有数）"
            description="「已確定」暫時用前幾抽的中選數推估，假設全部都有付款。用最新版同步工具跑一次「個握同步」就會改用券面上實際持有的張數。"
          />
          <div v-else class="ticket-note">
            已確定張數來自 forTUNE meets 券面的「保有数」（{{ formatDateTime(summary.ticket_scraped_at) }} 同步）<template v-if="summary.paid_order_count > 0">，
            第{{ summary.latest_round }}抽已付款的 {{ summary.paid_order_count }} 張單已經包含在內</template>
          </div>

          <div class="stats-row">
            <el-card class="stat-card">
              <div class="stat-num base">{{ summary.base_total }}</div>
              <div class="stat-label">已確定</div>
            </el-card>
            <el-card class="stat-card">
              <div class="stat-num add">+{{ checkedTotal }}</div>
              <div class="stat-label">勾選的本次中選（{{ checked.length }}/{{ summary.orders.length }} 張單）</div>
            </el-card>
            <el-card class="stat-card">
              <div class="stat-num total">{{ summary.base_total + checkedTotal }}</div>
              <div class="stat-label">付款後總張數</div>
            </el-card>
            <el-card class="stat-card" v-if="busiestDay">
              <div class="stat-num">{{ busiestDay.total }}</div>
              <div class="stat-label">最多的一天（{{ formatDate(busiestDay.date) }}）</div>
            </el-card>
          </div>

          <div class="layout">
            <!-- 可勾選的最新一抽中選單 -->
            <el-card class="section orders-card">
              <template #header>
                <div class="card-header">
                  <span>{{ summary.has_ticket ? '還沒付款的中選單' : `第${summary.latest_round}抽中選的單` }}</span>
                  <span class="header-actions">
                    <el-button size="small" text @click="checkAll">全選</el-button>
                    <el-button size="small" text @click="checkNone">全不選</el-button>
                  </span>
                </div>
              </template>

              <div v-if="summary.orders.length === 0" class="empty">
                {{ summary.has_ticket ? '沒有待付款的中選單，下方就是目前券面上的張數' : '這一抽沒有中選的單' }}
              </div>

              <el-checkbox-group v-else v-model="checked" class="order-list">
                <div v-for="o in summary.orders" :key="o.order_id" class="order-row">
                  <el-checkbox :value="o.order_id" class="order-check">
                    <span class="order-id">{{ o.order_id }}</span>
                    <span class="order-meta">{{ o.lottery_round ? `第${o.lottery_round}抽` : '無抽次' }} ・ {{ o.total_count }}張 ・ {{ o.slots.length }}格</span>
                  </el-checkbox>
                  <a
                    class="order-link"
                    :href="`https://fortunemusic.jp/mypage/apply_detail/${o.order_id}/`"
                    target="_blank"
                    rel="noopener"
                    title="到網站看這張單的明細"
                  >↗</a>
                </div>
              </el-checkbox-group>
            </el-card>

            <!-- 券面風格：日期摺疊面板 × (時間帯, メンバー) -->
            <div class="ticket-sheet">
              <div v-if="grid.length === 0" class="empty">沒有任何張數</div>

              <div
                v-for="d in grid"
                :key="d.date"
                class="tgl"
                :class="{ open: openDates.includes(d.date) }"
              >
                <button type="button" class="tgl-hook" @click="toggleDate(d.date)">
                  <span class="tgl-date">{{ formatDate(d.date) }}</span>
                  <span class="tgl-sum">
                    {{ d.base }}<template v-if="d.add > 0"> <span class="add">+{{ d.add }}</span></template>
                    ＝ <b>{{ d.total }}枚</b>
                  </span>
                  <span class="tgl-arrow" aria-hidden="true"></span>
                </button>

                <div v-if="openDates.includes(d.date)" class="tgl-content">
                  <div class="table-scroll">
                    <table class="ticket-table">
                      <thead>
                        <tr>
                          <th>時間帯</th>
                          <th>メンバー</th>
                          <th>已確定</th>
                          <th>本次</th>
                          <th>合計</th>
                        </tr>
                      </thead>
                      <tbody>
                        <tr v-for="r in d.rows" :key="`${r.session}:${r.member_name}`" :class="{ 'row-new': r.base === 0 }">
                          <td>
                            <span class="nowrap">{{ r.session }}</span>
                            <span v-if="r.time_range" class="nowrap time-range">{{ r.time_range }}</span>
                          </td>
                          <td><span class="nowrap">{{ r.member_name }}</span></td>
                          <td>{{ r.base }}枚</td>
                          <td :class="{ add: r.add > 0, muted: r.add === 0 }">{{ r.add ? `+${r.add}枚` : '—' }}</td>
                          <td class="total-cell">{{ r.total }}枚</td>
                        </tr>
                      </tbody>
                    </table>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="hint-text">
            💡 已確定（券面保有数）跟可以勾的未付款單，都是同步工具跑「個握同步」時一起讀取的快照；
            付完款或剛開完新一抽之後，重新同步一次才會反映最新狀態。
          </div>
        </template>
      </template>
    </template>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { getOnlineCalcSingles, getOnlineCalcSummary } from '../api/index'
import EmptyState from '../components/EmptyState.vue'
import ErrorState from '../components/ErrorState.vue'

const singles        = ref([])
const summary        = ref(null)
const selectedKey    = ref(null)
const checked        = ref([])
const openDates      = ref([])
const loaded         = ref(false)
const loadFailed     = ref(false)
// 單曲清單（整頁）跟單張單曲明細分開記錯誤：切換團體時某一次明細失敗，只在結果區塊顯示錯誤，
// 團體切換鈕還在，可以改選別的或重選一次
const summaryLoading = ref(false)
const summaryFailed  = ref(false)

const GROUP_LABELS = { nogizaka46: '乃木坂46', sakurazaka46: '櫻坂46', hinatazaka46: '日向坂46' }
const WEEKDAYS = ['日', '月', '火', '水', '木', '金', '土']

function groupLabel(g) {
  return GROUP_LABELS[g] || g || '—'
}

function singleKey(s) {
  return `${s.group}:${s.single_number}`
}

// event_date 是 "YYYY/MM/DD"，排成券面頁的「2026年12月6日（日）」
function formatDate(date) {
  const m = String(date).match(/^(\d{4})\/(\d{1,2})\/(\d{1,2})$/)
  if (!m) return date
  const d = new Date(+m[1], +m[2] - 1, +m[3])
  return `${+m[1]}年${+m[2]}月${+m[3]}日（${WEEKDAYS[d.getDay()]}）`
}

function formatDateTime(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const pad = n => String(n).padStart(2, '0')
  return `${d.getMonth() + 1}/${d.getDate()} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function toggleDate(date) {
  openDates.value = openDates.value.includes(date)
    ? openDates.value.filter(d => d !== date)
    : [...openDates.value, date]
}

const checkedTotal = computed(() => {
  if (!summary.value) return 0
  const picked = new Set(checked.value)
  return summary.value.orders
    .filter(o => picked.has(o.order_id))
    .reduce((sum, o) => sum + o.total_count, 0)
})

// grid 是畫面的主結果：已確定 + 勾選的本次中選，攤成「日期 → (部, 成員) → 張數」。
// 加總放在前端做，所以勾選是即時的，不用每次重新打後端。
const grid = computed(() => {
  if (!summary.value) return []
  const picked = new Set(checked.value)
  const byDate = new Map()
  // 時刻只有券面有，本次新增、券面上還沒有的格子用同一天同一部的時刻補上
  const timeRanges = new Map()
  for (const s of summary.value.base) {
    if (s.time_range) timeRanges.set(`${s.event_date} ${s.session}`, s.time_range)
  }

  function cell(slot) {
    if (!byDate.has(slot.event_date)) byDate.set(slot.event_date, new Map())
    const rows = byDate.get(slot.event_date)
    const key = `${slot.session} ${slot.member_name}`
    if (!rows.has(key)) {
      rows.set(key, {
        session: slot.session,
        time_range: timeRanges.get(`${slot.event_date} ${slot.session}`) || '',
        member_name: slot.member_name,
        base: 0,
        add: 0,
      })
    }
    return rows.get(key)
  }

  for (const slot of summary.value.base) cell(slot).base += slot.count
  for (const order of summary.value.orders) {
    if (!picked.has(order.order_id)) continue
    for (const slot of order.slots) cell(slot).add += slot.count
  }

  const sessionOrder = s => {
    const m = String(s).match(/\d+/)
    return m ? parseInt(m[0]) : 0
  }

  return [...byDate.entries()]
    .map(([date, rows]) => {
      const list = [...rows.values()]
        .map(r => ({ ...r, total: r.base + r.add }))
        .sort((a, b) => sessionOrder(a.session) - sessionOrder(b.session)
          || a.session.localeCompare(b.session)
          || a.member_name.localeCompare(b.member_name))
      return {
        date,
        rows: list,
        base:  list.reduce((n, r) => n + r.base, 0),
        add:   list.reduce((n, r) => n + r.add, 0),
        total: list.reduce((n, r) => n + r.total, 0),
      }
    })
    .filter(d => d.total > 0)
    .sort((a, b) => a.date.localeCompare(b.date))
})

const busiestDay = computed(() => {
  let best = null
  for (const d of grid.value) {
    if (!best || d.total > best.total) best = d
  }
  return best
})

function checkAll() {
  checked.value = summary.value?.orders.map(o => o.order_id) ?? []
}

function checkNone() {
  checked.value = []
}

async function loadSummary() {
  if (!selectedKey.value) return
  const [group, singleNumber] = selectedKey.value.split(':')
  summaryLoading.value = true
  summaryFailed.value = false
  try {
    const res = await getOnlineCalcSummary({ group, single_number: singleNumber })
    summary.value = res.data
    // 預設全部勾起來：使用者真正想知道的是「全付的話會變幾張」，再往下取消比較快
    checkAll()
    // 預設只展開有本次新增的日期，其餘收合，跟券面頁一樣一眼看完所有日期
    openDates.value = grid.value.filter(d => d.add > 0).map(d => d.date)
  } catch {
    summary.value = null
    summaryFailed.value = true
  } finally {
    summaryLoading.value = false
  }
}

async function load() {
  try {
    const res = await getOnlineCalcSingles()
    singles.value = res.data ?? []
    if (singles.value.length > 0) {
      selectedKey.value = singleKey(singles.value[0])
      await loadSummary()
    }
  } catch {
    loadFailed.value = true
  } finally {
    loaded.value = true
  }
}

load()
</script>

<style scoped>
.page-subtitle { color: #888; font-size: 13px; margin: 0 0 20px; }
html.dark .page-subtitle { color: #9aa3b5; }

/* 配色取自 forTUNE meets 券面頁（online_handshake_detail） */
.page {
  --tk-mint: #55e3cb;
  --tk-mint-border: #1ddab6;
  --tk-ink: #005947;
  --tk-text: #4b4b4b;
  --tk-closed: #ebebeb;
  --tk-row-line: #eef2f1;
  --tk-row-bg: #fff;
}
html.dark .page {
  --tk-text: #e2e8f0;
  --tk-closed: #2a2f3a;
  --tk-row-line: #2f3644;
  --tk-row-bg: #1e2430;
}

.section { margin-bottom: 16px; }
.group-switch { margin-bottom: 16px; }
.summary-loading { min-height: 200px; }
.empty { color: #999; text-align: center; padding: 32px 0; }
.hint-text { font-size: 12px; color: #999; margin-top: 12px; line-height: 1.6; }
.ticket-note { font-size: 12px; color: #888; margin: -4px 0 16px; }

.event-title {
  font-size: 20px;
  font-weight: 700;
  color: var(--tk-text);
  margin: 0 0 16px;
  line-height: 1.5;
}
.round-badge {
  display: inline-block;
  background: var(--tk-mint);
  color: var(--tk-ink);
  font-size: 13px;
  font-weight: 700;
  padding: 2px 10px;
  border-radius: 4px;
  margin-right: 8px;
  vertical-align: middle;
}

.stats-row { display: flex; gap: 16px; margin-bottom: 16px; flex-wrap: wrap; }
.stat-card { flex: 1; min-width: 140px; text-align: center; }
.stat-num { font-size: 26px; font-weight: bold; color: var(--tk-ink); }
html.dark .stat-num { color: var(--tk-mint); }
.stat-label { font-size: 13px; color: #888; margin-top: 4px; }
.stat-num.base { color: #64748b; }
html.dark .stat-num.base { color: #94a3b8; }

.add { color: #16a34a; font-weight: 600; }
html.dark .add { color: #4ade80; }
.muted { color: #b0b0b0; }

.layout { display: flex; gap: 16px; align-items: flex-start; flex-wrap: wrap; }
.orders-card { flex: 0 0 280px; min-width: 240px; }
.ticket-sheet { flex: 1; min-width: 300px; }

.card-header { display: flex; justify-content: space-between; align-items: center; }
.header-actions { display: flex; gap: 4px; }

.order-list { display: block; }
.order-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--el-border-color-lighter);
  padding: 2px 0;
}
.order-row:last-child { border-bottom: none; }
.order-check { height: auto; }
.order-id { font-family: ui-monospace, monospace; font-size: 13px; }
.order-meta { display: block; font-size: 11px; color: #999; }
.order-link { color: #999; text-decoration: none; padding: 0 4px; font-size: 13px; }
.order-link:hover { color: var(--tk-ink); }

/* 日期摺疊面板：收合是灰底、展開是薄荷綠底＋外框，右邊圓形箭頭，比照券面頁 .tgl02 */
.tgl { margin-bottom: 16px; border: 1.5px solid transparent; }
.tgl.open { border-color: var(--tk-mint-border); }
.tgl-hook {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  border: none;
  cursor: pointer;
  text-align: left;
  font: inherit;
  font-size: 15px;
  padding: 22px 24px;
  background: var(--tk-closed);
  color: var(--tk-text);
}
.tgl.open .tgl-hook { background: var(--tk-mint); color: var(--tk-ink); }
.tgl-date { flex: 1; }
.tgl-sum { font-size: 13px; white-space: nowrap; }
.tgl.open .tgl-sum .add { color: var(--tk-ink); }
.tgl-arrow {
  flex: 0 0 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--tk-mint);
  position: relative;
}
.tgl.open .tgl-arrow { background: #fff; }
.tgl-arrow::after {
  content: '';
  position: absolute;
  left: 50%;
  top: 50%;
  border-left: 5px solid transparent;
  border-right: 5px solid transparent;
  border-top: 6px solid var(--tk-ink);
  transform: translate(-50%, -35%);
}
.tgl.open .tgl-arrow::after { transform: translate(-50%, -65%) rotate(180deg); }

.tgl-content { padding-top: 3px; }
.table-scroll { overflow-x: auto; }
.ticket-table { width: 100%; border-collapse: collapse; font-size: 14px; }
.ticket-table th {
  background: var(--tk-mint);
  color: var(--tk-ink);
  font-weight: 400;
  font-size: 13px;
  padding: 16px 10px;
  text-align: center;
  white-space: nowrap;
}
.ticket-table td {
  background: var(--tk-row-bg);
  color: var(--tk-text);
  padding: 12px 10px;
  text-align: center;
  border-bottom: 1.5px solid var(--tk-row-line);
}
.ticket-table tbody tr:last-child td { border-bottom: none; }
.ticket-table tr.row-new td { background: rgba(85, 227, 203, 0.12); }
.ticket-table .total-cell { font-weight: 700; color: var(--tk-ink); }
html.dark .ticket-table .total-cell { color: var(--tk-mint); }
.nowrap { display: inline-block; white-space: nowrap; }
.time-range { margin-left: 6px; }
</style>
