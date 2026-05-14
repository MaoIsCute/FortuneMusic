<template>
  <div :class="['app', themeStore.isDark ? 'dark' : '']">
    <NavBar v-if="auth.isLoggedIn" />
    <main class="main">
      <router-view />
    </main>
  </div>
</template>

<script setup>
import NavBar from './components/NavBar.vue'
import { useThemeStore } from './stores/theme'
import { useAuthStore } from './stores/auth'

const themeStore = useThemeStore()
const auth = useAuthStore()
</script>

<style>
:root {
  --color-primary: #6366f1;
  --color-secondary: #818cf8;
  --color-gradient: linear-gradient(135deg, #6366f1 0%, #818cf8 100%);
}
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: 'Inter', sans-serif; background: #f5f7fa; }
.app.dark body { background: #1a1a2e; color: #eee; }
.main { padding: 24px; max-width: 1200px; margin: 0 auto; }
.page-title { font-size: 24px; font-weight: bold; margin-bottom: 24px; }
.page { padding: 8px 0; }
</style>
