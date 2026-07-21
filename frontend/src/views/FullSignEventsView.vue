<template>
  <div class="page">
    <h1 class="page-title">✍️ 簽名會紀錄</h1>
    <p class="page-subtitle">顯示你自己同步過的簽名會紀錄</p>

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
        <el-select v-model="filterMember" placeholder="選擇成員" clearable style="width:150px" @change="load">
          <el-option v-for="m in filteredMemberOptions" :key="`${m.group}:${m.name}`" :label="m.name" :value="`${m.group}:${m.name}`">
            <span :style="{ color: GROUP_COLORS[m.group] }">{{ m.name }}</span>
          </el-option>
        </el-select>
        <el-select v-model="filterSingle" placeholder="單曲" clearable style="width:130px" @change="load">
          <el-option v-for="s in filteredSingleOptions" :key="`${s.group}:${s.single_number}`"
            :label="formatSingle(s.single_name) || `${s.single_number}單`" :value="`${s.group}:${s.single_number}`">
            <span :style="{ color: GROUP_COLORS[s.group] }">{{ formatSingle(s.single_name) || `${s.single_number}單` }}</span>
          </el-option>
        </el-select>
      </div>

      <div v-if="rows.length === 0 && !loading" class="empty">尚無簽名會紀錄</div>
      <el-table v-else v-loading="loading" table-layout="auto" :data="rows" stripe>
        <el-table-column label="團體" min-width="80">
          <template #default="{ row }">
            <span :style="{ color: GROUP_COLORS[row.group], fontWeight: 500 }">{{ groupLabel(row.group) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="成員" min-width="65">
          <template #default="{ row }">
            <span :style="{ color: GROUP_COLORS[row.group] }">{{ row.member_name }}</span>
          </template>
        </el-table-column>
        <el-table-column label="單曲" min-width="240">
          <template #default="{ row }">{{ formatSingle(row.single_name) || `${row.single_number}單` }}</template>
        </el-table-column>
        <el-table-column label="場地" min-width="160">
          <template #default="{ row }">{{ row.venue || '—' }}</template>
        </el-table-column>
        <el-table-column prop="event_date" label="日期" min-width="90" />
        <el-table-column label="抽次" min-width="60" align="center">
          <template #default="{ row }">{{ row.lottery_round > 0 ? row.lottery_round + '抽' : '—' }}</template>
        </el-table-column>
        <el-table-column label="應募" min-width="70" align="right">
          <template #default="{ row }">{{ Math.round(row.applied_count / 3) }} 口</template>
        </el-table-column>
        <el-table-column label="結果" min-width="70" align="center">
          <template #default="{ row }">
            <span :class="row.won_count > 0 ? 'tag-won' : 'tag-lost'">
              {{ row.won_count > 0 ? '中選' : '落選' }}
            </span>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          v-if="total > pageSize"
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
import { getSignEvents } from '../api/index'
import { sortMembersByGroupAndGen } from '../utils/members'

const GROUP_COLORS = { nogizaka46: '#9333ea', sakurazaka46: '#ec4899', hinatazaka46: '#0ea5e9' }
const GROUP_LABELS = { nogizaka46: '乃木坂46', sakurazaka46: '櫻坂46', hinatazaka46: '日向坂46' }
function groupLabel(g) { return GROUP_LABELS[g] || g || '—' }

function formatSingle(name) {
  if (!name) return ''
  return name
    .replace(/(\d+)(?:st|nd|rd|th)シングル/, (_, n) => `${n}單`)
    .replace(/(\d+)(?:st|nd|rd|th)アルバム/, (_, n) => `${n}專`)
}

const rows     = ref([])
const total    = ref(0)
const page     = ref(1)
const pageSize = 50
const loading  = ref(false)

const filterGroup  = ref('')
const filterMember = ref('')
const filterSingle = ref('')

const memberList = ref([])
const singleList = ref([])
const GROUP_ORDER = { nogizaka46: 0, sakurazaka46: 1, hinatazaka46: 2 }

const filteredMemberOptions = computed(() =>
  filterGroup.value ? memberList.value.filter(m => m.group === filterGroup.value) : memberList.value
)
const filteredSingleOptions = computed(() =>
  filterGroup.value ? singleList.value.filter(s => s.group === filterGroup.value) : singleList.value
)

// 簽名會沒有像全握那樣的 by-member/by-single 聚合統計端點，成員/單曲下拉的選項直接從
// GetSignEvents 撈一批（page_size 用後端上限 100）湊出來，跟實際列表分頁各自獨立抓取。
// 簽名會資料量目前遠低於 100 筆，這個上限暫時夠用；如果之後單一使用者的簽名會紀錄超過
// 100 筆，下拉選項會少列出超過上限的那些單曲/成員（已知限制，屆時要再改成專用的聚合端點）
async function reloadFilterLists() {
  const res  = await getSignEvents({ page: 1, page_size: 100 })
  const list = res.data.data ?? []

  const nameGroupMap = new Map()
  const singleMap    = new Map()
  list.forEach(r => {
    r.member_name.split('・').forEach(n => { n = n.trim(); if (n) nameGroupMap.set(n, r.group || '') })
    const key = `${r.group}:${r.single_number}`
    if (!singleMap.has(key)) singleMap.set(key, { group: r.group, single_number: r.single_number, single_name: r.single_name })
  })

  memberList.value = sortMembersByGroupAndGen([...nameGroupMap.entries()].map(([name, group]) => ({ name, group })))
  singleList.value = [...singleMap.values()].sort((a, b) => {
    const gd = (GROUP_ORDER[a.group] ?? 9) - (GROUP_ORDER[b.group] ?? 9)
    return gd !== 0 ? gd : a.single_number - b.single_number
  })
}

function onGroupChange() {
  if (filterMember.value && !filteredMemberOptions.value.some(m => `${m.group}:${m.name}` === filterMember.value)) {
    filterMember.value = ''
  }
  if (filterSingle.value && !filteredSingleOptions.value.some(s => `${s.group}:${s.single_number}` === filterSingle.value)) {
    filterSingle.value = ''
  }
  load()
}

async function load() {
  page.value = 1
  await fetchPage()
}

async function fetchPage() {
  loading.value = true
  try {
    const params = { page: page.value, page_size: pageSize }
    if (filterGroup.value) params.group = filterGroup.value
    if (filterMember.value) {
      // 成員下拉的 value 是 "group:member_name" 組合字串，單曲下拉同理，避免三個團體號碼/
      // 名稱重疊時跨團體撈錯資料，跟 FullView.vue 同一套做法（見 CLAUDE.md #125/#126）
      const [mGroup, mName] = filterMember.value.split(':')
      params.group  = mGroup
      params.member = mName
    }
    if (filterSingle.value) {
      const [sGroup, sNum] = filterSingle.value.split(':')
      params.group         = sGroup
      params.single_number = sNum
    }
    const res = await getSignEvents(params)
    rows.value  = res.data.data ?? []
    total.value = res.data.total ?? 0
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await reloadFilterLists()
  await fetchPage()
})
</script>

<style scoped>
.page { background: #f5f7fa; min-height: 100vh; }
:deep(.el-table .cell) { white-space: nowrap; }
.page-title    { margin-bottom: 4px; }
.page-subtitle { color: #888; font-size: 13px; margin: 0 0 20px; }
html.dark .page-subtitle { color: #9aa3b5; }
.card {
  background: white;
  border-radius: 10px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.07);
  border: 1px solid #e5e7eb;
}
html.dark .card { background: #1e2030; border-color: #2e3450; box-shadow: 0 2px 12px rgba(0,0,0,0.4); }
.filters { display: flex; gap: 12px; margin-bottom: 16px; flex-wrap: wrap; }
.empty { color: #999; text-align: center; padding: 32px 0; }
.pagination { margin-top: 16px; display: flex; justify-content: flex-end; }
.tag-won  { color: #52c41a; font-weight: bold; }
.tag-lost { color: #ff4d4f; font-weight: bold; }
</style>
