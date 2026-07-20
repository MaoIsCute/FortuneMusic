<template>
  <MaintenanceView v-if="maintenance" :message="maintenanceMsg" />
  <div v-else :class="['app', themeStore.isDark ? 'dark' : '']">
    <ImpersonateBanner />
    <div :style="impersonate.user ? 'padding-top: 40px' : ''">
      <NavBar v-if="auth.isLoggedIn" />
      <main class="main">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import NavBar from './components/NavBar.vue'
import ImpersonateBanner from './components/ImpersonateBanner.vue'
import MaintenanceView from './views/MaintenanceView.vue'
import { useThemeStore } from './stores/theme'
import { useAuthStore } from './stores/auth'
import { useImpersonateStore } from './stores/impersonate'
import { getMe } from './api/index'

const themeStore = useThemeStore()
const auth = useAuthStore()
const impersonate = useImpersonateStore()

const maintenance    = ref(false)
const maintenanceMsg = ref('系統維護中，請稍後再試。')

const STATUS_URL = import.meta.env.DEV
  ? '/status.json'
  : 'https://raw.githubusercontent.com/MaoIsCute/FortuneMusic/main/status.json'

onMounted(async () => {
  try {
    const res = await fetch(STATUS_URL + '?t=' + Date.now())
    const data = await res.json()
    if (data.maintenance) {
      maintenance.value    = true
      maintenanceMsg.value = data.message || maintenanceMsg.value
      return
    }
  } catch {}

  if (auth.isLoggedIn && !auth.user) {
    try {
      const res = await getMe()
      auth.setUser(res.data)
    } catch {}
  }
})
</script>

<style>
:root {
  --color-primary: #6366f1;
  --color-secondary: #818cf8;
  --color-gradient: linear-gradient(135deg, #6366f1 0%, #818cf8 100%);
}
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: 'Inter', sans-serif; background: #f5f7fa; color: #303133; }
html.dark body { background: #1a1a2e; color: #e5eaf3; }
html.dark {
  --el-text-color-primary:     #e8eaf0;
  --el-text-color-regular:     #d4d8e3;
  --el-text-color-secondary:   #b8bfcc;
  --el-text-color-placeholder: #9aa3b5;
  --el-bg-color:               #1e2030;
  --el-bg-color-page:          #1a1a2e;
  --el-bg-color-overlay:       #252840;
  --el-border-color:           #3a3f5c;
  --el-border-color-light:     #2e3450;
  --el-fill-color:             #252840;
  --el-fill-color-light:       #1e2030;
}
.main { padding: 24px; max-width: 1200px; margin: 0 auto; }
.page-title { font-size: 24px; font-weight: bold; margin-bottom: 24px; }
.page { padding: 8px 0; }
/* 每個頁面元件自己的 .page { background: #f5f7fa } 在深色模式下沒有對應覆寫，
   會讓整個頁面背景卡在淺色，標題文字（吃 body 的深色模式字色）因此變成
   淺色字疊淺色底、幾乎看不見——在這裡統一蓋一次，不用每個頁面各自加 */
html.dark .page { background: #1a1a2e; }
</style>
