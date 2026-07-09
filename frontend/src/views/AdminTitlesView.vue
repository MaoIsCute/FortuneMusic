<template>
  <div class="page">
    <h1 class="page-title">🔧 單曲名稱</h1>

    <el-collapse v-model="openSections">

      <!-- タイトル未定 修正 -->
      <el-collapse-item name="titles">
        <template #title>
          <span class="collapse-title">問題列表</span>
          <el-button size="small" style="margin-left:12px" @click.stop="loadIssues">重新整理</el-button>
        </template>
        <div v-if="issues.length === 0" class="empty">目前沒有 タイトル未定 的紀錄</div>
        <el-table table-layout="auto" v-else :data="issues" stripe :row-style="rowStyle">
          <el-table-column label="團體" min-width="70">
            <template #default="{ row }">{{ groupLabel(row.group) }}</template>
          </el-table-column>
          <el-table-column label="單曲號" min-width="60">
            <template #default="{ row }">{{ row.single_number }}</template>
          </el-table-column>
          <el-table-column label="目前標題" min-width="280">
            <template #default="{ row }">
              <span v-if="!row.current_name" class="text-muted">（空白）</span>
              <span v-else>{{ row.current_name }}</span>
            </template>
          </el-table-column>
          <el-table-column label="筆數" min-width="50" prop="count" />
          <el-table-column label="修正標題">
            <template #default="{ row }">
              <el-input v-model="row._input" size="small" placeholder="輸入正確標題" style="width:320px" />
            </template>
          </el-table-column>
          <el-table-column label="" min-width="80">
            <template #default="{ row }">
              <el-button type="primary" size="small" :loading="row._loading" @click="fix(row)">修正</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-collapse-item>

      <!-- 批次登記 -->
      <el-collapse-item name="bulk">
        <template #title><span class="collapse-title">單曲名稱批次登記</span></template>
        <p class="sub-text">批次登記已知單曲名稱（不需要先出現 タイトル未定 問題），一行一筆，格式：<code>團體代碼,單曲號,單曲名稱</code>，團體代碼為 nogizaka46 / sakurazaka46 / hinatazaka46。專輯請填單曲號 0，同一團體可以登記多張不同名稱的專輯（不會視為重複）；單曲同一團體+單曲號重複時取最後一行。</p>
        <p class="sub-text">
          範例：
          <el-button size="small" text @click="copyExample(TITLE_EXAMPLE)">📋 複製範例</el-button>
        </p>
        <pre class="example-block">{{ TITLE_EXAMPLE }}</pre>
        <el-input
          v-model="bulkTitleText"
          type="textarea"
          :rows="6"
          placeholder="nogizaka46,42,42ndシングル『○○○』&#10;sakurazaka46,8,8thシングル『○○○』"
        />
        <el-button type="primary" :loading="bulkLoading" style="margin-top:8px" @click="submitBulkTitles">批次送出</el-button>
      </el-collapse-item>

      <!-- 已知單曲名稱 -->
      <el-collapse-item name="known">
        <template #title>
          <span class="collapse-title">已知單曲名稱</span>
          <el-button size="small" style="margin-left:12px" @click.stop="loadKnownTitles">重新整理</el-button>
        </template>
        <p class="sub-text">只列出資料庫裡已經出現過的單曲，不是官方完整發行紀錄；專輯（單曲號 0）沒有可靠編號，以名稱本身互相區分，同一團體的多張專輯會各自列出。</p>
        <el-select v-model="knownGroupFilter" placeholder="團體（全部）" clearable style="width:140px;margin-bottom:12px">
          <el-option label="乃木坂46" value="nogizaka46">
            <span style="color:#9333ea;font-weight:500">乃木坂46</span>
          </el-option>
          <el-option label="櫻坂46" value="sakurazaka46">
            <span style="color:#ec4899;font-weight:500">櫻坂46</span>
          </el-option>
          <el-option label="日向坂46" value="hinatazaka46">
            <span style="color:#0ea5e9;font-weight:500">日向坂46</span>
          </el-option>
        </el-select>
        <div v-if="filteredKnownTitles.length === 0" class="empty">沒有資料</div>
        <el-table table-layout="auto" v-else :data="filteredKnownTitles" stripe size="small" max-height="500" :row-style="rowStyle">
          <el-table-column label="團體" min-width="120">
            <template #default="{ row }">
              <el-select v-if="row._editing" v-model="row._group" size="small" style="width:110px">
                <el-option v-for="g in GROUP_OPTIONS" :key="g.value" :value="g.value" :label="g.label">
                  <span :style="{ color: g.color, fontWeight: 500 }">{{ g.label }}</span>
                </el-option>
                <template #prefix>
                  <span :style="{ color: GROUP_COLORS[row._group], fontWeight: 600 }">●</span>
                </template>
              </el-select>
              <span v-else :style="{ color: GROUP_COLORS[row.group], fontWeight: 500 }">{{ groupLabel(row.group) }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="single_number" label="單曲號" min-width="60" />
          <el-table-column label="單曲名稱" min-width="300">
            <template #default="{ row }">
              <el-input v-if="row._editing" v-model="row._input" size="small" style="width:100%" />
              <span v-else :style="{ color: GROUP_COLORS[row.group] }">{{ row.single_name }}</span>
            </template>
          </el-table-column>
          <el-table-column label="發售日" min-width="140">
            <template #default="{ row }">
              <el-date-picker v-if="row._editing" v-model="row._date" type="date"
                value-format="YYYY-MM-DD" format="YYYY-MM-DD"
                size="small" placeholder="YYYY-MM-DD" style="width:130px" />
              <span v-else style="color:#888;font-size:12px">{{ row.release_date || '—' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="來源" min-width="90">
            <template #default="{ row }">{{ sourceLabel(row.source) }}</template>
          </el-table-column>
          <el-table-column label="" min-width="130">
            <template #default="{ row }">
              <template v-if="row._editing">
                <el-button type="primary" size="small" :loading="row._loading" @click="saveKnown(row)">確認</el-button>
                <el-button size="small" @click="cancelEdit(row)">取消</el-button>
              </template>
              <el-button v-else size="small" @click="startEdit(row)">編輯</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-collapse-item>

    </el-collapse>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getAdminTitleIssues, getAdminKnownTitles, fixSingleTitle, bulkSetTitles } from '../api/index'

const openSections = ref(['titles', 'bulk', 'known'])
const issues = ref([])

const TITLE_EXAMPLE = 'nogizaka46,42,42ndシングル『○○○』\nsakurazaka46,8,8thシングル『○○○』'

async function copyExample(text) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已複製到剪貼簿')
  } catch {
    ElMessage.error('複製失敗，請手動選取複製')
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

async function fix(row) {
  if (!row._input.trim()) {
    ElMessage.warning('請輸入正確標題')
    return
  }
  row._loading = true
  try {
    const orgAlbumName = row.single_number === 0 ? (row.current_name || '') : ''
    const res = await fixSingleTitle(row.group, row.single_number, row._input.trim(), orgAlbumName)
    ElMessage.success(`已更新 ${res.data.updated} 筆`)
    await loadIssues()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '更新失敗')
  } finally {
    row._loading = false
  }
}

const GROUP_LABELS = { nogizaka46: '乃木坂46', sakurazaka46: '櫻坂46', hinatazaka46: '日向坂46' }
const GROUP_COLORS = { nogizaka46: '#9333ea', sakurazaka46: '#ec4899', hinatazaka46: '#0ea5e9' }
const GROUP_OPTIONS = [
  { value: 'nogizaka46',  label: '乃木坂46', color: '#9333ea' },
  { value: 'sakurazaka46', label: '櫻坂46',  color: '#ec4899' },
  { value: 'hinatazaka46', label: '日向坂46', color: '#0ea5e9' },
]
function groupLabel(g) {
  return GROUP_LABELS[g] || g || '—'
}
function rowStyle({ row }) {
  return { color: GROUP_COLORS[row.group] }
}

const bulkTitleText = ref('')
const bulkLoading    = ref(false)

async function submitBulkTitles() {
  const lines = bulkTitleText.value.split('\n').map(l => l.trim()).filter(Boolean)
  if (lines.length === 0) {
    ElMessage.warning('請輸入至少一行')
    return
  }

  // 同一團體+單曲號重複時排重，取最後一行
  const titleMap = new Map()
  for (const line of lines) {
    const parts = line.split(',')
    if (parts.length < 3) {
      ElMessage.error(`格式錯誤：${line}`)
      return
    }
    const group = parts[0].trim()
    const singleNumber = parseInt(parts[1].trim())
    const singleName = parts.slice(2).join(',').trim()
    if (!GROUP_LABELS[group]) {
      ElMessage.error(`團體代碼不正確：${group}`)
      return
    }
    if (isNaN(singleNumber) || !singleName) {
      ElMessage.error(`格式錯誤：${line}`)
      return
    }
    // 專輯（single_number === 0）沒有可靠編號，用名稱本身排重，不然不同專輯會被誤判成重複
    const key = singleNumber === 0 ? `${group}:0:${singleName}` : `${group}:${singleNumber}`
    titleMap.set(key, { group, single_number: singleNumber, single_name: singleName })
  }

  const duplicateCount = lines.length - titleMap.size
  const titles = Array.from(titleMap.values())

  bulkLoading.value = true
  try {
    const res = await bulkSetTitles(titles)
    const dupMsg = duplicateCount > 0 ? `（已自動排除 ${duplicateCount} 筆重複，取最後一筆）` : ''
    ElMessage.success(`已登記 ${res.data.applied} 筆，回填更新 ${res.data.updated} 筆${dupMsg}`)
    bulkTitleText.value = ''
    await loadIssues()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '批次送出失敗')
  } finally {
    bulkLoading.value = false
  }
}

