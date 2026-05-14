<template>
  <nav class="navbar">
    <div class="brand" @click="router.push('/dashboard')">🎵 Fortune Tracker</div>
    <div class="links">
      <router-link to="/dashboard">總覽</router-link>
      <router-link to="/records">紀錄</router-link>
      <router-link to="/scrape">爬蟲</router-link>
    </div>
    <div class="actions">
      <el-switch v-model="isDark" @change="themeStore.toggleDark()" active-text="🌙" inactive-text="☀️" />
      <span class="user">{{ auth.user?.name }}</span>
      <el-button size="small" @click="auth.logout(); router.push('/')">登出</el-button>
    </div>
  </nav>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useThemeStore } from '../stores/theme'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const themeStore = useThemeStore()
const auth = useAuthStore()
const isDark = computed(() => themeStore.isDark)
</script>

<style scoped>
.navbar {
  display: flex;
  align-items: center;
  padding: 0 24px;
  height: 60px;
  background: var(--color-gradient);
  color: white;
  gap: 24px;
}
.brand { font-weight: bold; font-size: 18px; cursor: pointer; }
.links { display: flex; gap: 16px; flex: 1; }
.links a { color: white; text-decoration: none; opacity: 0.85; }
.links a:hover, .links a.router-link-active { opacity: 1; font-weight: bold; }
.actions { display: flex; align-items: center; gap: 12px; }
.user { font-size: 14px; }
</style>
