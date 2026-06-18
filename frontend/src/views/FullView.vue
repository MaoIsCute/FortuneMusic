<template>
  <div class="page">
    <h1 class="page-title">📋 全握紀錄</h1>

    <div class="card">
      <div class="filters">
        <el-select v-model="filterGroup" placeholder="團體" clearable style="width:120px" @change="onGroupChange">
          <el-option label="乃木坂46" value="nogizaka46" />
          <el-option label="櫻坂46" value="sakurazaka46" />
          <el-option label="日向坂46" value="hinatazaka46" />
        </el-select>
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
        <el-table-column prop="member_name" label="成員" width="120" />
        <el-table-column prop="event_type" label="類型" width="70" />
        <el-table-column label="場地" width="160">
          <template #default="{ row }">
            <span style="white-space:nowrap">{{ row.venue || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="event_date" label="日期" width="100" />
        <el-table-column prop="session" label="部數" width="70" />
        <el-table-column label="單曲" width="70">
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
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getFullRecords, getFullOverallStats, getFullStatsByMember, getFullStatsBySingle } from '../api/index'
import { sortMembersByGen } from '../utils/members'

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

async function reloadFilterLists() {
  const groupParam = filterGroup.value ? { group: filterGroup.value } : {}
  const [statsRes, memberRes, singleRes] = await Promise.all([
    getFullOverallStats(),
    getFullStatsByMember(groupParam),
    getFullStatsBySingle(groupParam),
  ])
  const memberStats = memberRes.data ?? []
  const allNames = new Set()
  memberStats.forEach(m => m.member_name.split('・').forEach(n => n.trim() && allNames.add(n.trim())))
  memberList.value = sortMembersByGen([...allNames])
  venueList.value = [...new Set((statsRes.data.by_type ?? []).map(r => r.venue).filter(v => v))]
  singleList.value = singleRes.data ?? []
}

async function onGroupChange() {
  filterMember.value = ''
  filterSingle.value = null
  await reloadFilterLists()
  await loadRecords()
}

onMounted(async () => {
  await reloadFilterLists()
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
  if (filterGroup.value)  params.group  = filterGroup.value
  if (filterMember.value) params.member = filterMember.value
  if (filterType.value)   params.event_type = filterType.value
  if (filterVenue.value)  params.venue = filterVenue.value
  if (filterSingle.value !== null) params.single_number = filterSingle.value
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

function formatSingle(name) {
  if (!name) return ''
  return name
    .replace(/(\d+)(?:st|nd|rd|th)シングル/, (_, n) => `${n}單`)
    .replace(/(\d+)(?:st|nd|rd|th)アルバム/, (_, n) => `${n}專`)
}
</script>

<style scoped>
.page { background: #f5f7fa; min-height: 100vh; }
.card {
  background: white;
  border-radius: 10px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.07);
  border: 1px solid #e5e7eb;
}
.filters { display: flex; gap: 12px; margin-bottom: 16px; flex-wrap: wrap; }
.pagination { margin-top: 16px; display: flex; justify-content: flex-end; }
.rate { font-weight: bold; }
.rate.high { color: #52c41a; }
.rate.mid  { color: #faad14; }
.rate.low  { color: #ff4d4f; }
</style>
