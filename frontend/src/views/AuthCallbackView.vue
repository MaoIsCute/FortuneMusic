<template>
  <div class="callback-page">
    <div class="callback-card">
      <div v-if="error" class="error">
        <p>登入失敗：{{ error }}</p>
        <a href="/">返回登入頁</a>
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
import { getMe } from '../api/index'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const error = ref('')

onMounted(async () => {
  const token = route.query.token
  if (!token) {
    error.value = '未取得 token'
    return
  }
  auth.setToken(token)
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
</style>
