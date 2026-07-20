<template>
  <nav class="navbar">
    <div class="brand" @click="router.push('/dashboard')">🎵 Fortune Tracker</div>
    <div class="links">
      <router-link to="/dashboard">全員統計</router-link>

      <div class="nav-group">
        <a :class="{ 'router-link-active': isRecordsActive }" @click.stop="toggle('records')">個握 ▾</a>
        <div v-show="openMenu === 'records'" class="nav-menu">
          <a @click="go('/records')">個握紀錄</a>
          <a @click="go('/spending')">個握花費</a>
          <a @click="go('/records/analysis')">個握分析</a>
        </div>
      </div>

      <div class="nav-group">
        <a :class="{ 'router-link-active': isFullActive }" @click.stop="toggle('full')">全握 ▾</a>
        <div v-show="openMenu === 'full'" class="nav-menu">
          <a @click="go('/full')">全握紀錄</a>
          <a @click="go('/full/spending')">全握花費</a>
          <a @click="go('/full/analysis')">全握分析</a>
          <a @click="go('/full/sign-events')">簽名會紀錄</a>
        </div>
      </div>

      <router-link to="/scrape">同步工具</router-link>

      <div v-if="isAdmin" class="nav-group">
        <a :class="{ 'router-link-active': isAdminActive }" @click.stop="toggle('admin')">管理 ▾</a>
        <div v-show="openMenu === 'admin'" class="nav-menu">
          <a @click="go('/admin/users')">使用者管理</a>
          <a @click="go('/admin/maintenance')">資料維護</a>
          <a @click="go('/admin/titles')">單曲名稱</a>
          <a @click="go('/admin/venues')">場地管理</a>
          <a @click="go('/admin/sign-events')">簽名會紀錄</a>
        </div>
      </div>
    </div>
    <div class="actions">
      <el-switch v-model="isDark" @change="themeStore.toggleDark()" active-text="🌙" inactive-text="☀️" />
      <span class="version">v{{ APP_VERSION }}</span>
      <span class="user">{{ auth.user?.name }}</span>
      <el-button size="small" @click="auth.logout(); router.push('/')">登出</el-button>
    </div>
  </nav>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useThemeStore } from '../stores/theme'
import { useAuthStore } from '../stores/auth'

const APP_VERSION = '1.6'

const router = useRouter()
const route  = useRoute()
const themeStore = useThemeStore()
const auth = useAuthStore()
const isDark = computed(() => themeStore.isDark)
const isAdmin = computed(() => !!auth.user?.is_admin)
const isRecordsActive = computed(() => ['/records', '/spending', '/records/analysis'].includes(route.path))
const isFullActive    = computed(() => ['/full', '/full/spending', '/full/analysis', '/full/sign-events'].includes(route.path))
const isAdminActive   = computed(() => route.path.startsWith('/admin'))

const openMenu = ref(null)

function toggle(name) {
  openMenu.value = openMenu.value === name ? null : name
}

function go(path) {
  router.push(path)
  openMenu.value = null
}

function closeMenu() { openMenu.value = null }
onMounted(() => document.addEventListener('click', closeMenu))
onUnmounted(() => document.removeEventListener('click', closeMenu))
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
.links { display: flex; gap: 16px; flex: 1; align-items: center; }
.links a { color: white; text-decoration: none; opacity: 0.85; cursor: pointer; user-select: none; }
.links a:hover, .links a.router-link-active { opacity: 1; font-weight: bold; }
.nav-group { position: relative; }
.nav-menu {
  position: absolute;
  top: calc(100% + 8px);
  left: 0;
  background: white;
  border-radius: 6px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.15);
  min-width: 120px;
  z-index: 1000;
  overflow: hidden;
}
.nav-menu a {
  display: block;
  padding: 9px 16px;
  color: #333 !important;
  opacity: 1 !important;
  font-weight: normal !important;
  font-size: 14px;
}
.nav-menu a:hover { background: #f5f7fa; }
html.dark .nav-menu       { background: #252840; box-shadow: 0 2px 12px rgba(0,0,0,0.4); }
html.dark .nav-menu a     { color: #d4d8e3 !important; }
html.dark .nav-menu a:hover { background: #2c3154; }
.actions { display: flex; align-items: center; gap: 12px; }
.version { font-size: 11px; opacity: 0.6; }
.user { font-size: 14px; }
</style>
