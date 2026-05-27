<template>
  <div class="page">
    <h1 class="page-title">🔧 管理</h1>

    <el-card class="section">
      <template #header>
        <span>タイトル未定 修正</span>
        <el-button style="float:right" size="small" @click="load">重新整理</el-button>
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
import { ElMessage } from 'element-plus'
import { getAdminTitleIssues, fixSingleTitle } from '../api/index'

const router = useRouter()

const issues = ref([])

async function load() {
  try {
    const res = await getAdminTitleIssues()
    issues.value = (res.data ?? []).map(item => ({
      ...item,
      _input:   item.suggested_name || '',
      _loading: false,
    }))
  } catch (e) {
    if (e.response?.status === 403) {
      ElMessage.error('無權限')
      router.replace('/dashboard')
    }
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
    await load()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '更新失敗')
  } finally {
    row._loading = false
  }
}

onMounted(load)
</script>

<style scoped>
.section { margin-bottom: 24px; }
.empty { color: #999; text-align: center; padding: 32px 0; }
</style>
