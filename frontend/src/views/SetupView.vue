<template>
  <div class="setup-page">
    <div class="setup-card">
      <div class="setup-icon">👋</div>
      <h1 class="setup-title">歡迎使用 Fortune Tracker</h1>
      <p class="setup-desc">目前還沒有任何抽選資料。<br>請依照以下步驟完成設定，就能開始同步你的抽選紀錄。</p>

      <div class="steps">
        <div class="step">
          <div class="step-num">1</div>
          <div class="step-text">
            <div class="step-title">安裝同步工具</div>
            <div class="step-sub">將同步工具安裝到你的 Chrome 瀏覽器</div>
          </div>
        </div>
        <div class="step">
          <div class="step-num">2</div>
          <div class="step-text">
            <div class="step-title">連結你的帳號</div>
            <div class="step-sub">點下方按鈕，讓同步工具認識你的帳號</div>
          </div>
        </div>
        <div class="step">
          <div class="step-num">3</div>
          <div class="step-text">
            <div class="step-title">開始同步資料</div>
            <div class="step-sub">點 Chrome 右上角的同步工具圖示，點「同步」開始抓取</div>
          </div>
        </div>
      </div>

      <router-link to="/scrape">
        <el-button type="primary" size="large" class="goto-btn">
          開始設定 →
        </el-button>
      </router-link>
    </div>
  </div>
</template>

<script setup>
</script>

<style scoped>
.setup-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.setup-card {
  background: white;
  border-radius: 16px;
  padding: 48px 40px;
  max-width: 520px;
  width: 100%;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.08);
  text-align: center;
}

.setup-icon {
  font-size: 48px;
  margin-bottom: 16px;
}

.setup-title {
  font-size: 24px;
  font-weight: bold;
  color: #222;
  margin: 0 0 12px;
}

.setup-desc {
  color: #666;
  line-height: 1.7;
  margin: 0 0 32px;
}

.steps {
  display: flex;
  flex-direction: column;
  gap: 20px;
  margin-bottom: 36px;
  text-align: left;
}

.step {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.step-num {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--color-primary);
  color: white;
  font-weight: bold;
  font-size: 15px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.step-title {
  font-weight: 600;
  color: #333;
  margin-bottom: 4px;
}

.step-sub {
  font-size: 13px;
  color: #888;
  line-height: 1.5;
}

.goto-btn {
  width: 100%;
  font-size: 16px;
  height: 48px;
}

html.dark .setup-card   { background: #1e2030; box-shadow: 0 4px 24px rgba(0,0,0,0.4); }
html.dark .setup-title  { color: #e8eaf0; }
html.dark .setup-desc   { color: #9aa3b5; }
html.dark .step-title   { color: #d4d8e3; }
html.dark .step-sub     { color: #6b7490; }
</style>
