<template>
  <div class="page">
    <h1 class="page-title">🏟️ 場地管理</h1>

    <el-collapse v-model="openSections">

      <!-- 缺少/衝突場地 -->
      <el-collapse-item name="issues">
        <template #title>
          <span class="collapse-title">缺少/衝突場地</span>
          <el-button size="small" style="margin-left:12px" @click.stop="loadVenueIssues">重新整理</el-button>
        </template>
        <p class="sub-text">早期抓取版本沒有解析実体場地欄位，來源網站也無法回溯，只能人工登記；登記後會立即回填既有空白紀錄，之後同一張單同一天的新資料（補抓、重新同步）也會自動套用，不用每次手動修。也包含場地跟已登記值不一致、或同一場次出現多種互相衝突場地文字的紀錄，需人工比對挑出正確版本。</p>
        <div v-if="venueIssues.length === 0" class="empty">目前沒有缺少/衝突場地的紀錄</div>
        <el-table table-layout="auto" v-else :data="venueIssues" stripe :row-style="rowStyle">
          <el-table-column label="團體" min-width="70">
            <template #default="{ row }">{{ groupLabel(row.group) }}</template>
          </el-table-column>
          <el-table-column label="單曲號" min-width="60" prop="single_number" />
          <el-table-column label="單曲名稱" min-width="200" prop="single_name" />
          <el-table-column label="日期" min-width="100" prop="event_date" />
          <el-table-column label="目前場地" min-width="150">
            <template #default="{ row }">
              <span v-if="!row.current_venue" class="text-muted">（空白）</span>
              <span v-else>{{ row.current_venue }}</span>
            </template>
          </el-table-column>
          <el-table-column label="筆數" min-width="50" prop="count" />
          <el-table-column label="修正為">
            <template #default="{ row }">
              <el-input v-model="row._input" size="small" placeholder="輸入場地名稱" style="width:220px" />
            </template>
          </el-table-column>
          <el-table-column label="" min-width="80">
            <template #default="{ row }">
              <el-button type="primary" size="small" :loading="row._loading" @click="fixVenueRow(row)">修正</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-collapse-item>

      <!-- 批次登記 -->
      <el-collapse-item name="bulk">
        <template #title><span class="collapse-title">場地批次登記</span></template>
        <p class="sub-text">批次登記已知場地（不需要先出現缺少場地問題），一行一筆，格式：<code>團體代碼,單曲號,日期,場地</code>，團體代碼為 nogizaka46 / sakurazaka46 / hinatazaka46，日期格式跟 DB 一致（例：2023/11/19）。同一團體+單曲號+日期重複時取最後一行。</p>
        <p class="sub-text">
          範例：
          <el-button size="small" text @click="copyExample(VENUE_EXAMPLE)">📋 複製範例</el-button>
        </p>
        <pre class="example-block">{{ VENUE_EXAMPLE }}</pre>
        <el-input
          v-model="bulkVenueText"
          type="textarea"
          :rows="6"
          placeholder="nogizaka46,33,2023/11/19,幕張メッセ&#10;sakurazaka46,8,2024/4/7,京都パルスプラザ"
        />
        <el-button type="primary" :loading="bulkLoading" style="margin-top:8px" @click="submitBulkVenues">批次送出</el-button>
      </el-collapse-item>

      <!-- 已知場地 -->
      <el-collapse-item name="known">
        <template #title>
          <span class="collapse-title">已知場地</span>
          <el-button size="small" style="margin-left:12px" @click.stop="loadKnownVenues">重新整理</el-button>
        </template>
        <p class="sub-text">只列出資料庫裡已經出現過或已登記的場次場地，不是官方完整場次紀錄。</p>
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
        <div v-if="filteredKnownVenues.length === 0" class="empty">沒有資料</div>
        <el-table table-layout="auto" v-else :data="filteredKnownVenues" stripe size="small" max-height="500" :row-style="rowStyle">
          <el-table-column label="團體" min-width="70">
            <template #default="{ row }">
              <span :style="{ color: GROUP_COLORS[row.group], fontWeight: 500 }">{{ groupLabel(row.group) }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="single_number" label="單曲號" min-width="60" />
          <el-table-column prop="single_name" label="單曲名稱" min-width="200" />
          <el-table-column prop="event_date" label="日期" min-width="100" />
          <el-table-column label="場地" min-width="200">
            <template #default="{ row }">
              <el-input v-if="row._editing" v-model="row._input" size="small" style="width:100%" />
              <span v-else>{{ row.venue }}</span>
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
import { getAdminVenueIssues, fixVenue, bulkSetVenues, getAdminKnownVenues } from '../api/index'

const openSections = ref(['issues', 'bulk', 'known'])
const venueIssues = ref([])

const VENUE_EXAMPLE = 'nogizaka46,33,2023/11/19,幕張メッセ\nsakurazaka46,8,2024/4/7,京都パルスプラザ'

async function copyExample(text) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已複製到剪貼簿')
  } catch {
    ElMessage.error('複製失敗，請手動選取複製')
  }
}

