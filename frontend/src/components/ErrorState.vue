<template>
  <div class="error-card">
    <div class="error-icon">⚠️</div>
    <p class="error-title">資料載入失敗</p>
    <p class="error-sub">伺服器暫時無法連線，請稍後再試。<br>如問題持續請聯繫管理員。</p>
    <el-button type="primary" @click="reload">重新整理頁面</el-button>
  </div>
</template>

<script setup>
function reload() { window.location.reload() }
</script>

<style scoped>
.error-card {
  background: white;
  border-radius: 16px;
  padding: 48px 32px;
  text-align: center;
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
  margin-bottom: 32px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}
.error-icon  { font-size: 48px; }
.error-title { font-size: 20px; font-weight: bold; color: #333; margin: 0; }
.error-sub   { color: #888; line-height: 1.7; margin: 0; }
html.dark .error-card  { background: #1e2030; }
html.dark .error-title { color: #e8eaf0; }
html.dark .error-sub   { color: #9aa3b5; }
</style>
