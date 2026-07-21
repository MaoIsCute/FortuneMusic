<template>
  <div class="page">
    <h1 class="page-title">🎁 商品抽選（生寫／海報）紀錄</h1>
    <p class="page-subtitle">顯示你自己同步過的商品抽選申請紀錄，這類獎品目前無法從來源網站取得中選結果，僅記錄申請口數</p>

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
        <el-select v-model="filterMember" placeholder="成員" clearable style="width:120px">
          <el-option v-for="m in memberOptions" :key="`${m.group}:${m.name}`"
            :label="m.name" :value="`${m.group}:${m.name}`">
            <span :style="{ color: GROUP_COLORS[m.group], fontWeight: 500 }">{{ m.name }}</span>
          </el-option>
        </el-select>
        <el-select v-model="filterPrize" placeholder="獎品" clearable style="width:160px">
          <el-option v-for="code in prizeOptions" :key="code" :label="formatPrizeName(code)" :value="code" />
        </el-select>
      </div>

      <div v-if="filteredRows.length === 0 && !loading" class="empty">尚無商品抽選紀錄</div>
      <el-table v-else v-loading="loading" table-layout="auto" :data="filteredRows" stripe>
        <el-table-column label="團體" min-width="80">
          <template #default="{ row }">
            <span :style="{ color: GROUP_COLORS[row.group], fontWeight: 500 }">{{ groupLabel(row.group) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="單曲" min-width="70" sortable sort-by="single_number">
          <template #default="{ row }">{{ row.single_number }}單</template>
        </el-table-column>
        <el-table-column label="獎品" min-width="140" sortable sort-by="prize_code">
          <template #default="{ row }">{{ formatPrizeName(row.prize_code) }}</template>
        </el-table-column>
        <el-table-column label="成員" min-width="100" sortable sort-by="member_name">
          <template #default="{ row }">
            <span :style="{ color: GROUP_COLORS[row.group] }">{{ row.member_name }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="unit_count" label="口數" min-width="70" sortable>
          <template #default="{ row }">{{ row.unit_count }} 口</template>
        </el-table-column>
        <el-table-column label="結果" width="110" align="center">
          <template #default="{ row }">
            <el-select v-if="editingId === row.id" v-model="editValue" size="small" style="width:90px">
              <el-option label="抽選中" value="pending" />
              <el-option label="中選" value="won" />
              <el-option label="落選" value="lost" />
            </el-select>
            <span v-else :class="resultClass(row.won_status)">{{ resultLabel(row.won_status) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="" width="80" align="center">
          <template #default="{ row }">
            <el-button v-if="editingId === row.id" size="small" type="primary" :loading="savingId === row.id" @click="confirmResult(row)">確定</el-button>
            <el-button v-else link size="small" @click="startEdit(row)">修改</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getPrizes, updatePrizeResult } from '../api/index'

const GROUP_COLORS = { nogizaka46: '#9333ea', sakurazaka46: '#ec4899', hinatazaka46: '#0ea5e9' }
const GROUP_LABELS = { nogizaka46: '乃木坂46', sakurazaka46: '櫻坂46', hinatazaka46: '日向坂46' }
function groupLabel(g) { return GROUP_LABELS[g] || g || '—' }

// prize_code 是後端存的原始值，中文名稱只在這裡（顯示層）轉換，跟單曲名稱用 formatSingle() 同一套做法；
// 對不到的（以後網站新增其他 p_xxx 獎品）直接 fallback 顯示原始代碼，不會讓新獎品消失不見
const PRIZE_NAMES = {
  p_sign_photo:         '密藏生寫',
  p_sign_solo_poster:   '個人簽名海報',
  p_premium_sign_photo: '特別簽名生寫',
}
function formatPrizeName(code) {
  return PRIZE_NAMES[code] || code
}

// 抓到的 applied_count 是網站顯示的「枚數」，不是口數——密藏生寫、個人簽名海報都是 2 枚湊 1 口，
// 換算才是實際申請口數；跟簽名會 applied_count 以 3 為一組、前端才換算成口數同一套做法（見
// CLAUDE.md 簽名會口數顯示邏輯）。特別簽名生寫目前沒有另外確認換算比例，先當 1 枚 1 口處理，
// 之後如果確認不是 1:1 要再補進這個對照表
const PRIZE_SHEETS_PER_UNIT = {
  p_sign_photo:       2,
  p_sign_solo_poster: 2,
}
function unitCount(code, count) {
  const perUnit = PRIZE_SHEETS_PER_UNIT[code] || 1
  return Math.round(count / perUnit)
}

const allRows = ref([])
const loading = ref(false)

const filterGroup  = ref('')
const filterMember = ref('')
const filterPrize  = ref('')

const memberOptions = computed(() => {
  const map = new Map()
  allRows.value.forEach(r => {
    if (filterGroup.value && r.group !== filterGroup.value) return
    r.member_name.split('・').forEach(n => {
      n = n.trim()
      if (!n) return
      const key = `${r.group}:${n}`
      if (!map.has(key)) map.set(key, { group: r.group, name: n })
    })
  })
  return [...map.values()].sort((a, b) => a.name.localeCompare(b.name, 'ja'))
})

const prizeOptions = computed(() => {
  const set = new Set()
  allRows.value.forEach(r => {
    if (filterGroup.value && r.group !== filterGroup.value) return
    set.add(r.prize_code)
  })
  return [...set]
})

const filteredRows = computed(() => allRows.value.filter(r => {
  if (filterGroup.value && r.group !== filterGroup.value) return false
  if (filterMember.value) {
    const [mGroup, mName] = filterMember.value.split(':')
    if (r.group !== mGroup || !r.member_name.split('・').map(n => n.trim()).includes(mName)) return false
  }
  if (filterPrize.value && r.prize_code !== filterPrize.value) return false
  return true
}))

function onGroupChange() {
  if (filterMember.value && !memberOptions.value.some(m => `${m.group}:${m.name}` === filterMember.value)) {
    filterMember.value = ''
  }
}

async function load() {
  loading.value = true
  try {
    const res = await getPrizes()
    allRows.value = (res.data.data ?? []).map(r => ({ ...r, unit_count: unitCount(r.prize_code, r.applied_count) }))
  } finally {
    loading.value = false
  }
}

// 商品抽選的中選結果來源網站讀不到，只能讓使用者自己標記；未標記過（won_status 空字串）
// 顯示「抽選中」，點「修改」才切成可選的下拉，選完按「確定」才真的送出更新資料庫
const editingId = ref(null)
const editValue = ref('')
const savingId  = ref(null)

function resultLabel(status) {
  if (status === 'won')  return '中選'
  if (status === 'lost') return '落選'
  return '抽選中'
}
function resultClass(status) {
  if (status === 'won')  return 'tag-won'
  if (status === 'lost') return 'tag-lost'
  return 'tag-pending'
}

function startEdit(row) {
  editingId.value = row.id
  // el-select 把綁定值的空字串當成「沒選」處理，只會顯示 placeholder，不會顯示
  // value="" 那個選項的文字，所以「抽選中」這個狀態內部改用 'pending' 代稱，
  // 送出更新時再換回後端認得的空字串（won_status 欄位維持不動）
  editValue.value = row.won_status || 'pending'
}

async function confirmResult(row) {
  savingId.value = row.id
  try {
    const status = editValue.value === 'pending' ? '' : editValue.value
    await updatePrizeResult(row.id, status)
    const target = allRows.value.find(r => r.id === row.id)
    if (target) target.won_status = status
    editingId.value = null
  } finally {
    savingId.value = null
  }
}

onMounted(load)
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
.tag-won     { color: #52c41a; font-weight: bold; }
.tag-lost    { color: #ff4d4f; font-weight: bold; }
.tag-pending { color: #999; }
</style>
