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
import { exchangeToken, getMe } from '../api/index'

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
  try {
    const res = await getMe()
    auth.setUser(res.data)
  } catch {
    // token 有效但取不到 user，不影響登入
  }
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
