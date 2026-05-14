<template>
  <div class="login-page">
    <div class="login-card">
      <div class="logo">🎵 Fortune Tracker</div>
      <p class="subtitle">乃木坂46 抽選統計</p>
      <a :href="googleLoginUrl" class="google-btn">
        <img src="https://www.google.com/favicon.ico" width="18" />
        Google でログイン
      </a>
    </div>
  </div>
</template>

<script setup>
const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080'
const googleLoginUrl = apiUrl + '/auth/google'
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-gradient);
}
.login-card {
  background: white;
  border-radius: 16px;
  padding: 48px 40px;
  text-align: center;
  box-shadow: 0 20px 60px rgba(0,0,0,0.15);
  min-width: 320px;
}
.logo { font-size: 28px; font-weight: bold; margin-bottom: 8px; }
.subtitle { color: #666; margin-bottom: 32px; }
.google-btn {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  padding: 12px 24px;
  border: 1px solid #ddd;
  border-radius: 8px;
  text-decoration: none;
  color: #333;
  font-size: 15px;
  transition: background 0.2s;
}
.google-btn:hover { background: #f5f5f5; }
</style>
