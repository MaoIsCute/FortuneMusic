<template>
  <div class="page">
    <h1 class="page-title">🔧 管理</h1>

    <el-collapse v-model="openSections">

      <!-- 使用者管理 -->
      <el-collapse-item name="users">
        <template #title>
          <span class="collapse-title">使用者管理</span>
          <el-button size="small" style="margin-left:12px" @click.stop="loadUsers">重新整理</el-button>
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
          <el-table-column label="" width="100">
            <template #default="{ row }">
              <el-button type="primary" size="small" plain @click="viewAs(row)">模擬畫面</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-collapse-item>

      <!-- 刪除資料 -->
      <el-collapse-item name="delete">
        <template #title><span class="collapse-title">刪除資料</span></template>

        <div class="delete-form">
          <el-select v-model="del.recordType" style="width:120px" @change="clearPreview">
            <el-option label="個握" value="records" />
            <el-option label="全握" value="full-records" />
            <el-option label="個握花費" value="purchases" />
          </el-select>
          <el-select v-model="del.userId" placeholder="選擇使用者" style="width:200px" clearable @change="clearPreview">
            <el-option v-for="u in users" :key="u.id" :label="`${u.name} (${u.email})`" :value="u.id" />
          </el-select>
          <el-select v-if="del.recordType !== 'purchases'" v-model="del.group" placeholder="團體（全部）" clearable style="width:120px" @change="clearPreview">
            <el-option label="乃木坂46" value="nogizaka46" />
            <el-option label="櫻坂46" value="sakurazaka46" />
            <el-option label="日向坂46" value="hinatazaka46" />
          </el-select>
          <el-select v-model="del.mode" style="width:140px" @change="clearPreview">
            <el-option label="全部" value="all" />
            <el-option label="指定單曲" value="single" />
            <el-option label="指定日期範圍" value="date" />
          </el-select>
          <el-input v-if="del.mode === 'single'" v-model="del.singleNumber" placeholder="單曲號" style="width:100px" type="number" @change="clearPreview" />
          <el-date-picker
            v-if="del.mode === 'date'"
            v-model="del.dateRange"
            type="daterange"
            range-separator="～"
            start-placeholder="開始日期"
            end-placeholder="結束日期"
            format="YYYY/M/D"
            value-format="YYYY/M/D"
            style="width:260px"
            @change="clearPreview"
          />
          <el-button :disabled="!del.userId" :loading="previewLoading" @click="queryPreview">查詢</el-button>
        </div>

        <!-- 查詢結果 -->
        <template v-if="previewExecuted">
          <div v-if="previewTotal === 0" class="empty">查無符合資料</div>
          <template v-else>
            <div class="preview-header">共 <b>{{ previewTotal }}</b> 筆符合條件</div>

            <!-- 個握 -->
            <el-table v-if="del.recordType === 'records'" :data="previewData" stripe size="small" max-height="400">
              <el-table-column prop="member_name" label="成員" width="120" />
              <el-table-column label="單曲" width="70">
                <template #default="{ row }">{{ formatSingle(row.single_name) }}</template>
              </el-table-column>
              <el-table-column prop="event_date" label="日期" width="100" />
              <el-table-column prop="session" label="部數" width="80" />
              <el-table-column label="抽次" width="70">
                <template #default="{ row }">{{ row.lottery_round > 0 ? row.lottery_round + '抽' : '—' }}</template>
              </el-table-column>
              <el-table-column prop="applied_count" label="應募" width="60" align="right" />
              <el-table-column prop="won_count" label="中選" width="60" align="right" />
            </el-table>

            <!-- 全握 -->
            <el-table v-else-if="del.recordType === 'full-records'" :data="previewData" stripe size="small" max-height="400">
              <el-table-column prop="member_name" label="成員" width="130" />
              <el-table-column prop="event_type" label="類型" width="65" />
              <el-table-column label="場地" width="150">
                <template #default="{ row }"><span style="white-space:nowrap">{{ row.venue || '—' }}</span></template>
              </el-table-column>
              <el-table-column prop="event_date" label="日期" width="100" />
              <el-table-column prop="session" label="部數" width="80" />
              <el-table-column label="單曲" width="70">
                <template #default="{ row }">{{ formatSingle(row.single_name) }}</template>
              </el-table-column>
              <el-table-column prop="applied_count" label="應募" width="60" align="right" />
              <el-table-column prop="won_count" label="中選" width="60" align="right" />
            </el-table>

            <!-- 個握花費 -->
            <el-table v-else :data="previewData" stripe size="small" max-height="400">
              <el-table-column prop="member_name" label="成員" width="120" />
              <el-table-column label="單曲" width="70">
                <template #default="{ row }">{{ formatSingle(row.single_name) }}</template>
              </el-table-column>
              <el-table-column prop="event_date" label="日期" width="100" />
              <el-table-column prop="session" label="部數" width="80" />
              <el-table-column prop="unit_price" label="單價" width="80" align="right" />
              <el-table-column prop="quantity" label="數量" width="60" align="right" />
              <el-table-column prop="subtotal" label="小計" width="80" align="right" />
            </el-table>

            <el-pagination
              v-if="previewTotal > 50"
              v-model:current-page="previewPage"
              :page-size="50"
              :total="previewTotal"
              layout="prev, pager, next"
              style="margin-top:12px;display:flex;justify-content:flex-end"
              @current-change="loadPreviewPage"
            />

            <div class="delete-action">
              <el-button type="danger" @click="execDelete">刪除這 {{ previewTotal }} 筆</el-button>
            </div>
          </template>
        </template>

      </el-collapse-item>

      <!-- タイトル未定 修正 -->
      <el-collapse-item name="titles">
        <template #title>
          <span class="collapse-title">タイトル未定 修正</span>
          <el-button size="small" style="margin-left:12px" @click.stop="loadIssues">重新整理</el-button>
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
              <el-input v-model="row._input" size="small" placeholder="輸入正確標題" style="width:320px" />
            </template>
          </el-table-column>
          <el-table-column label="" width="90">
            <template #default="{ row }">
              <el-button type="primary" size="small" :loading="row._loading" @click="fix(row)">修正</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-collapse-item>

      <!-- 抓取紀錄 -->
      <el-collapse-item name="logs">
        <template #title>
          <span class="collapse-title">抓取紀錄</span>
          <el-button size="small" style="margin-left:12px" @click.stop="loadScrapeLogs">重新整理</el-button>
        </template>
        <div v-if="scrapeLogs.length === 0" class="empty">尚無紀錄</div>
        <el-table v-else :data="scrapeLogs" stripe>
          <el-table-column label="使用者" width="140">
            <template #default="{ row }">{{ row.user_name }}<br/><span class="sub-text">{{ row.user_email }}</span></template>
          </el-table-column>
          <el-table-column prop="type" label="類型" width="90" />
          <el-table-column label="時間" width="140">
            <template #default="{ row }">{{ row.created_at ? row.created_at.replace('T', ' ').slice(0, 16) : '—' }}</template>
          </el-table-column>
          <el-table-column label="新增" width="65" align="right" prop="new_count" />
          <el-table-column label="跳過" width="65" align="right" prop="skip_count" />
          <el-table-column label="時長" width="75" align="right">
            <template #default="{ row }">{{ row.duration_sec > 0 ? formatDuration(row.duration_sec) : '—' }}</template>
          </el-table-column>
          <el-table-column label="狀態">
            <template #default="{ row }">
              <span v-if="row.error" class="tag-error">❌ {{ row.error }}</span>
              <span v-else class="tag-ok">✅ 成功</span>
            </template>
          </el-table-column>
        </el-table>
      </el-collapse-item>

      <!-- 簽名會紀錄 -->
      <el-collapse-item name="sign">
        <template #title>
          <span class="collapse-title">簽名會紀錄</span>
          <el-button size="small" style="margin-left:12px" @click.stop="loadSignEvents">重新整理</el-button>
        </template>
        <div class="filter-row">
          <el-select v-model="signFilter.userId" placeholder="篩選使用者" clearable style="width:200px" @change="signPage=1;loadSignEvents()">
            <el-option v-for="u in users" :key="u.id" :label="`${u.name} (${u.email})`" :value="u.id" />
          </el-select>
          <el-input v-model="signFilter.member" placeholder="成員名稱" clearable style="width:150px" @change="signPage=1;loadSignEvents()" />
          <el-input v-model="signFilter.singleNumber" placeholder="單曲號" clearable style="width:100px" type="number" @change="signPage=1;loadSignEvents()" />
        </div>
        <div v-if="signEvents.length === 0" class="empty">尚無簽名會紀錄</div>
        <el-table v-else :data="signEvents" stripe>
          <el-table-column label="使用者" width="140">
            <template #default="{ row }">{{ row.user_name }}<br/><span class="sub-text">{{ row.user_email }}</span></template>
          </el-table-column>
          <el-table-column prop="member_name" label="成員" width="110" />
          <el-table-column label="單曲" width="80">
            <template #default="{ row }">{{ formatSingle(row.single_name) || `${row.single_number}單` }}</template>
          </el-table-column>
          <el-table-column prop="event_date" label="日期" width="120" />
          <el-table-column label="抽次" width="70" align="center">
            <template #default="{ row }">{{ row.lottery_round > 0 ? row.lottery_round + '抽' : '—' }}</template>
          </el-table-column>
          <el-table-column label="應募" width="75" align="right">
            <template #default="{ row }">{{ Math.round(row.applied_count / 3) }} 口</template>
          </el-table-column>
          <el-table-column label="結果" width="75" align="center">
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
      </el-collapse-item>

    </el-collapse>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getAdminTitleIssues, fixSingleTitle, getAdminUsers, deleteUserRecords, deleteUserFullRecords, deleteUserPurchases, previewUserRecords, previewUserFullRecords, previewUserPurchases, getAdminScrapeLogs, getAdminSignEvents } from '../api/index'
