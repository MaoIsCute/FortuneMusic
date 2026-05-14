<template>
  <div class="page">
    <h1 class="page-title">📋 抽選紀錄</h1>
    <div class="filters">
      <el-select v-model="filterMember" placeholder="選擇成員" clearable @change="filter">
        <el-option v-for="m in memberList" :key="m" :label="m" :value="m" />
      </el-select>
    </div>
    <el-table :data="filtered" stripe>
      <el-table-column prop="member_name" label="成員" />
      <el-table-column prop="event_date" label="日期" />
      <el-table-column prop="session" label="部數" />
      <el-table-column prop="applied_count" label="應募數" />
      <el-table-column prop="won_count" label="中選數" />
      <el-table-column prop="scraped_at" label="爬取時間" />
    </el-table>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getRecords } from '../api/index'

const records = ref([])
const filterMember = ref('')

onMounted(async () => {
  const res = await getRecords()
  records.value = res.data
})

const memberList = computed(() => [...new Set(records.value.map(r => r.member_name))])
const filtered = computed(() =>
  filterMember.value ? records.value.filter(r => r.member_name === filterMember.value) : records.value
)
</script>

<style scoped>
.filters { margin-bottom: 20px; }
</style>
