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

    <!-- 類型分析 -->
    <div v-if="byType.length" class="section-card">
      <h2 class="section-title">類型分析</h2>
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
    </div>

    <!-- 成員統計 -->
    <div class="section-card">
      <h2 class="section-title">成員統計</h2>
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
    </div>

    <!-- 紀錄列表 -->
    <div class="section-card">
      <h2 class="section-title">詳細紀錄</h2>
      <div class="filters">
        <el-select v-model="filterMember" placeholder="選擇成員" clearable @change="loadRecords">
          <el-option v-for="m in memberList" :key="m" :label="m" :value="m" />
        </el-select>
        <el-select v-model="filterType" placeholder="類型" clearable @change="loadRecords">
          <el-option label="実体" value="実体" />
          <el-option label="線上" value="線上" />
        </el-select>
        <el-input v-model="filterVenue" placeholder="場地" clearable @change="loadRecords" style="width:120px" />
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
        <el-table-column prop="won_count" label="中選" width="60" />
        <el-table-column label="結果" width="70">
          <template #default="{ row }">
            <span :class="row.won_count > 0 ? 'won' : 'lost'">
              {{ row.won_count > 0 ? '當選' : '落選' }}
            </span>
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
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getFullRecords, getFullOverallStats, getFullStatsByMember } from '../api/index'
import { sortMembersByGen } from '../utils/members'

const overall = ref({ total_applied: 0, total_won: 0 })
const byType = ref([])
const memberStats = ref([])
const memberList = ref([])

const records = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = 50

const filterMember = ref('')
const filterType = ref('')
const filterVenue = ref('')
const filterRound = ref(null)

const overallRate = computed(() => {
  if (!overall.value.total_applied) return '0.0'
  return (overall.value.total_won / overall.value.total_applied * 100).toFixed(1)
})

onMounted(async () => {
  const [statsRes, memberRes] = await Promise.all([
    getFullOverallStats(),
    getFullStatsByMember(),
  ])
  overall.value = statsRes.data.overall ?? { total_applied: 0, total_won: 0 }
  byType.value = statsRes.data.by_type ?? []
  memberStats.value = memberRes.data ?? []
  memberList.value = sortMembersByGen(memberStats.value.map(m => m.member_name))
  await loadRecords()
})

async function loadRecords() {
  page.value = 1
  await fetchPage()
}

async function fetchPage() {
  const params = { page: page.value, page_size: pageSize }
  if (filterMember.value) params.member = filterMember.value
  if (filterType.value) params.event_type = filterType.value
  if (filterVenue.value) params.venue = filterVenue.value
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
.stats-grid {
  display: flex;
  gap: 16px;
  margin-bottom: 24px;
  flex-wrap: wrap;
}
.stat-card {
  background: white;
  border-radius: 8px;
  padding: 16px 24px;
  box-shadow: 0 1px 6px rgba(0,0,0,0.08);
  min-width: 140px;
}
.stat-label { font-size: 13px; color: #888; margin-bottom: 6px; }
.stat-value { font-size: 24px; font-weight: bold; color: #222; }
.stat-value.highlight { color: #6366f1; }
.section-card {
  background: white;
  border-radius: 8px;
  padding: 20px 24px;
  box-shadow: 0 1px 6px rgba(0,0,0,0.08);
  margin-bottom: 20px;
}
.section-title { font-size: 16px; font-weight: bold; margin: 0 0 16px; }
.filters {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}
.pagination { margin-top: 16px; display: flex; justify-content: flex-end; }
.rate { font-weight: bold; }
.rate.high { color: #52c41a; }
.rate.mid  { color: #faad14; }
.rate.low  { color: #ff4d4f; }
.won { color: #52c41a; font-weight: bold; }
.lost { color: #ff4d4f; font-weight: bold; }
</style>
