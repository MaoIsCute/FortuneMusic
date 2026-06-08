<template>
  <div class="empty-card">
    <div class="empty-icon">📭</div>
    <p class="empty-title">還沒有任何資料</p>
    <p class="empty-sub">點 Chrome 右上角的同步工具圖示，再點「同步」開始抓取你的抽選紀錄。</p>
  </div>
</template>

<style scoped>
.empty-card {
  background: white;
  border-radius: 16px;
  padding: 48px 32px;
  text-align: center;
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
  margin-bottom: 32px;
}
.empty-icon  { font-size: 48px; margin-bottom: 16px; }
.empty-title { font-size: 20px; font-weight: bold; color: #333; margin: 0 0 12px; }
.empty-sub   { color: #888; line-height: 1.7; margin: 0; }
html.dark .empty-card  { background: #1e2030; }
html.dark .empty-title { color: #e8eaf0; }
html.dark .empty-sub   { color: #9aa3b5; }
</style>
