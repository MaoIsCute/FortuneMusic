<template>
  <div class="page">
    <h1 class="page-title">🔧 簽名會紀錄</h1>

    <div class="card">
      <div class="card-header">
        <span class="card-title">簽名會紀錄</span>
        <el-button size="small" @click="loadSignEvents">重新整理</el-button>
      </div>
      <div class="filter-row">
        <el-select v-model="signFilter.userId" placeholder="篩選使用者" clearable style="width:200px" @change="signPage=1;loadSignEvents()">
          <el-option v-for="u in users" :key="u.id" :label="`${u.name} (${u.email})`" :value="u.id" />
        </el-select>
        <el-select v-model="signFilter.group" placeholder="團體" clearable style="width:120px" @change="onGroupChange">
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
        <el-select v-model="signFilter.member" placeholder="成員" clearable style="width:130px" @change="signPage=1;loadSignEvents()">
          <el-option v-for="m in filteredMemberOptions" :key="`${m.group}:${m.name}`" :label="m.name" :value="`${m.group}:${m.name}`">
            <span :style="{ color: GROUP_COLORS[m.group], fontWeight: 500 }">{{ m.name }}</span>
          </el-option>
        </el-select>
        <el-select v-model="signFilter.singleNumber" placeholder="單曲" clearable style="width:120px" @change="signPage=1;loadSignEvents()">
          <el-option v-for="s in filteredSingleOptions" :key="`${s.group}:${s.single_number}`"
            :label="`${s.single_number}單`" :value="`${s.group}:${s.single_number}`">
            <span :style="{ color: GROUP_COLORS[s.group], fontWeight: 500 }">{{ s.single_number }}單</span>
          </el-option>
        </el-select>
      </div>
      <div v-if="signEvents.length === 0" class="empty">尚無簽名會紀錄</div>
      <el-table table-layout="auto" v-else :data="signEvents" stripe>
        <el-table-column label="使用者" min-width="140">
          <template #default="{ row }">{{ row.user_name }}<br/><span class="sub-text">{{ row.user_email }}</span></template>
        </el-table-column>
        <el-table-column prop="member_name" label="成員" min-width="65" />
        <el-table-column label="單曲" min-width="240">
          <template #default="{ row }">{{ formatSingle(row.single_name) || `${row.single_number}單` }}</template>
        </el-table-column>
        <el-table-column label="場地" min-width="160">
          <template #default="{ row }">{{ row.venue || '—' }}</template>
        </el-table-column>
        <el-table-column prop="event_date" label="日期" min-width="90" />
        <el-table-column label="抽次" min-width="70" align="center">
          <template #default="{ row }">{{ row.lottery_round > 0 ? row.lottery_round + '抽' : '—' }}</template>
        </el-table-column>
        <el-table-column label="應募" min-width="75" align="right">
          <template #default="{ row }">{{ Math.round(row.applied_count / 3) }} 口</template>
        </el-table-column>
        <el-table-column label="結果" min-width="75" align="center">
          <template #default="{ row }">
            <span :class="row.won_count > 0 ? 'tag-won' : 'tag-lost'">
              {{ row.won_count > 0 ? '中選' : '落選' }}
            </span>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        v-if="signTotal > signPageSize"
        v-model:current-page="signPage"
        :page-size="signPageSize"
        :total="signTotal"
        layout="prev, pager, next"
        style="margin-top:12px;justify-content:flex-end;display:flex"
        @current-change="loadSignEvents"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getAdminUsers, getAdminSignEvents } from '../api/index'

const GROUP_COLORS = { nogizaka46: '#9333ea', sakurazaka46: '#ec4899', hinatazaka46: '#0ea5e9' }

const users      = ref([])
const signEvents  = ref([])
const signTotal   = ref(0)
const signPage    = ref(1)
const signPageSize = 50
const signFilter  = ref({ userId: null, group: '', member: '', singleNumber: '' })

const memberOptions = ref([])
const singleOptions = ref([])

const filteredMemberOptions = computed(() =>
  signFilter.value.group ? memberOptions.value.filter(m => m.group === signFilter.value.group) : memberOptions.value
)
const filteredSingleOptions = computed(() =>
  signFilter.value.group ? singleOptions.value.filter(s => s.group === signFilter.value.group) : singleOptions.value
)

async function loadUsers() {
  try {
    const res = await getAdminUsers()
    users.value = res.data ?? []
  } catch {}
}

function formatSingle(name) {
  if (!name) return ''
  return name
    .replace(/(\d+)(?:st|nd|rd|th)シングル/, (_, n) => `${n}單`)
    .replace(/(\d+)(?:st|nd|rd|th)アルバム/, (_, n) => `${n}專`)
}

// 管理者這頁看的是所有使用者的簽名會紀錄，沒有專屬的 by-member/by-single 聚合端點，成員/單曲
// 下拉選項直接從 GetAdminSignEvents 撈一批（page_size 用後端上限 100）湊出來，跟個人版
// FullSignEventsView.vue 同一套做法；選項的 value 是 "group:member_name"/"group:single_number"
// 組合字串，避免三個團體號碼/名稱重疊時跨團體撈錯資料（見 CLAUDE.md #125/#126）
async function reloadFilterLists() {
  const res  = await getAdminSignEvents({ page: 1, page_size: 100 })
  const list = res.data.data ?? []

  const nameGroupMap = new Map()
  const singleMap    = new Map()
  list.forEach(r => {
    r.member_name.split('・').forEach(n => { n = n.trim(); if (n) nameGroupMap.set(n, r.group || '') })
    const key = `${r.group}:${r.single_number}`
    if (!singleMap.has(key)) singleMap.set(key, { group: r.group, single_number: r.single_number })
  })

  memberOptions.value = [...nameGroupMap.entries()]
    .map(([name, group]) => ({ name, group }))
    .sort((a, b) => a.name.localeCompare(b.name, 'ja'))
  singleOptions.value = [...singleMap.values()].sort((a, b) => a.single_number - b.single_number)
}

function onGroupChange() {
  if (signFilter.value.member && !filteredMemberOptions.value.some(m => `${m.group}:${m.name}` === signFilter.value.member)) {
    signFilter.value.member = ''
  }
  if (signFilter.value.singleNumber && !filteredSingleOptions.value.some(s => `${s.group}:${s.single_number}` === signFilter.value.singleNumber)) {
    signFilter.value.singleNumber = ''
  }
  signPage.value = 1
  loadSignEvents()
}

async function loadSignEvents() {
  try {
    const params = { page: signPage.value, page_size: signPageSize }
    if (signFilter.value.userId) params.user_id = signFilter.value.userId
    if (signFilter.value.group)  params.group   = signFilter.value.group
    if (signFilter.value.member) {
      const [mGroup, mName] = signFilter.value.member.split(':')
      params.group  = mGroup
      params.member = mName
    }
    if (signFilter.value.singleNumber) {
      const [sGroup, sNum] = signFilter.value.singleNumber.split(':')
      params.group         = sGroup
      params.single_number = sNum
    }
    const res = await getAdminSignEvents(params)
    signEvents.value = res.data.data ?? []
    signTotal.value  = res.data.total ?? 0
  } catch {}
}

onMounted(() => {
  loadUsers()
  reloadFilterLists()
  loadSignEvents()
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
.tag-won   { color: #52c41a; font-weight: bold; }
.tag-lost  { color: #ff4d4f; font-weight: bold; }
.filter-row { display: flex; flex-wrap: wrap; gap: 10px; margin-bottom: 14px; }
</style>
