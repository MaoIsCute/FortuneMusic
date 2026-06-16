<template>
  <div class="page">
    <h1 class="page-title">🤝 全握統計</h1>

    <!-- 整體統計 -->
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
        <div class="stat-label">整體中選率</div>
        <div class="stat-value highlight">{{ overallRate }}%</div>
      </div>
    </div>

    <el-collapse v-model="openSections" style="margin-top:20px">

    <!-- 類型分析 -->
    <el-collapse-item v-if="byType.length" name="type">
      <template #title><span class="collapse-title">類型分析</span></template>
      <el-table :data="byType" stripe>
        <el-table-column prop="event_type" label="類型" width="80" />
        <el-table-column label="場地" width="80">
          <template #default="{ row }">{{ row.venue || '—' }}</template>
        </el-table-column>
        <el-table-column prop="total_applied" label="應募" width="80" />
        <el-table-column prop="total_won" label="中選" width="80" />
        <el-table-column label="中選率" width="90">
          <template #default="{ row }">
            <span :class="rateClass(row.win_rate)">{{ row.win_rate }}%</span>
          </template>
        </el-table-column>
      </el-table>
    </el-collapse-item>

    <!-- 成員統計 -->
    <el-collapse-item name="member">
      <template #title><span class="collapse-title">成員統計</span></template>
      <el-table :data="memberStats" stripe max-height="400">
        <el-table-column prop="member_name" label="成員" />
        <el-table-column prop="total_applied" label="應募" width="80" />
        <el-table-column prop="total_won" label="中選" width="80" />
        <el-table-column label="中選率" width="90">
          <template #default="{ row }">
            <span :class="rateClass(row.win_rate)">{{ row.win_rate }}%</span>
          </template>
        </el-table-column>
      </el-table>
    </el-collapse-item>

    <!-- 紀錄列表 -->
    <el-collapse-item name="records">
      <template #title><span class="collapse-title">詳細紀錄</span></template>
      <div class="filters">
        <el-select v-model="filterMember" placeholder="選擇成員" clearable @change="loadRecords">
          <el-option v-for="m in memberList" :key="m" :label="m" :value="m" />
        </el-select>
        <el-select v-model="filterType" placeholder="類型" clearable @change="onTypeChange" style="width:100px">
          <el-option label="実体" value="実体" />
          <el-option label="線上" value="線上" />
        </el-select>
        <el-select v-model="filterVenue" placeholder="場地" clearable @change="loadRecords" style="width:130px"
          :disabled="filterType === '線上'">
          <el-option v-for="v in venueList" :key="v" :label="v" :value="v" />
        </el-select>
        <el-select v-model="filterSingle" placeholder="單曲" clearable @change="loadRecords" style="width:100px">
          <el-option v-for="s in singleList" :key="s.single_number" :label="formatSingle(s.single_name)" :value="s.single_number" />
        </el-select>
        <el-select v-model="filterRound" placeholder="抽次" clearable @change="loadRecords" style="width:100px">
          <el-option label="1抽" :value="1" />
          <el-option label="1.5抽" :value="1.5" />
          <el-option label="2抽" :value="2" />
        </el-select>
      </div>
      <el-table :data="records" stripe>
        <el-table-column prop="member_name" label="成員" />
        <el-table-column prop="event_type" label="類型" width="70" />
        <el-table-column label="場地" width="70">
          <template #default="{ row }">{{ row.venue || '—' }}</template>
        </el-table-column>
        <el-table-column prop="event_date" label="日期" width="100" />
        <el-table-column prop="session" label="部數" width="70" />
        <el-table-column label="單曲" >
          <template #default="{ row }">{{ formatSingle(row.single_name) }}</template>
        </el-table-column>
        <el-table-column label="抽次" width="70">
          <template #default="{ row }">{{ row.lottery_round > 0 ? row.lottery_round + '抽' : '—' }}</template>
        </el-table-column>
        <el-table-column prop="applied_count" label="應募" width="60" />
        <el-table-column prop="won_count" label="中選" width="60" align="right" />
        <el-table-column label="中選率" width="75" align="right">
          <template #default="{ row }">
            <span v-if="row.applied_count > 0" :class="rateClass((row.won_count / row.applied_count * 100).toFixed(1))">
              {{ (row.won_count / row.applied_count * 100).toFixed(1) }}%
            </span>
            <span v-else>—</span>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination">
        <el-pagination
          v-model:current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="total, prev, pager, next"
          @current-change="fetchPage"
        />
      </div>
    </el-collapse-item>

    <!-- 成員詳細分析 -->
    <el-collapse-item name="detail">
      <template #title><span class="collapse-title">成員詳細分析</span></template>

      <div class="detail-filters">
        <el-select v-model="detailMember" placeholder="選擇成員" clearable style="width:160px" @change="loadDetail">
          <el-option v-for="m in memberList" :key="m" :label="m" :value="m" />
        </el-select>
        <el-select v-model="detailVenue" placeholder="場地（全部）" clearable style="width:140px" @change="loadDetail">
          <el-option v-for="v in venueList" :key="v" :label="v" :value="v" />
        </el-select>
        <el-checkbox-group v-model="selectedRounds">
          <el-checkbox :value="1">1抽</el-checkbox>
          <el-checkbox :value="1.5">1.5抽</el-checkbox>
          <el-checkbox :value="2">2抽</el-checkbox>
        </el-checkbox-group>
      </div>

      <div v-if="!detailMember" class="empty">請先選擇成員</div>
      <div v-else-if="detailLoading" class="empty">載入中...</div>
      <div v-else-if="detailRows.length === 0" class="empty">無資料</div>
      <el-table v-else :data="detailRows" stripe border>
        <el-table-column label="單曲" width="90" fixed>
          <template #default="{ row }">{{ formatSingle(row.single_name) }}</template>
        </el-table-column>
        <el-table-column
          v-for="col in detailColumns"
          :key="col.key"
          :label="col.label"
          align="center"
          width="90"
        >
          <template #default="{ row }">
            <template v-if="row.cells[col.key]">
              <span :class="rateClass((row.cells[col.key].won / row.cells[col.key].applied * 100).toFixed(1))">
                {{ (row.cells[col.key].won / row.cells[col.key].applied * 100).toFixed(1) }}%
              </span>
              <div class="detail-sub">{{ row.cells[col.key].won }}/{{ row.cells[col.key].applied }}</div>
            </template>
            <span v-else class="text-muted">—</span>
          </template>
        </el-table-column>
      </el-table>
    </el-collapse-item>

    </el-collapse>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { getFullRecords, getFullOverallStats, getFullStatsByMember, getFullStatsBySingle, getFullDetailStats } from '../api/index'
