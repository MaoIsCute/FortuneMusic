<template>
  <div class="page">
    <h1 class="page-title">📊 全握分析</h1>

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
        <div class="filters">
          <el-select v-model="memberFilterType" placeholder="類型" clearable style="width:100px" @change="onMemberTypeChange">
            <el-option label="実体" value="実体" />
            <el-option label="線上" value="線上" />
          </el-select>
          <el-select v-model="memberFilterVenue" placeholder="場地" clearable style="width:160px"
            :disabled="memberFilterType === '線上'" @change="loadMemberStats">
            <el-option v-for="v in venueList" :key="v" :label="v" :value="v" />
          </el-select>
        </div>
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

      <!-- 成員詳細分析 -->
      <el-collapse-item name="detail">
        <template #title><span class="collapse-title">成員詳細分析</span></template>

        <div class="detail-filters">
          <el-select v-model="detailMember" placeholder="選擇成員" clearable style="width:160px" @change="loadDetail">
            <el-option v-for="m in memberList" :key="m" :label="m" :value="m" />
          </el-select>
          <el-select v-model="detailType" placeholder="類型（全部）" clearable style="width:120px" @change="onDetailTypeChange">
            <el-option label="実体" value="実体" />
            <el-option label="線上" value="線上" />
          </el-select>
          <el-select v-model="detailVenue" :placeholder="detailType === '線上' ? '無' : '場地（全部）'" clearable style="width:140px"
            :disabled="detailType === '線上'" @change="loadDetail">
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
          <el-table-column label="搭檔" width="130">
            <template #default="{ row }">
              <span v-if="row.partner" class="partner-name">{{ row.partner }}</span>
              <span v-else class="text-muted">—</span>
            </template>
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
import { ref, computed, onMounted } from 'vue'
import { getFullOverallStats, getFullStatsByMember, getFullDetailStats, getFullStatsBySingle } from '../api/index'
import { sortMembersByGen } from '../utils/members'

const overall    = ref({ total_applied: 0, total_won: 0 })
const byType     = ref([])
const memberStats = ref([])
const memberList = ref([])
const venueList  = ref([])

const memberFilterType  = ref('')
const memberFilterVenue = ref('')

const openSections = ref(['type', 'member'])

const detailMember   = ref('')
const detailType     = ref('')
const detailVenue    = ref('')
const selectedRounds = ref([1, 1.5])
const detailData     = ref([])
const detailLoading  = ref(false)

const overallRate = computed(() => {
  if (!overall.value.total_applied) return '0.0'
  return (overall.value.total_won / overall.value.total_applied * 100).toFixed(1)
})

const detailColumns = computed(() => {
  const sessions = [...new Set(detailData.value.map(r => r.session))].sort()
  const rounds   = [...selectedRounds.value].sort((a, b) => a - b)
  const cols = []
  for (const session of sessions) {
    for (const round of rounds) {
      const label = rounds.length === 1 ? (session || '—') : `${session || '—'} ${round}抽`
      cols.push({ session, round, label, key: `${session}:${round}` })
    }
  }
  return cols
})

const detailRows = computed(() => {
  const map = {}
  detailData.value.forEach(r => {
    const key = `${r.single_number}:${r.member_name}`
    if (!map[key]) {
      const partners = r.member_name.split('・').filter(n => n !== detailMember.value)
      map[key] = {
        single_number: r.single_number,
        single_name:   r.single_name,
        partner:       partners.length > 0 ? partners.join('・') : '',
        cells: {},
      }
    }
    map[key].cells[`${r.session}:${r.lottery_round}`] = { applied: r.total_applied, won: r.total_won }
  })
  return Object.values(map).sort((a, b) =>
    a.single_number !== b.single_number ? a.single_number - b.single_number : a.partner.localeCompare(b.partner)
  )
})

function onMemberTypeChange() {
  if (memberFilterType.value === '線上') memberFilterVenue.value = ''
  loadMemberStats()
}

async function loadMemberStats() {
  const params = {}
  if (memberFilterType.value)  params.event_type = memberFilterType.value
  if (memberFilterVenue.value) params.venue = memberFilterVenue.value
  const res = await getFullStatsByMember(params)
  memberStats.value = res.data ?? []
}

function onDetailTypeChange() {
  if (detailType.value === '線上') detailVenue.value = ''
  loadDetail()
}

async function loadDetail() {
  if (!detailMember.value) { detailData.value = []; return }
  detailLoading.value = true
  try {
    const params = { member: detailMember.value }
    if (detailType.value)  params.event_type = detailType.value
    if (detailVenue.value) params.venue = detailVenue.value
    const res = await getFullDetailStats(params)
    detailData.value = res.data ?? []
  } finally {
    detailLoading.value = false
  }
}

onMounted(async () => {
  const statsRes = await getFullOverallStats()
  overall.value = statsRes.data.overall ?? { total_applied: 0, total_won: 0 }
  byType.value  = statsRes.data.by_type ?? []
  venueList.value = [...new Set(byType.value.map(r => r.venue).filter(v => v))]
  await loadMemberStats()
  const allNames = new Set()
  memberStats.value.forEach(m => m.member_name.split('・').forEach(n => n.trim() && allNames.add(n.trim())))
  memberList.value = sortMembersByGen([...allNames])
})

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
.filters { display: flex; gap: 12px; margin-bottom: 16px; flex-wrap: wrap; }
.detail-filters { display: flex; flex-wrap: wrap; gap: 12px; align-items: center; margin-bottom: 16px; }
.detail-sub { font-size: 11px; color: #999; }
.partner-name { font-size: 12px; color: #6366f1; }
.text-muted { color: #bbb; }
.empty { text-align: center; color: #999; padding: 40px 0; }
.rate { font-weight: bold; }
.rate.high { color: #52c41a; }
.rate.mid  { color: #faad14; }
.rate.low  { color: #ff4d4f; }
</style>
