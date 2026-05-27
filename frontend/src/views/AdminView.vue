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
        <el-table-column prop="record_count" label="筆數" width="80" />
        <el-table-column label="最後同步" width="170">
          <template #default="{ row }">
            {{ row.last_scraped ? row.last_scraped.replace('T', ' ').slice(0, 16) : '—' }}
          </template>
        </el-table-column>
        <el-table-column label="" width="120">
          <template #default="{ row }">
            <el-button
              type="danger"
              size="small"
              :disabled="row.record_count === 0"
              @click="confirmDelete(row)"
            >
              清除資料
            </el-button>
          </template>
        </el-table-column>
      </el-table>
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
import { getAdminTitleIssues, fixSingleTitle, getAdminUsers, deleteUserRecords } from '../api/index'

const router = useRouter()
const users  = ref([])
const issues = ref([])

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

async function confirmDelete(row) {
  try {
    await ElMessageBox.confirm(
      `確定要清除 ${row.email} 的全部 ${row.record_count} 筆資料嗎？此操作無法復原。`,
      '清除確認',
      { confirmButtonText: '確定清除', cancelButtonText: '取消', type: 'warning' }
    )
    const res = await deleteUserRecords(row.id)
    ElMessage.success(`已清除 ${res.data.deleted} 筆`)
    await loadUsers()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e.response?.data?.error || '清除失敗')
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
</style>