const knownTitles = ref([])
const knownGroupFilter = ref('')
const filteredKnownTitles = computed(() =>
  knownGroupFilter.value ? knownTitles.value.filter(t => t.group === knownGroupFilter.value) : knownTitles.value
)

const SOURCE_LABELS = { correction: '已登記修正', records: '個握紀錄推測', purchases: '購入紀錄推測' }
function sourceLabel(s) {
  return SOURCE_LABELS[s] || s || '—'
}

async function loadKnownTitles() {
  try {
    const res = await getAdminKnownTitles()
    knownTitles.value = (res.data ?? []).map(t => ({
      ...t,
      _editing: false,
      _input: t.single_name,
      _date: t.release_date || '',
      _group: t.group,
      _loading: false,
    }))
  } catch {}
}

function startEdit(row) {
  row._editing = true
  row._input = row.single_name
  row._date = row.release_date || ''
  row._group = row.group
}

function cancelEdit(row) {
  row._editing = false
  row._input = row.single_name
  row._date = row.release_date || ''
  row._group = row.group
}

async function saveKnown(row) {
  const newName = row._input.trim()
  const newDate = row._date || ''
  const newGroup = row._group
  if (!newName) { cancelEdit(row); return }
  if (newName === row.single_name && newDate === (row.release_date || '') && newGroup === row.group) { cancelEdit(row); return }
  row._loading = true
  try {
    const orgAlbumName = row.single_number === 0 ? (row.org_album_name || row.single_name) : ''
    const res = await fixSingleTitle(newGroup, row.single_number, newName, orgAlbumName, newDate)
    ElMessage.success(`已更新 ${res.data.updated} 筆`)
    await Promise.all([loadKnownTitles(), loadIssues()])
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '更新失敗')
    row._loading = false
    row._editing = false
  }
}

onMounted(() => {
  loadIssues()
  loadKnownTitles()
})
</script>

<style scoped>
.page { background: #f5f7fa; min-height: 100vh; }

:deep(.el-table .cell) { white-space: nowrap; }
:deep(.el-collapse) { border: none; background: transparent; }
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
:deep(.el-collapse-item.is-active .el-collapse-item__header) { border-bottom-color: #e5e7eb; }
:deep(.el-collapse-item__arrow) { color: #6b7280; }
:deep(.el-collapse-item__wrap) { background: white; border: none; }
:deep(.el-collapse-item__content) { padding: 16px 20px 20px; }

.collapse-title { font-weight: 600; font-size: 14px; }
.empty { color: #999; text-align: center; padding: 32px 0; }
.sub-text { font-size: 11px; color: #999; display: block; margin-bottom: 8px; }
.text-muted { color: #999; }
.example-block {
  font-family: monospace;
  font-size: 12px;
  background: #f5f7fa;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  padding: 8px 12px;
  margin: 0 0 10px;
  white-space: pre;
  overflow-x: auto;
  user-select: all;
}
</style>
