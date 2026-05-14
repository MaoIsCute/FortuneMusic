<template>
  <div class="page">
    <h1 class="page-title">📊 總覽</h1>
    <div class="stats-grid" v-if="stats">
      <div class="stat-card">
        <div class="stat-label">總應募數</div>
        <div class="stat-value">{{ stats.total_applied }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">總中選數</div>
        <div class="stat-value">{{ stats.total_won }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">總中選率</div>
        <div class="stat-value highlight">{{ stats.win_rate }}%</div>
      </div>
    </div>
    <div class="member-list">
      <h2>成員選擇</h2>
      <div class="member-grid">
        <div
          v-for="m in members"
          :key="m.member_name"
          class="member-chip"
          @click="goToMember(m.member_name)"
        >
          {{ m.member_name }}
          <span class="rate">{{ m.win_rate }}%</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getStats, getStatsByMember } from '../api/index'

const router = useRouter()
const stats = ref(null)
const members = ref([])

onMounted(async () => {
  const [s, m] = await Promise.all([getStats(), getStatsByMember()])
  stats.value = s.data
  members.value = m.data
})

function goToMember(name) {
  router.push({ name: 'Member', params: { name } })
}
</script>

<style scoped>
.stats-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 32px;
}
.stat-card {
  background: white;
  border-radius: 12px;
  padding: 24px;
  text-align: center;
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
}
.stat-label { color: #888; font-size: 14px; margin-bottom: 8px; }
.stat-value { font-size: 32px; font-weight: bold; }
.stat-value.highlight { color: var(--color-primary); }
.member-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 16px;
}
.member-chip {
  padding: 8px 16px;
  background: white;
  border-radius: 20px;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(0,0,0,0.08);
  transition: transform 0.2s;
  display: flex;
  gap: 8px;
  align-items: center;
}
.member-chip:hover { transform: translateY(-2px); }
.rate { color: var(--color-primary); font-weight: bold; }
</style>