async function loadVenueIssues() {
  try {
    const res = await getAdminVenueIssues()
    venueIssues.value = (res.data ?? []).map(item => ({
      ...item,
      _input:   item.suggested_venue || '',
      _loading: false,
    }))
  } catch {}
}

async function fixVenueRow(row) {
  if (!row._input.trim()) {
    ElMessage.warning('請輸入場地名稱')
    return
  }
  row._loading = true
  try {
    const res = await fixVenue(row.group, row.single_number, row.event_date, row._input.trim())
    ElMessage.success(`已更新 ${res.data.updated} 筆`)
    await loadVenueIssues()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '更新失敗')
  } finally {
    row._loading = false
  }
}

const GROUP_LABELS = { nogizaka46: '乃木坂46', sakurazaka46: '櫻坂46', hinatazaka46: '日向坂46' }
const GROUP_COLORS = { nogizaka46: '#9333ea', sakurazaka46: '#ec4899', hinatazaka46: '#0ea5e9' }
function groupLabel(g) {
  return GROUP_LABELS[g] || g || '—'
}
function rowStyle({ row }) {
  return { color: GROUP_COLORS[row.group] }
}

const bulkVenueText = ref('')
const bulkLoading    = ref(false)

async function submitBulkVenues() {
  const lines = bulkVenueText.value.split('\n').map(l => l.trim()).filter(Boolean)
  if (lines.length === 0) {
    ElMessage.warning('請輸入至少一行')
    return
  }

  const venueMap = new Map()
  for (const line of lines) {
    const parts = line.split(',')
    if (parts.length < 4) {
      ElMessage.error(`格式錯誤：${line}`)
      return
    }
    const group = parts[0].trim()
    const singleNumber = parseInt(parts[1].trim())
    const eventDate = parts[2].trim()
    const venue = parts.slice(3).join(',').trim()
    if (!GROUP_LABELS[group]) {
      ElMessage.error(`團體代碼不正確：${group}`)
      return
    }
    if (isNaN(singleNumber) || !eventDate || !venue) {
      ElMessage.error(`格式錯誤：${line}`)
      return
    }
    const key = `${group}:${singleNumber}:${eventDate}`
    venueMap.set(key, { group, single_number: singleNumber, event_date: eventDate, venue })
  }

  const duplicateCount = lines.length - venueMap.size
  const venues = Array.from(venueMap.values())

  bulkLoading.value = true
  try {
    const res = await bulkSetVenues(venues)
    const dupMsg = duplicateCount > 0 ? `（已自動排除 ${duplicateCount} 筆重複，取最後一筆）` : ''
    ElMessage.success(`已登記 ${res.data.applied} 筆，回填更新 ${res.data.updated} 筆${dupMsg}`)
    bulkVenueText.value = ''
    await loadVenueIssues()
    await loadKnownVenues()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '批次送出失敗')
  } finally {
    bulkLoading.value = false
  }
}

const knownVenues = ref([])
const knownGroupFilter = ref('')
const filteredKnownVenues = computed(() =>
  knownGroupFilter.value ? knownVenues.value.filter(v => v.group === knownGroupFilter.value) : knownVenues.value
)

const SOURCE_LABELS = { correction: '已登記修正', records: '全握紀錄推測' }
function sourceLabel(s) {
  return SOURCE_LABELS[s] || s || '—'
}

async function loadKnownVenues() {
  try {
    const res = await getAdminKnownVenues()
    knownVenues.value = (res.data ?? []).map(v => ({
      ...v,
      _editing: false,
      _input: v.venue,
      _loading: false,
    }))
  } catch {}
}

function startEdit(row) {
  row._editing = true
  row._input = row.venue
}

function cancelEdit(row) {
  row._editing = false
  row._input = row.venue
}

async function saveKnown(row) {
  const newVenue = row._input.trim()
  if (!newVenue) { cancelEdit(row); return }
  if (newVenue === row.venue) { cancelEdit(row); return }
  row._loading = true
  try {
    const res = await fixVenue(row.group, row.single_number, row.event_date, newVenue)
    ElMessage.success(`已更新 ${res.data.updated} 筆`)
    await Promise.all([loadKnownVenues(), loadVenueIssues()])
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '更新失敗')
    row._loading = false
    row._editing = false
  }
}

onMounted(() => {
  loadVenueIssues()
  loadKnownVenues()
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
