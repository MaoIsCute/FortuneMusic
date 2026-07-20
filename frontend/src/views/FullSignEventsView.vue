<template>
  <div class="page">
    <h1 class="page-title">✍️ 簽名會紀錄</h1>
    <p class="page-subtitle">顯示你自己同步過的簽名會紀錄</p>

    <div class="card">
      <div class="filters">
        <el-select v-model="filterGroup" placeholder="團體" clearable style="width:120px" @change="load">
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
        <el-input v-model="filterMember" placeholder="成員名稱" clearable style="width:150px"
          @change="load" @clear="load" />
        <el-input v-model="filterSingle" placeholder="單曲號" clearable style="width:100px" type="number"
          @change="load" @clear="load" />
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
import { ref, onMounted } from 'vue'
import { getSignEvents } from '../api/index'

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

async function load() {
  page.value = 1
  await fetchPage()
}

async function fetchPage() {
  loading.value = true
  try {
    const params = { page: page.value, page_size: pageSize }
    if (filterGroup.value)  params.group         = filterGroup.value
    if (filterMember.value) params.member        = filterMember.value.trim()
    if (filterSingle.value) params.single_number = filterSingle.value
    const res = await getSignEvents(params)
    rows.value  = res.data.data ?? []
    total.value = res.data.total ?? 0
  } finally {
    loading.value = false
  }
}

onMounted(fetchPage)
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
