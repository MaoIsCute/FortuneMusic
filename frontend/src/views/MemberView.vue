<template>
  <div class="page">
    <div class="member-header">
      <h1>{{ name }}</h1>
    </div>
    <div class="tabs">
      <button :class="['tab', tab === 'date' ? 'active' : '']" @click="tab = 'date'">日期別</button>
      <button :class="['tab', tab === 'session' ? 'active' : '']" @click="tab = 'session'">部數別</button>
      <button :class="['tab', tab === 'records' ? 'active' : '']" @click="tab = 'records'">紀錄</button>
    </div>
    <div v-if="tab === 'date'" class="table-wrap">
      <el-table :data="byDate" stripe>
        <el-table-column prop="event_date" label="日期" />
        <el-table-column prop="total_applied" label="應募數" />
        <el-table-column prop="total_won" label="中選數" />
        <el-table-column prop="win_rate" label="中選率">
          <template #default="{ row }">
            <span class="rate">{{ row.win_rate }}%</span>
          </template>
        </el-table-column>
      </el-table>
    </div>
    <div v-if="tab === 'session'" class="table-wrap">
      <el-table :data="bySession" stripe>
        <el-table-column prop="session" label="部數" />
        <el-table-column prop="total_applied" label="應募數" />
        <el-table-column prop="total_won" label="中選數" />
        <el-table-column prop="win_rate" label="中選率">
          <template #default="{ row }">
            <span class="rate">{{ row.win_rate }}%</span>
          </template>
        </el-table-column>
      </el-table>
    </div>
    <div v-if="tab === 'records'" class="table-wrap">
      <el-table :data="records" stripe>
        <el-table-column prop="event_date" label="日期" />
        <el-table-column prop="session" label="部數" />
        <el-table-column prop="applied_count" label="應募" />
        <el-table-column prop="won_count" label="中選" />
        <el-table-column prop="scraped_at" label="爬取時間" />
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useThemeStore } from '../stores/theme'
import { getStatsByDate, getStatsBySession, getRecords } from '../api/index'

const route = useRoute()
const themeStore = useThemeStore()
const name = ref(route.params.name)
const tab = ref('date')
const byDate = ref([])
const bySession = ref([])
const records = ref([])

async function loadData() {
  themeStore.setMember(name.value)
  const [d, s, r] = await Promise.all([
    getStatsByDate(),
    getStatsBySession(),
    getRecords(),
  ])
  byDate.value = d.data.filter(x => x.member_name === name.value)
  bySession.value = s.data.filter(x => x.member_name === name.value)
  records.value = r.data.filter(x => x.member_name === name.value)
}

onMounted(loadData)
watch(() => route.params.name, (n) => { name.value = n; loadData() })
</script>

<style scoped>
.member-header {
  padding: 24px;
  background: var(--color-gradient);
  border-radius: 12px;
  color: white;
  margin-bottom: 24px;
}
.member-header h1 { margin: 0; font-size: 28px; }
.tabs { display: flex; gap: 8px; margin-bottom: 20px; }
.tab {
  padding: 8px 20px;
  border-radius: 20px;
  border: 2px solid var(--color-primary);
  background: white;
  color: var(--color-primary);
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s;
}
.tab.active { background: var(--color-primary); color: white; }
.rate { color: var(--color-primary); font-weight: bold; }
</style>