import { useImpersonateStore } from '../stores/impersonate'
import { useDataStore } from '../stores/data'

const router = useRouter()
const impersonateStore = useImpersonateStore()
const dataStore = useDataStore()
const openSections = ref(['users'])
const users      = ref([])
const issues     = ref([])
const scrapeLogs = ref([])
const signEvents  = ref([])
const signTotal   = ref(0)
const signPage    = ref(1)
const signPageSize = 50
const signFilter  = ref({ userId: null, member: '', singleNumber: '' })

const del = ref({
  mode:         'all',
  recordType:   'records',
  userId:       null,
  group:        '',
  singleNumber: '',
  dateRange:    [],
})

const previewData     = ref([])
const previewTotal    = ref(0)
const previewPage     = ref(1)
const previewExecuted = ref(false)
const previewLoading  = ref(false)

function buildDelParams() {
  const params = {}
  if (del.value.group)        params.group         = del.value.group
  if (del.value.mode === 'single' && del.value.singleNumber) params.single_number = del.value.singleNumber
  if (del.value.mode === 'date' && del.value.dateRange?.length === 2) {
    params.date_from = del.value.dateRange[0]
    params.date_to   = del.value.dateRange[1]
  }
  return params
}

function clearPreview() {
  previewData.value     = []
  previewTotal.value    = 0
  previewPage.value     = 1
  previewExecuted.value = false
}