import { sortMembersByGen } from '../utils/members'

const overall = ref({ total_applied: 0, total_won: 0 })
const byType = ref([])
const memberStats = ref([])
const memberList = ref([])
const venueList = ref([])
const singleList = ref([])

const openSections = ref(['type', 'member', 'records'])

// 成員詳細分析
const detailMember = ref('')
const detailVenue  = ref('')
const selectedRounds = ref([1, 1.5])
const detailData   = ref([])
const detailLoading = ref(false)

const detailColumns = computed(() => {
  const sessions = [...new Set(detailData.value.map(r => r.session))].sort()
  const rounds = [...selectedRounds.value].sort((a, b) => a - b)
  const cols = []
  for (const session of sessions) {
    for (const round of rounds) {
      const label = rounds.length === 1
        ? (session || '—')
        : `${session || '—'} ${round}抽`
      cols.push({ session, round, label, key: `${session}:${round}` })
    }
  }
  return cols
})

const detailRows = computed(() => {
  const map = {}
  detailData.value.forEach(r => {
    if (!map[r.single_number])
      map[r.single_number] = { single_number: r.single_number, single_name: r.single_name, cells: {} }
    map[r.single_number].cells[`${r.session}:${r.lottery_round}`] = { applied: r.total_applied, won: r.total_won }
  })
  return Object.values(map).sort((a, b) => a.single_number - b.single_number)
})

async function loadDetail() {
  if (!detailMember.value) { detailData.value = []; return }
  detailLoading.value = true
  try {
    const params = { member: detailMember.value }
    if (detailVenue.value) params.venue = detailVenue.value
    const res = await getFullDetailStats(params)
    detailData.value = res.data ?? []
  } finally {
    detailLoading.value = false
  }
}

