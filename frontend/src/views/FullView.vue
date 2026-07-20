<template>
  <div class="page">
    <h1 class="page-title">📋 全握紀錄</h1>
    <template v-if="loaded">
    <ErrorState v-if="loadFailed" />
    <EmptyState v-else-if="isEmpty" />
    <template v-else>

    <div class="card">
      <div class="filters">
        <el-select v-model="filterGroup" placeholder="團體" clearable style="width:120px" @change="onGroupChange">
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
        <el-select v-model="filterMember" placeholder="選擇成員" clearable @change="loadRecords">
          <el-option v-for="m in memberList" :key="`${m.group}:${m.name}`" :label="m.name" :value="`${m.group}:${m.name}`">
            <span :style="{ color: GROUP_COLORS[m.group || filterGroup] }">{{ m.name }}</span>
          </el-option>
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
          <el-option v-for="s in singleList" :key="`${s.group}:${s.single_number}`" :label="formatSingle(s.single_name)" :value="`${s.group}:${s.single_number}`">
            <span :style="{ color: GROUP_COLORS[s.group || filterGroup] }">{{ formatSingle(s.single_name) }}</span>
          </el-option>
        </el-select>
        <el-select v-model="filterRound" placeholder="抽次" clearable @change="loadRecords" style="width:100px">
          <el-option label="1抽" :value="1" />
          <el-option label="1.5抽" :value="1.5" />
          <el-option label="2抽" :value="2" />
        </el-select>
      </div>

      <el-table table-layout="auto" :data="records" stripe>
        <el-table-column label="成員" min-width="70">
          <template #default="{ row }">
            <span :style="{ color: GROUP_COLORS[row.group], fontWeight: 500 }">{{ row.member_name }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="event_type" label="類型" min-width="55" />
        <el-table-column label="場地" min-width="160">
          <template #default="{ row }">
            <span style="white-space:nowrap">{{ row.venue || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="event_date" label="日期" min-width="90" />
        <el-table-column prop="session" label="部數" min-width="60" />
        <el-table-column label="單曲" min-width="220">
          <template #default="{ row }">{{ formatSingle(row.single_name) }}</template>
        </el-table-column>
        <el-table-column label="抽次" min-width="55">
          <template #default="{ row }">{{ row.lottery_round > 0 ? row.lottery_round + '抽' : '—' }}</template>
        </el-table-column>
        <el-table-column prop="applied_count" label="應募" min-width="50" />
        <el-table-column prop="won_count" label="中選" min-width="50" align="right" />
        <el-table-column label="中選率" min-width="70" align="right">
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
    </div>

    </template>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getFullRecords, getFullOverallStats, getFullStatsByMember, getFullStatsBySingle } from '../api/index'
import { sortMembersByGroupAndGen } from '../utils/members'
import EmptyState from '../components/EmptyState.vue'
import ErrorState from '../components/ErrorState.vue'

const GROUP_COLORS = { nogizaka46: '#9333ea', sakurazaka46: '#ec4899', hinatazaka46: '#0ea5e9' }

const memberList = ref([])
const venueList  = ref([])
const singleList = ref([])

const records  = ref([])
const total    = ref(0)
const page     = ref(1)
const pageSize = 50

const filterGroup  = ref('')
const filterMember = ref('')
const filterType   = ref('')
const filterVenue  = ref('')
const filterSingle = ref(null)
const filterRound  = ref(null)

// 這裡不用 stores/data.js 的 dataStore（那個追的是個握 getStats，跟全握是不同資料來源），
// 空/錯誤狀態就用這個頁面自己抓到的結果判斷，跟 RecordsView.vue 同一套 loaded/loadFailed/isEmpty 模式
const loaded     = ref(false)
const loadFailed = ref(false)
const isEmpty = computed(() =>
  loaded.value && total.value === 0 &&
  !filterGroup.value && !filterMember.value && !filterType.value && !filterVenue.value && !filterSingle.value && !filterRound.value
)

async function reloadFilterLists() {
  const groupParam = filterGroup.value ? { group: filterGroup.value } : {}
  const [statsRes, memberRes, singleRes] = await Promise.all([
    getFullOverallStats(),
    getFullStatsByMember(groupParam),
    getFullStatsBySingle(groupParam),
  ])
  const memberStats = memberRes.data ?? []
  const nameGroupMap = new Map()
  memberStats.forEach(m => m.member_name.split('・').forEach(n => { n = n.trim(); if (n) nameGroupMap.set(n, m.group || '') }))
  memberList.value = sortMembersByGroupAndGen([...nameGroupMap.entries()].map(([name, group]) => ({ name, group })))
  venueList.value = [...new Set((statsRes.data.by_type ?? []).map(r => r.venue).filter(v => v))]
  const GROUP_ORDER = { nogizaka46: 0, sakurazaka46: 1, hinatazaka46: 2 }
  singleList.value = (singleRes.data ?? []).sort((a, b) => {
    const gd = (GROUP_ORDER[a.group] ?? 9) - (GROUP_ORDER[b.group] ?? 9)
    if (gd !== 0) return gd
    const rd_a = a.release_date || '', rd_b = b.release_date || ''
    if (rd_a !== rd_b) {
      if (!rd_a) return 1
      if (!rd_b) return -1
      return rd_a.localeCompare(rd_b)
    }
    if (a.single_number !== b.single_number) {
      if (a.single_number === 0) return 1
      if (b.single_number === 0) return -1
      return a.single_number - b.single_number
    }
    return (a.single_name ?? '').localeCompare(b.single_name ?? '', 'ja')
  })
}

async function onGroupChange() {
  filterMember.value = ''
  filterSingle.value = null
  await reloadFilterLists()
  await loadRecords()
}

onMounted(async () => {
  try {
    await reloadFilterLists()
    await loadRecords()
  } catch {
    loadFailed.value = true
  } finally {
    loaded.value = true
  }
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
  if (filterGroup.value)  params.group  = filterGroup.value
  if (filterMember.value) {
    // 成員下拉的 value 是 "group:member_name" 組合字串，同樣的道理見下面單曲那段的說明，
    // 見 CLAUDE.md #126
    const [mGroup, mName] = filterMember.value.split(':')
    params.group  = mGroup
    params.member = mName
  }
  if (filterType.value)   params.event_type = filterType.value
  if (filterVenue.value)  params.venue = filterVenue.value
  if (filterSingle.value !== null) {
    // 單曲下拉的 value 是 "group:single_number" 組合字串（不是純數字），因為三個團體各自從 1 開始
    // 編號、號碼範圍會重疊——只送單曲號不夠，一定要一起帶團體，不然 GetFullRecords 的
    // single_number 篩選會跨團體撈到同號碼的其他團體資料，見 CLAUDE.md #125（同一個問題
    // RecordsView.vue 已經在 #113 修過）
    const [sGroup, sNum] = filterSingle.value.split(':')
    params.group = sGroup
    params.single_number = sNum
  }
  if (filterRound.value  !== null) params.lottery_round = filterRound.value
  const res = await getFullRecords(params)
  records.value = res.data.data ?? []
  total.value   = res.data.total ?? 0
}

function rateClass(rate) {
  if (rate >= 80) return 'rate high'
  if (rate >= 40) return 'rate mid'
  return 'rate low'
}

// 這個列表欄寬有限，單曲欄只顯示「N單」不帶書名號標題（跟個握紀錄等其他頁面不同，
// 那些欄位夠寬會保留完整標題）——正則後面加 .* 把標題整段吃掉一起換成短格式
function formatSingle(name) {
  if (!name) return ''
  return name
    .replace(/(\d+)(?:st|nd|rd|th)シングル.*/, (_, n) => `${n}單`)
    .replace(/(\d+)(?:st|nd|rd|th)アルバム.*/, (_, n) => `${n}專`)
}
</script>

<style scoped>
:deep(.el-table .cell) { white-space: nowrap; }
.page { background: #f5f7fa; min-height: 100vh; }
.card {
  background: white;
  border-radius: 10px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.07);
  border: 1px solid #e5e7eb;
}
html.dark .card { background: #1e2030; border-color: #2e3450; box-shadow: 0 2px 12px rgba(0,0,0,0.4); }
.filters { display: flex; gap: 12px; margin-bottom: 16px; flex-wrap: wrap; }
.pagination { margin-top: 16px; display: flex; justify-content: flex-end; }
.rate { font-weight: bold; }
.rate.high { color: #52c41a; }
.rate.mid  { color: #faad14; }
.rate.low  { color: #ff4d4f; }
</style>
