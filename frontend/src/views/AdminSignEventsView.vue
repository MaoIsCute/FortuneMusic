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
        <el-input v-model="signFilter.member" placeholder="成員名稱" clearable style="width:150px" @change="signPage=1;loadSignEvents()" />
        <el-input v-model="signFilter.singleNumber" placeholder="單曲號" clearable style="width:100px" type="number" @change="signPage=1;loadSignEvents()" />
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
import { ref, onMounted } from 'vue'
import { getAdminUsers, getAdminSignEvents } from '../api/index'

const users      = ref([])
const signEvents  = ref([])
const signTotal   = ref(0)
const signPage    = ref(1)
const signPageSize = 50
const signFilter  = ref({ userId: null, member: '', singleNumber: '' })

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

async function loadSignEvents() {
  try {
    const params = { page: signPage.value, page_size: signPageSize }
    if (signFilter.value.userId) params.user_id = signFilter.value.userId
    if (signFilter.value.member.trim()) params.member = signFilter.value.member.trim()
    if (signFilter.value.singleNumber) params.single_number = signFilter.value.singleNumber
    const res = await getAdminSignEvents(params)
    signEvents.value = res.data.data ?? []
    signTotal.value  = res.data.total ?? 0
  } catch {}
}

onMounted(() => {
  loadUsers()
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
.tag-won   { color: #52c41a; font-weight: bold; }
.tag-lost  { color: #ff4d4f; font-weight: bold; }
.filter-row { display: flex; flex-wrap: wrap; gap: 10px; margin-bottom: 14px; }
</style>
