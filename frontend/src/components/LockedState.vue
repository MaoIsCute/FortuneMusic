<template>
  <div class="locked-card">
    <div class="locked-icon">🔒</div>
    <p class="locked-title">{{ title }}</p>
    <p class="locked-sub">{{ sub }}</p>
    <el-button type="primary" @click="router.push('/scrape')">前往同步工具</el-button>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'

defineProps({
  title: { type: String, required: true },
  sub:   { type: String, required: true },
})

const router = useRouter()
</script>

<style scoped>
.locked-card {
  background: white;
  border-radius: 16px;
  padding: 48px 32px;
  text-align: center;
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
}
.locked-icon  { font-size: 48px; margin-bottom: 16px; }
.locked-title { font-size: 20px; font-weight: bold; color: #333; margin: 0 0 12px; }
.locked-sub   { color: #888; line-height: 1.7; margin: 0 0 20px; }
html.dark .locked-card  { background: #1e2030; box-shadow: 0 2px 12px rgba(0,0,0,0.4); }
html.dark .locked-title { color: #e8eaf0; }
html.dark .locked-sub   { color: #9aa3b5; }
</style>
