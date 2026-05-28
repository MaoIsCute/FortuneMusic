<template>
  <div class="page">
    <h1 class="page-title">🔧 管理</h1>

    <!-- 使用者管理 -->
    <el-card class="section">
      <template #header>
        <span>使用者管理</span>
        <el-button style="float:right" size="small" @click="loadUsers">重新整理</el-button>
      </template>
      <el-table :data="users" stripe>
        <el-table-column prop="email" label="Email" />
        <el-table-column prop="name" label="名稱" width="120" />
        <el-table-column prop="record_count" label="個握筆數" width="100" />
        <el-table-column label="最後同步" width="160">
          <template #default="{ row }">
            {{ row.last_scraped ? row.last_scraped.replace('T', ' ').slice(0, 16) : '—' }}
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 刪除資料 -->
    <el-card class="section">
      <template #header><span>刪除資料</span></template>
      <div class="delete-form">
        <el-select v-model="del.mode" placeholder="選擇刪除方式" style="width:180px">
          <el-option label="清除某人全部資料" value="all" />
          <el-option label="清除特定單曲" value="single" />
          <el-option label="清除特定日期範圍" value="date" />
        </el-select>

        <el-select v-model="del.recordType" style="width:100px">
          <el-option label="個握" value="records" />
          <el-option label="全握" value="full-records" />
        </el-select>

        <el-select v-model="del.userId" placeholder="選擇使用者" style="width:200px" clearable>
          <el-option v-for="u in users" :key="u.id" :label="`${u.name} (${u.email})`" :value="u.id" />
        </el-select>

        <template v-if="del.mode === 'single'">
          <el-input v-model="del.singleNumber" placeholder="單曲號 (如 41)" style="width:130px" type="number" />
        </template>

        <template v-if="del.mode === 'date'">
          <el-date-picker
            v-model="del.dateRange"
            type="daterange"
            range-separator="～"
            start-placeholder="開始日期"
            end-placeholder="結束日期"
            format="YYYY/M/D"
            value-format="YYYY/M/D"
            style="width:260px"
          />
        </template>

        <el-button type="danger" :disabled="!del.userId || !del.mode" @click="execDelete">
          確定刪除
        </el-button>
      </div>
    </el-card>

    <!-- タイトル未定 修正 -->
    <el-card class="section">
      <template #header>
        <span>タイトル未定 修正</span>
        <el-button style="float:right" size="small" @click="loadIssues">重新整理</el-button>
      </template>

      <div v-if="issues.length === 0" class="empty">目前沒有 タイトル未定 的紀錄</div>

      <el-table v-else :data="issues" stripe>
        <el-table-column label="單曲號" width="80">
          <template #default="{ row }">{{ row.single_number }}</template>
        </el-table-column>
        <el-table-column prop="current_name" label="目前標題" />
        <el-table-column label="筆數" width="70" prop="count" />
        <el-table-column label="修正標題">
          <template #default="{ row }">
            <el-input
              v-model="row._input"
              size="small"
              placeholder="輸入正確標題"
              style="width: 320px"
            />
          </template>
        </el-table-column>
        <el-table-column label="" width="90">
          <template #default="{ row }">
            <el-button
              type="primary"
              size="small"
              :loading="row._loading"
              @click="fix(row)"
            >
              修正
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getAdminTitleIssues, fixSingleTitle, getAdminUsers, deleteUserRecords, deleteUserFullRecords } from '../api/index'

const router = useRouter()
const users  = ref([])
const issues = ref([])

const del = ref({
  mode:         '',
  recordType:   'records',
  userId:       null,
  singleNumber: '',
  dateRange:    [],
})

async function loadUsers() {
  try {
    const res = await getAdminUsers()
    users.value = res.data ?? []
  } catch (e) {
    if (e.response?.status === 403) {
      ElMessage.error('無權限')
      router.replace('/dashboard')
    }
  }
}

async function loadIssues() {
  try {
    const res = await getAdminTitleIssues()
    issues.value = (res.data ?? []).map(item => ({
      ...item,
      _input:   item.suggested_name || '',
      _loading: false,
    }))
  } catch {}
}

async function execDelete() {
  const user = users.value.find(u => u.id === del.value.userId)
  if (!user) return

  const modeLabel = { all: '全部資料', single: `第 ${del.value.singleNumber} 單`, date: `${del.value.dateRange?.[0]} ～ ${del.value.dateRange?.[1]}` }
  const typeLabel = del.value.recordType === 'full-records' ? '全握' : '個握'

  try {
    await ElMessageBox.confirm(
      `確定要刪除 ${user.name}（${user.email}）的${typeLabel} ${modeLabel[del.value.mode]} 資料嗎？此操作無法復原。`,
      '刪除確認',
      { confirmButtonText: '確定刪除', cancelButtonText: '取消', type: 'warning' }
    )
  } catch { return }

  const params = {}
  if (del.value.mode === 'single' && del.value.singleNumber) {
    params.single_number = del.value.singleNumber
  }
  if (del.value.mode === 'date' && del.value.dateRange?.length === 2) {
    params.date_from = del.value.dateRange[0]
    params.date_to   = del.value.dateRange[1]
  }

  try {
    const fn  = del.value.recordType === 'full-records' ? deleteUserFullRecords : deleteUserRecords
    const res = await fn(del.value.userId, params)
    ElMessage.success(`已刪除 ${res.data.deleted} 筆`)
    await loadUsers()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '刪除失敗')
  }
}

async function fix(row) {
  if (!row._input.trim()) {
    ElMessage.warning('請輸入正確標題')
    return
  }
  row._loading = true
  try {
    const res = await fixSingleTitle(row.single_number, row._input.trim())
    ElMessage.success(`已更新 ${res.data.updated} 筆`)
    await loadIssues()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '更新失敗')
  } finally {
    row._loading = false
  }
}

onMounted(() => {
  loadUsers()
  loadIssues()
})
</script>

<style scoped>
.section { margin-bottom: 24px; }
.empty { color: #999; text-align: center; padding: 32px 0; }
.delete-form { display: flex; flex-wrap: wrap; gap: 10px; align-items: center; }
</style>
