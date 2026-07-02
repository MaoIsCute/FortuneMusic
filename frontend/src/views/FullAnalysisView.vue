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
        <el-table table-layout="auto" :data="byType" stripe>
          <el-table-column prop="event_type" label="類型" min-width="60" sortable />
          <el-table-column label="場地" min-width="200" sortable :sort-by="row => row.venue || ''">
            <template #default="{ row }">{{ row.venue || '—' }}</template>
          </el-table-column>
          <el-table-column prop="total_applied" label="應募" min-width="70" sortable />
          <el-table-column prop="total_won" label="中選" min-width="70" sortable />
          <el-table-column prop="win_rate_num" label="中選率" min-width="80" sortable>
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
          <el-table-column prop="member_name" label="成員" sortable />
          <el-table-column prop="total_applied" label="應募" width="80" sortable />
          <el-table-column prop="total_won" label="中選" width="80" sortable />
          <el-table-column prop="win_rate_num" label="中選率" width="90" sortable>
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
            <el-option v-for="m in memberList" :key="m.name" :label="m.name" :value="m.name">
            <span :style="{ color: GROUP_COLORS[m.group] }">{{ m.name }}</span>
          </el-option>
          </el-select>
          <el-select v-model="detailType" style="width:120px" @change="onDetailTypeChange">
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
        <el-table v-else :data="detailRows" stripe border table-layout="auto">
          <el-table-column label="單曲" width="90" fixed>
            <template #default="{ row }">{{ formatSingle(row.single_name) || row.single_number + '單' }}</template>
          </el-table-column>
          <el-table-column label="場地" min-width="120">
            <template #default="{ row }">{{ row.venue || '—' }}</template>
          </el-table-column>
          <el-table-column label="搭檔" width="130">
            <template #default="{ row }">
              <span v-if="row.partner" class="partner-name">{{ row.partner }}</span>
              <span v-else class="text-muted">—</span>
            </template>
          </el-table-column>
          <el-table-column
            v-for="session in detailSessions"
            :key="session"
            :label="session || '—'"
            align="center"
          >
            <el-table-column
              v-for="round in selectedRoundsSorted"
              :key="round"
              :label="round + '抽'"
              align="center"
              width="80"
            >
              <template #default="{ row }">
                <template v-if="row.cells[`${session}:${round}`]">
                  <span :class="rateClass((row.cells[`${session}:${round}`].won / row.cells[`${session}:${round}`].applied * 100).toFixed(1))">
                    {{ (row.cells[`${session}:${round}`].won / row.cells[`${session}:${round}`].applied * 100).toFixed(1) }}%
                  </span>
                  <div class="detail-sub">{{ row.cells[`${session}:${round}`].won }}/{{ row.cells[`${session}:${round}`].applied }}</div>
                </template>
                <span v-else class="text-muted">—</span>
              </template>
            </el-table-column>
          </el-table-column>
        </el-table>
      </el-collapse-item>

    </el-collapse>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getFullOverallStats, getFullStatsByMember, getFullDetailStats, getFullStatsBySingle } from '../api/index'
import { sortMembersByGroupAndGen } from '../utils/members'

const GROUP_COLORS = { nogizaka46: '#9333ea', sakurazaka46: '#ec4899', hinatazaka46: '#0ea5e9' }

const overall    = ref({ total_applied: 0, total_won: 0 })
const byType     = ref([])
const memberStats = ref([])
const memberList = ref([])
const venueList  = ref([])

const memberFilterType  = ref('')
const memberFilterVenue = ref('')

const openSections = ref(['type', 'member'])

const detailMember   = ref('')
const detailType     = ref('実体')
const detailVenue    = ref('')
const selectedRounds = ref([1])
const detailData     = ref([])
const detailLoading  = ref(false)

const overallRate = computed(() => {
  if (!overall.value.total_applied) return '0.0'
  return (overall.value.total_won / overall.value.total_applied * 100).toFixed(1)
})

const detailSessions      = computed(() => [...new Set(detailData.value.map(r => r.session))].sort())
const selectedRoundsSorted = computed(() => [...selectedRounds.value].sort((a, b) => a - b))

const detailRows = computed(() => {
  const map = {}
  detailData.value.forEach(r => {
    const key = `${r.single_number}:${r.member_name}:${r.venue}`
    if (!map[key]) {
      const partners = r.member_name.split('・').filter(n => n !== detailMember.value)
      map[key] = {
        single_number: r.single_number,
        single_name:   r.single_name,
        venue:         r.venue || '',
        partner:       partners.length > 0 ? partners.join('・') : '',
        cells: {},
      }
    }
    map[key].cells[`${r.session}:${r.lottery_round}`] = { applied: r.total_applied, won: r.total_won }
  })
  return Object.values(map).sort((a, b) =>
    a.single_number !== b.single_number
      ? a.single_number - b.single_number
      : a.venue.localeCompare(b.venue) || a.partner.localeCompare(b.partner)
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
  memberStats.value = (res.data ?? []).map(r => ({ ...r, win_rate_num: parseFloat(r.win_rate) }))
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
  byType.value  = (statsRes.data.by_type ?? []).map(r => ({ ...r, win_rate_num: parseFloat(r.win_rate) }))
  venueList.value = [...new Set(byType.value.map(r => r.venue).filter(v => v))]
  await loadMemberStats()
  const nameGroupMap = new Map()
  memberStats.value.forEach(m => m.member_name.split('・').forEach(n => { n = n.trim(); if (n) nameGroupMap.set(n, m.group || '') }))
  memberList.value = sortMembersByGroupAndGen([...nameGroupMap.entries()].map(([name, group]) => ({ name, group })))
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
:deep(.el-table .cell) { white-space: nowrap; }

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
