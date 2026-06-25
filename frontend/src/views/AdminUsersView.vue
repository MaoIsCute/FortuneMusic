<template>
  <div class="page">
    <h1 class="page-title">🔧 使用者管理</h1>

    <div class="card">
      <div class="card-header">
        <span class="card-title">使用者列表</span>
        <el-button size="small" @click="loadUsers">重新整理</el-button>
      </div>
      <el-table table-layout="auto" :data="users" stripe>
        <el-table-column prop="email" label="Email" min-width="220" />
        <el-table-column prop="name" label="名稱" min-width="90" />
        <el-table-column prop="record_count" label="個握筆數" min-width="80" />
        <el-table-column label="最後同步" min-width="160">
          <template #default="{ row }">
            {{ row.last_scraped ? row.last_scraped.replace('T', ' ').slice(0, 16) : '—' }}
          </template>
        </el-table-column>
        <el-table-column label="" min-width="100">
          <template #default="{ row }">
            <el-button type="primary" size="small" plain @click="viewAs(row)">模擬畫面</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getAdminUsers } from '../api/index'
import { useImpersonateStore } from '../stores/impersonate'
import { useDataStore } from '../stores/data'

const router = useRouter()
const impersonateStore = useImpersonateStore()
const dataStore = useDataStore()
const users = ref([])

function viewAs(user) {
  impersonateStore.start(user)
  dataStore.invalidate()
  router.push('/dashboard')
}

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

onMounted(loadUsers)
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
</style>