async function queryPreview() {
  if (!del.value.userId) return
  previewLoading.value = true
  previewPage.value = 1
  try {
    await loadPreviewPage()
    previewExecuted.value = true
  } finally {
    previewLoading.value = false
  }
}

async function loadPreviewPage() {
  const params = { ...buildDelParams(), page: previewPage.value }
  const fnMap = {
    'records':     previewUserRecords,
    'full-records': previewUserFullRecords,
    'purchases':   previewUserPurchases,
  }
  const res = await fnMap[del.value.recordType](del.value.userId, params)
  previewData.value  = res.data.data  ?? []
  previewTotal.value = res.data.total ?? 0
}

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
  const typeLabel = { records: '個握', 'full-records': '全握', purchases: '個握花費' }[del.value.recordType] ?? '個握'
  try {
    await ElMessageBox.confirm(
      `確定要刪除 ${user.name}（${user.email}）的 ${typeLabel} 共 ${previewTotal.value} 筆資料？此操作無法復原。`,
      '刪除確認',
      { confirmButtonText: '確定刪除', cancelButtonText: '取消', type: 'warning' }
    )
  } catch { return }
  try {
    const fnMap = { records: deleteUserRecords, 'full-records': deleteUserFullRecords, purchases: deleteUserPurchases }
    const res = await fnMap[del.value.recordType](del.value.userId, buildDelParams())
    ElMessage.success(`已刪除 ${res.data.deleted} 筆`)
    clearPreview()
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

function formatSingle(name) {
  if (!name) return ''
  return name
    .replace(/(\d+)(?:st|nd|rd|th)シングル/, (_, n) => `${n}單`)
    .replace(/(\d+)(?:st|nd|rd|th)アルバム/, (_, n) => `${n}專`)
}

function formatDuration(sec) {
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return m > 0 ? `${m}m${s}s` : `${s}s`
}

async function loadScrapeLogs() {
  try {
    const res = await getAdminScrapeLogs()
    scrapeLogs.value = res.data ?? []
  } catch {}
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
  loadIssues()
  loadScrapeLogs()
  loadSignEvents()
})
</script>

<style scoped>
.page { background: #f5f7fa; min-height: 100vh; }

:deep(.el-collapse) {
  border: none;
  background: transparent;
}

:deep(.el-collapse-item) {
  margin-bottom: 12px;
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid #e5e7eb;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.07);
  background: white;
}

:deep(.el-collapse-item__header) {
  height: 52px;
  padding: 0 20px;
  font-size: 14px;
  font-weight: 600;
  color: #111827;
  background: white;
  border-bottom: 1px solid transparent;
}

:deep(.el-collapse-item.is-active .el-collapse-item__header) {
  border-bottom-color: #e5e7eb;
}

:deep(.el-collapse-item__arrow) {
  color: #6b7280;
}

:deep(.el-collapse-item__wrap) {
  background: white;
  border: none;
}

:deep(.el-collapse-item__content) {
  padding: 16px 20px 20px;
}

.collapse-title { font-weight: 600; font-size: 14px; }
.empty { color: #999; text-align: center; padding: 32px 0; }
.delete-form { display: flex; flex-wrap: wrap; gap: 10px; align-items: center; }
.sub-text { font-size: 11px; color: #999; }
.tag-ok    { color: #059669; font-size: 13px; }
.tag-error { color: #dc2626; font-size: 13px; }
.tag-won   { color: #52c41a; font-weight: bold; }
.tag-lost  { color: #ff4d4f; font-weight: bold; }
.filter-row { display: flex; flex-wrap: wrap; gap: 10px; margin-bottom: 14px; }
</style>