watch([detailMember, detailVenue], loadDetail)

const records = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = 50

const filterMember = ref('')
const filterType = ref('')
const filterVenue = ref('')
const filterSingle = ref(null)
const filterRound = ref(null)

const overallRate = computed(() => {
  if (!overall.value.total_applied) return '0.0'
  return (overall.value.total_won / overall.value.total_applied * 100).toFixed(1)
})

onMounted(async () => {
  const [statsRes, memberRes, singleRes] = await Promise.all([
    getFullOverallStats(),
    getFullStatsByMember(),
    getFullStatsBySingle(),
  ])
  overall.value = statsRes.data.overall ?? { total_applied: 0, total_won: 0 }
  byType.value = statsRes.data.by_type ?? []
  memberStats.value = memberRes.data ?? []
  const allNames = new Set()
  memberStats.value.forEach(m => m.member_name.split('・').forEach(n => n.trim() && allNames.add(n.trim())))
  memberList.value = sortMembersByGen([...allNames])
  venueList.value = [...new Set((statsRes.data.by_type ?? []).map(r => r.venue).filter(v => v))]
  singleList.value = singleRes.data ?? []
  await loadRecords()
})

function onTypeChange() {
  if (filterType.value === '線上') filterVenue.value = ''
  loadRecords()
}

async function loadRecords() {
  page.value = 1
  await fetchPage()
}

async function fetchPage() {
  const params = { page: page.value, page_size: pageSize }
  if (filterMember.value) params.member = filterMember.value
  if (filterType.value) params.event_type = filterType.value
  if (filterVenue.value) params.venue = filterVenue.value
  if (filterSingle.value !== null) params.single_number = filterSingle.value
  if (filterRound.value !== null) params.lottery_round = filterRound.value
  const res = await getFullRecords(params)
  records.value = res.data.data ?? []
  total.value = res.data.total ?? 0
}

function rateClass(rate) {
  if (rate >= 80) return 'rate high'
  if (rate >= 40) return 'rate mid'
  return 'rate low'
}

function formatSingle(name) {
  if (!name) return ''
  return name
    .replace(/(\d+)(?:st|nd|rd|th)シングル/, (_, n) => `${n}單`)
    .replace(/(\d+)(?:st|nd|rd|th)アルバム/, (_, n) => `${n}專`)
}
</script>

<style scoped>
.page { background: #f5f7fa; min-height: 100vh; }

.stats-grid {
  display: flex;
  gap: 16px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}
.stat-card {
  background: white;
  border-radius: 10px;
  padding: 16px 24px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.07);
  border: 1px solid #e5e7eb;
  min-width: 140px;
}
.stat-label { font-size: 13px; color: #888; margin-bottom: 6px; }
.stat-value { font-size: 24px; font-weight: bold; color: #222; }
.stat-value.highlight { color: #6366f1; }

:deep(.el-collapse) { border: none; background: transparent; }
:deep(.el-collapse-item) {
  margin-bottom: 12px;
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid #e5e7eb;
  box-shadow: 0 2px 8px rgba(0,0,0,0.07);
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
:deep(.el-collapse-item.is-active .el-collapse-item__header) {
  border-bottom-color: #e5e7eb;
}
:deep(.el-collapse-item__arrow) { color: #6b7280; }
:deep(.el-collapse-item__wrap) { background: white; border: none; }
:deep(.el-collapse-item__content) { padding: 16px 20px 20px; }

.collapse-title { font-weight: 600; font-size: 14px; }
.detail-filters { display: flex; flex-wrap: wrap; gap: 12px; align-items: center; margin-bottom: 16px; }
.detail-sub { font-size: 11px; color: #999; }
.text-muted { color: #bbb; }
.filters { display: flex; gap: 12px; margin-bottom: 16px; flex-wrap: wrap; }
.pagination { margin-top: 16px; display: flex; justify-content: flex-end; }
.rate { font-weight: bold; }
.rate.high { color: #52c41a; }
.rate.mid  { color: #faad14; }
.rate.low  { color: #ff4d4f; }
.won { color: #52c41a; font-weight: bold; }
.lost { color: #ff4d4f; font-weight: bold; }
</style>
