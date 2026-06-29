<template>
  <div class="page">
    <h1 class="page-title">📋 抽選紀錄</h1>
    <template v-if="loaded">
    <ErrorState v-if="loadFailed" />
    <EmptyState v-else-if="isEmpty" />
    <template v-else>
    <div class="filters">
      <el-select v-model="filterGroup" placeholder="團體" clearable style="width:120px" @change="onGroupChange">
        <el-option label="乃木坂46" value="nogizaka46" />
        <el-option label="櫻坂46" value="sakurazaka46" />
        <el-option label="日向坂46" value="hinatazaka46" />
      </el-select>
      <el-select v-model="filterMember" placeholder="選擇成員" clearable @change="loadRecords">
        <el-option v-for="m in memberList" :key="m" :label="m" :value="m" />
      </el-select>
      <el-select v-model="filterSingle" placeholder="選擇單曲" clearable @change="loadRecords">
        <el-option v-for="s in singleList" :key="s" :label="formatSingle(s)" :value="s" />
      </el-select>
      <el-select v-model="filterRound" placeholder="選擇次數" clearable @change="loadRecords">
        <el-option v-for="r in roundList" :key="r" :label="formatRound(r)" :value="r" />
      </el-select>
    </div>
    <el-table :data="records" stripe>
      <el-table-column prop="member_name" label="成員" />
      <el-table-column label="單曲">
        <template #default="{ row }">{{ formatSingle(row.single_name) || row.event_name }}</template>
      </el-table-column>
      <el-table-column label="次數" width="80">
        <template #default="{ row }">{{ formatRound(row.lottery_round) }}</template>
      </el-table-column>
      <el-table-column prop="event_date" label="日期" width="110" />
      <el-table-column prop="session" label="部數" width="90" />
      <el-table-column prop="applied_count" label="應募" width="70" />
      <el-table-column prop="won_count" label="中選" width="70" />
      <el-table-column label="中選率" width="90">
        <template #default="{ row }">
          <span :class="rateClass(row)">{{ calcRate(row) }}%</span>
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
    </template>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getRecords, getStatsByMember, getDetailStats } from '../api/index'
import { sortMembersByGen } from '../utils/members'
import { useDataStore } from '../stores/data'
import EmptyState from '../components/EmptyState.vue'
import ErrorState from '../components/ErrorState.vue'

const records  = ref([])
const total    = ref(0)
const page     = ref(1)
const pageSize = 20

const filterGroup  = ref('')
const filterMember = ref('')
const filterSingle = ref('')
const filterRound  = ref('')

const memberList = ref([])
const singleList = ref([])
const roundList  = ref([])
const loaded     = ref(false)
const loadFailed = ref(false)
const dataStore  = useDataStore()

const isEmpty = computed(() =>
  loaded.value && total.value === 0 && !filterMember.value && !filterSingle.value && !filterRound.value
)

async function reloadFilterLists() {
  const groupParam = filterGroup.value ? { group: filterGroup.value } : {}
  const [membersRes, detailRes] = await Promise.all([
    getStatsByMember(groupParam),
    getDetailStats(groupParam),
  ])
  memberList.value = sortMembersByGen((membersRes.data ?? []).map(m => m.member_name))
  const rows = detailRes.data ?? []
  // 單曲以 single_number 去重（避免同一張單曲有新舊兩種名稱時出現重複選項），
  // 名稱優先取非 タイトル未定/非空的版本；專輯（single_number=0）沒有可靠編號，改用名稱本身當 key
  const singleMap = new Map()
  for (const r of rows) {
    if (!r.single_name) continue
    if (r.single_number > 0) {
      const existing = singleMap.get(r.single_number)
      if (!existing || existing.includes('タイトル未定') || existing === '')
        singleMap.set(r.single_number, r.single_name)
    } else {
      singleMap.set(`a:${r.single_name}`, r.single_name)
    }
  }
  singleList.value = [...singleMap.values()].sort()
  roundList.value  = [...new Set(rows.map(r => r.lottery_round).filter(Boolean))].sort((a, b) => a - b)
}

async function onGroupChange() {
  filterMember.value = ''
  filterSingle.value = ''
  await reloadFilterLists()
  await loadRecords()
}

onMounted(async () => {
  if (dataStore.hasData === false) {
    loaded.value = true
    return
  }
  try {
    await reloadFilterLists()
    await loadRecords()
  } catch {
    loadFailed.value = true
  } finally {
    loaded.value = true
  }
})

async function loadRecords() {
  page.value = 1
  await fetchPage()
}

async function fetchPage() {
  const params = { page: page.value, page_size: pageSize }
  if (filterGroup.value)  params.group  = filterGroup.value
  if (filterMember.value) params.member = filterMember.value
  if (filterSingle.value) params.single = filterSingle.value
  if (filterRound.value)  params.round  = filterRound.value
  const res = await getRecords(params)
  records.value = res.data.data ?? []
  total.value   = res.data.total ?? 0
}

function calcRate(row) {
  if (!row.applied_count) return '0.0'
  return (row.won_count / row.applied_count * 100).toFixed(1)
}

function rateClass(row) {
  const r = row.applied_count ? row.won_count / row.applied_count * 100 : 0
  if (r >= 80) return 'rate high'
  if (r >= 40) return 'rate mid'
  return 'rate low'
}

function formatSingle(name) {
  if (!name) return ''
  return name
    .replace(/(\d+)(?:st|nd|rd|th)シングル/, (_, n) => `${n}單`)
    .replace(/(\d+)(?:st|nd|rd|th)アルバム/, (_, n) => `${n}專`)
    .replace(/^アルバム/, '專輯')
}

function formatRound(round) {
  return round ? `${round}抽` : ''
}
</script>

<style scoped>
.filters {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}
.pagination { margin-top: 20px; display: flex; justify-content: flex-end; }
.rate { font-weight: bold; }
.rate.high { color: #52c41a; }
.rate.mid  { color: #faad14; }
.rate.low  { color: #ff4d4f; }
</style>
