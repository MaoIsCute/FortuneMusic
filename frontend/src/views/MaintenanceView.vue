<template>
  <div class="maintenance">
    <div class="card">
      <div class="icon">🔧</div>
      <h1>系統維護中</h1>
      <p>{{ message }}</p>
    </div>
  </div>
</template>

<script setup>
defineProps({ message: { type: String, default: '系統維護中，請稍後再試。' } })
</script>

<style scoped>
.maintenance {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f7fa;
}
.card {
  text-align: center;
  background: #fff;
  border-radius: 16px;
  padding: 56px 48px;
  box-shadow: 0 4px 24px rgba(0,0,0,0.08);
  max-width: 420px;
  width: 90%;
}
.icon { font-size: 56px; margin-bottom: 20px; }
h1 { font-size: 24px; font-weight: 700; color: #303133; margin-bottom: 12px; }
p  { font-size: 15px; color: #909399; line-height: 1.6; }
</style>
