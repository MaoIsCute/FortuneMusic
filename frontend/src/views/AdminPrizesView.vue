<template>
  <div class="page">
    <h1 class="page-title">🔧 商品抽選紀錄</h1>

    <div class="card">
      <div class="card-header">
        <span class="card-title">商品抽選紀錄</span>
        <el-button size="small" @click="loadPrizes">重新整理</el-button>
      </div>
      <div class="filter-row">
        <el-select v-model="filter.userId" placeholder="篩選使用者" clearable style="width:200px" @change="page=1;loadPrizes()">
          <el-option v-for="u in users" :key="u.id" :label="`${u.name} (${u.email})`" :value="u.id" />
        </el-select>
        <el-select v-model="filter.group" placeholder="團體" clearable style="width:120px" @change="onGroupChange">
          <el-option label="乃木坂46" value="nogizaka46" />
          <el-option label="櫻坂46" value="sakurazaka46" />
          <el-option label="日向坂46" value="hinatazaka46" />
        </el-select>
        <el-select v-model="filter.member" placeholder="成員" clearable style="width:130px" @change="page=1;loadPrizes()">
          <el-option v-for="m in filteredMemberOptions" :key="`${m.group}:${m.name}`" :label="m.name" :value="`${m.group}:${m.name}`" />
        </el-select>
        <el-select v-model="filter.prizeCode" placeholder="獎品" clearable style="width:160px" @change="page=1;loadPrizes()">
          <el-option v-for="code in Object.keys(PRIZE_NAMES)" :key="code" :label="formatPrizeName(code)" :value="code" />
        </el-select>
      </div>
      <div v-if="rows.length === 0" class="empty">尚無商品抽選紀錄</div>
      <el-table table-layout="auto" v-else :data="rows" stripe>
        <el-table-column label="使用者" min-width="140">
          <template #default="{ row }">{{ row.user_name }}<br/><span class="sub-text">{{ row.user_email }}</span></template>
        </el-table-column>
        <el-table-column label="團體" min-width="80">
          <template #default="{ row }">{{ groupLabel(row.group) }}</template>
        </el-table-column>
        <el-table-column label="單曲" min-width="70">
          <template #default="{ row }">{{ row.single_number }}單</template>
        </el-table-column>
        <el-table-column label="獎品" min-width="140">
          <template #default="{ row }">{{ formatPrizeName(row.prize_code) }}</template>
        </el-table-column>
        <el-table-column prop="member_name" label="成員" min-width="100" />
        <el-table-column label="口數" min-width="70" align="right">
          <template #default="{ row }">{{ unitCount(row.prize_code, row.applied_count) }} 口</template>
        </el-table-column>
        <el-table-column label="結果" width="80" align="center">
          <template #default="{ row }">
            <span :class="resultClass(row.won_status)">{{ resultLabel(row.won_status) }}</span>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        v-if="total > pageSize"
        v-model:current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="prev, pager, next"
        style="margin-top:12px;justify-content:flex-end;display:flex"
        @current-change="loadPrizes"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getAdminUsers, getAdminPrizes } from '../api/index'

const GROUP_LABELS = { nogizaka46: '乃木坂46', sakurazaka46: '櫻坂46', hinatazaka46: '日向坂46' }
function groupLabel(g) { return GROUP_LABELS[g] || g || '—' }

// prize_code 是後端存的原始值，中文名稱只在這裡（顯示層）轉換，跟 FullPrizesView.vue 同一份對照表
const PRIZE_NAMES = {
  p_sign_photo:         '密藏生寫',
  p_sign_solo_poster:   '個人簽名海報',
  p_premium_sign_photo: '特別簽名生寫',
}
function formatPrizeName(code) {
  return PRIZE_NAMES[code] || code
}

// applied_count 是枚數不是口數，密藏生寫／個人簽名海報都是 2 枚湊 1 口，跟 FullPrizesView.vue
// 同一份換算表（見那邊的說明）
const PRIZE_SHEETS_PER_UNIT = {
  p_sign_photo:       2,
  p_sign_solo_poster: 2,
}
function unitCount(code, count) {
  const perUnit = PRIZE_SHEETS_PER_UNIT[code] || 1
  return Math.round(count / perUnit)
}

// 中選結果是使用者自己在個人版 FullPrizesView.vue 手動標記的，管理者這頁只唯讀顯示，不提供編輯
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

const users    = ref([])
const rows     = ref([])
const total    = ref(0)
const page     = ref(1)
const pageSize = 50
const filter   = ref({ userId: null, group: '', member: '', prizeCode: '' })

const memberOptions = ref([])
const filteredMemberOptions = computed(() =>
  filter.value.group ? memberOptions.value.filter(m => m.group === filter.value.group) : memberOptions.value
)

async function loadUsers() {
  try {
    const res = await getAdminUsers()
    users.value = res.data ?? []
  } catch {}
}

// 管理者這頁看的是所有使用者的商品抽選紀錄，成員下拉選項直接從 GetAdminPrizes 撈一批
// （page_size 用後端上限 100）湊出來，跟個人版 FullPrizesView.vue 同一套做法
async function reloadFilterLists() {
  const res  = await getAdminPrizes({ page: 1, page_size: 100 })
  const list = res.data.data ?? []
  const map  = new Map()
  list.forEach(r => {
    r.member_name.split('・').forEach(n => {
      n = n.trim()
      if (!n) return
      const key = `${r.group}:${n}`
      if (!map.has(key)) map.set(key, { group: r.group, name: n })
    })
  })
  memberOptions.value = [...map.values()].sort((a, b) => a.name.localeCompare(b.name, 'ja'))
}

function onGroupChange() {
  if (filter.value.member && !filteredMemberOptions.value.some(m => `${m.group}:${m.name}` === filter.value.member)) {
    filter.value.member = ''
  }
  page.value = 1
  loadPrizes()
}

async function loadPrizes() {
  try {
    const params = { page: page.value, page_size: pageSize }
    if (filter.value.userId)    params.user_id    = filter.value.userId
    if (filter.value.group)     params.group      = filter.value.group
    if (filter.value.member) {
      const [mGroup, mName] = filter.value.member.split(':')
      params.group  = mGroup
      params.member = mName
    }
    if (filter.value.prizeCode) params.prize_code = filter.value.prizeCode
    const res = await getAdminPrizes(params)
    rows.value  = res.data.data ?? []
    total.value = res.data.total ?? 0
  } catch {}
}

onMounted(() => {
  loadUsers()
  reloadFilterLists()
  loadPrizes()
})
</script>

<style scoped>
.page { background: #f5f7fa; min-height: 100vh; }
:deep(.el-table .cell) { white-space: nowrap; }
.card {
  border-radius: 10px;
  border: 1px solid #e5e7eb;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.07);
  background: white;
  padding: 16px 20px 20px;
}
.card-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.card-title { font-weight: 600; font-size: 14px; color: #111827; }
.empty { color: #999; text-align: center; padding: 32px 0; }
.sub-text { font-size: 11px; color: #999; }
html.dark .card       { background: #1e2030; border-color: #2e3450; box-shadow: 0 2px 12px rgba(0,0,0,0.4); }
html.dark .card-title { color: #e8eaf0; }
.filter-row { display: flex; flex-wrap: wrap; gap: 10px; margin-bottom: 14px; }
.tag-won     { color: #52c41a; font-weight: bold; }
.tag-lost    { color: #ff4d4f; font-weight: bold; }
.tag-pending { color: #999; }
</style>
