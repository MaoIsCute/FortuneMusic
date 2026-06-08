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
            <div class="step-install">
              <a href="https://github.com/MaoIsCute/FortuneMusic/raw/main/FTExtension.zip" target="_blank" class="download-btn">
                ⬇️ 下載同步工具
              </a>
              <ol class="install-steps">
                <li>下載後解壓縮 zip 檔案</li>
                <li>Chrome 網址列輸入
                  <span class="copy-row">
                    <code>chrome://extensions/</code>
                    <button class="copy-btn" @click="copy">{{ copied ? '已複製！' : '複製' }}</button>
                  </span>
                </li>
                <li>右上角開啟「開發人員模式」</li>
                <li>點「載入未封裝項目」→ 選剛才解壓縮的資料夾</li>
              </ol>
            </div>
          </div>
        </div>
        <div class="step">
          <div class="step-num">2</div>
          <div class="step-text">
            <div class="step-title">連結你的帳號</div>
            <div class="step-sub">安裝完成後，點下方按鈕讓同步工具認識你的帳號</div>
            <div class="step-action">
              <el-button
                v-if="linkStatus !== 'success'"
                type="primary"
                :loading="linking"
                @click="linkExtension"
              >連結帳號</el-button>
              <span v-if="linkStatus === 'success'" class="link-success">✅ 連結成功！</span>
              <span v-if="linkStatus === 'error'" class="link-error">{{ linkError }}</span>
            </div>
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

      <el-button
        v-if="linkStatus === 'success'"
        type="primary"
        size="large"
        class="goto-btn"
        @click="router.replace('/dashboard')"
      >前往主頁 →</el-button>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { getScrapeToken } from '../api/index'

const router = useRouter()
const EXTENSION_ID = 'gdclpkfeiocedicokoenhconeoocigeh'

const copied    = ref(false)
const linking   = ref(false)
const linkStatus = ref('')
const linkError  = ref('')

function copy() {
  navigator.clipboard.writeText('chrome://extensions/')
  copied.value = true
  setTimeout(() => { copied.value = false }, 2000)
}

async function linkExtension() {
  linking.value   = true
  linkStatus.value = ''
  linkError.value  = ''
  try {
    const res = await getScrapeToken()
    const backendUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080'
    await new Promise((resolve, reject) => {
      if (!window.chrome?.runtime?.sendMessage) { reject(new Error('not_found')); return }
      chrome.runtime.sendMessage(
        EXTENSION_ID,
        { type: 'FORTUNE_SETUP', token: res.data.scrape_token, backendUrl },
        (response) => {
          const err = chrome.runtime.lastError
          if (err) reject(new Error(err.message))
          else if (response?.success) resolve()
          else reject(new Error('failed'))
        }
      )
    })
    linkStatus.value = 'success'
  } catch (e) {
    linkStatus.value = 'error'
    const msg = e.message || ''
    linkError.value = msg.includes('Could not establish connection') || msg === 'not_found'
      ? '找不到同步工具，請確認已依步驟 1 安裝並啟用。'
      : '連結失敗，請重試。'
  } finally {
    linking.value = false
  }
}
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

.step-install { margin-top: 10px; }
.download-btn {
  display: inline-block;
  padding: 6px 16px;
  background: var(--color-primary);
  color: white;
  border-radius: 6px;
  text-decoration: none;
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 10px;
}
.download-btn:hover { opacity: 0.85; }
.install-steps {
  margin: 0;
  padding-left: 18px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 13px;
  color: #666;
}
.install-steps code {
  background: #f0f0f0;
  padding: 1px 5px;
  border-radius: 4px;
  font-size: 12px;
}

html.dark .setup-card   { background: #1e2030; box-shadow: 0 4px 24px rgba(0,0,0,0.4); }
html.dark .setup-title  { color: #e8eaf0; }
html.dark .setup-desc   { color: #9aa3b5; }
html.dark .step-title   { color: #d4d8e3; }
html.dark .step-sub     { color: #6b7490; }
html.dark .install-steps { color: #9aa3b5; }
html.dark .install-steps code { background: #2e3450; color: #d4d8e3; }
.step-action { margin-top: 10px; display: flex; flex-direction: column; gap: 6px; }
.link-success { font-size: 14px; color: #52c41a; font-weight: 600; }
.link-error   { font-size: 13px; color: #ff4d4f; }
.copy-row { display: inline-flex; align-items: center; gap: 6px; }
.copy-btn {
  padding: 1px 8px;
  font-size: 11px;
  border: 1px solid #ccc;
  border-radius: 4px;
  background: white;
  cursor: pointer;
  color: #555;
}
.copy-btn:hover { background: #f0f0f0; }
html.dark .copy-btn { background: #2e3450; border-color: #3a3f5c; color: #b8bfcc; }
</style>
