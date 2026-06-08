<template>
  <div class="callback-page">
    <div class="callback-card">
      <div v-if="error" class="error">
        <p>登入失敗：{{ error }}</p>
        <p class="redirect-hint">{{ countdown }} 秒後自動返回登入頁⋯</p>
        <a href="/">立即返回</a>
      </div>
      <div v-else>
        <p>登入中，請稍候⋯</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { exchangeToken, getMe, getScrapeToken } from '../api/index'

const EXTENSION_ID = 'gdclpkfeiocedicokoenhconeoocigeh'

async function tryAutoLink() {
  if (!window.chrome?.runtime?.sendMessage) return
  try {
    const res = await getScrapeToken()
    const backendUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080'
    await new Promise((resolve) => {
      chrome.runtime.sendMessage(
        EXTENSION_ID,
        { type: 'FORTUNE_SETUP', token: res.data.scrape_token, backendUrl },
        () => { chrome.runtime.lastError; resolve() }
      )
    })
  } catch {}
}

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const error = ref('')
const countdown = ref(3)

function redirectToLogin() {
  let t = countdown.value
  const timer = setInterval(() => {
    t--
    countdown.value = t
    if (t <= 0) {
      clearInterval(timer)
      router.replace('/')
    }
  }, 1000)
}

onMounted(async () => {
  const code = route.query.code
  if (!code) {
    error.value = '未取得授權碼'
    redirectToLogin()
    return
  }
  try {
    const res = await exchangeToken(code)
    auth.setToken(res.data.token)
    auth.setRefreshToken(res.data.refresh_token)
  } catch {
    error.value = '登入失敗，請重試'
    redirectToLogin()
    return
  }
  await Promise.all([
    getMe().then(res => auth.setUser(res.data)).catch(() => {}),
    tryAutoLink(),
  ])
  const redirect = localStorage.getItem('redirectAfterLogin') || '/dashboard'
  localStorage.removeItem('redirectAfterLogin')
  router.replace(redirect)
})
</script>

<style scoped>
.callback-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-gradient);
}
.callback-card {
  background: white;
  border-radius: 16px;
  padding: 48px 40px;
  text-align: center;
  font-size: 16px;
  color: #555;
}
.error { color: #e53e3e; }
.error a { color: var(--color-primary); }
.redirect-hint { font-size: 13px; color: #999; margin: 8px 0; }
</style>
