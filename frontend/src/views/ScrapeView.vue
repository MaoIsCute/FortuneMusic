<template>
  <div class="page">
    <h1 class="page-title">同步工具設定</h1>
    <div v-if="outdated" class="update-banner">
      ⚠️ 你的同步工具版本較舊（v{{ installedVersion }}，最新版 v{{ latestVersion }}），建議重新下載安裝最新版。
      <a href="https://github.com/MaoIsCute/FortuneMusic/raw/main/FTExtension.zip" target="_blank">立即更新 →</a>
    </div>
    <!-- 連結前 -->
    <div v-if="statusType !== 'success'" class="setup-card">
      <p class="desc">
        {{ firstTime ? '目前還沒有任何抽選資料，請先安裝同步工具、連結帳號，就能開始同步你的抽選紀錄。' : '點擊下方按鈕，自動將你的帳號與同步工具連結，完成一鍵設定。' }}
      </p>
      <el-button type="primary" size="large" :loading="loading" @click="authorize">
        連結同步工具
      </el-button>
      <p v-if="statusMsg" :class="['status-msg', statusType]">{{ statusMsg }}</p>
      <div v-if="statusType === 'error' || firstTime" class="install-guide">
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
          <li>安裝完成後，再點一次「連結同步工具」</li>
        </ol>
      </div>
    </div>

    <!-- 連結成功後：引導下一步 -->
    <div v-else class="setup-card">
      <div class="success-icon">✅</div>
      <h2 class="success-title">連結成功！</h2>
      <p class="desc">最後一步：點 Chrome 右上角的同步工具圖示，再點「同步」，就會開始抓取你的資料。</p>
      <div class="hint-box">
        <p>抓取完成後，點「個握 ▾ → 個握分析」就能看到你的統計資料了。</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { getExtensionVersion } from '../utils/extension'

const copied = ref(false)
function copy() {
  navigator.clipboard.writeText('chrome://extensions/')
  copied.value = true
  setTimeout(() => { copied.value = false }, 2000)
}
import { getScrapeToken } from '../api/index'

const EXTENSION_ID = 'gdclpkfeiocedicokoenhconeoocigeh'

const router = useRouter()
const route  = useRoute()
// 從「個握分析」偵測到擴充功能未安裝時會帶 ?firstTime=1 導過來（原本是導去一個獨立的
// /setup 歡迎頁，內容跟這裡幾乎重複，改成同一個頁面用這個參數決定要不要主動展開安裝步驟，
// 不用先讓使用者點過一次「連結同步工具」失敗才看得到，見 CLAUDE.md）
const firstTime = computed(() => route.query.firstTime === '1')
const loading   = ref(false)
const statusMsg = ref('')
const statusType = ref('')

// 已安裝的擴充功能版本落後 status.json 記錄的最新版本時，提醒使用者更新
const installedVersion = ref(null)
const latestVersion    = ref(null)
const outdated = computed(() =>
  installedVersion.value && latestVersion.value && parseFloat(installedVersion.value) < parseFloat(latestVersion.value)
)

const STATUS_URL = import.meta.env.DEV
  ? '/status.json'
  : 'https://raw.githubusercontent.com/MaoIsCute/FortuneMusic/main/status.json'

onMounted(async () => {
  installedVersion.value = await getExtensionVersion()
  if (!installedVersion.value) return
  try {
    const res = await fetch(STATUS_URL + '?t=' + Date.now())
    const data = await res.json()
    latestVersion.value = data.latest_extension_version || null
  } catch {}
})

async function authorize() {
  loading.value   = true
  statusMsg.value = ''
  try {
    const res = await getScrapeToken()
    const token      = res.data.scrape_token
    const backendUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080'

    await new Promise((resolve, reject) => {
      if (!window.chrome?.runtime?.sendMessage) { reject(new Error('not_found')); return }
      chrome.runtime.sendMessage(
        EXTENSION_ID,
        { type: 'FORTUNE_SETUP', token, backendUrl },
        (response) => {
          if (chrome.runtime.lastError) {
            reject(new Error(chrome.runtime.lastError.message))
          } else if (response?.success) {
            resolve()
          } else {
            reject(new Error('擴充功能回應失敗'))
          }
        }
      )
    })

    statusMsg.value  = '連結成功！同步工具已設定完成。'
    statusType.value = 'success'
  } catch (e) {
    const msg = e.message || ''
    statusMsg.value = msg.includes('Could not establish connection') || msg.includes('Extension') || msg === 'not_found'
      ? '找不到同步工具，請先下載並安裝。'
      : msg
    statusType.value = 'error'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.update-banner {
  background: #fffbeb;
  border: 1px solid #fde68a;
  border-radius: 8px;
  padding: 12px 16px;
  font-size: 14px;
  color: #92400e;
  max-width: 480px;
  margin-bottom: 20px;
}
.update-banner a { color: var(--color-primary); font-weight: 600; margin-left: 4px; }
html.dark .update-banner { background: #451a03; border-color: #78350f; color: #fde68a; }

.setup-card {
  background: white;
  border-radius: 12px;
  padding: 40px 32px;
  max-width: 480px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.desc { color: #555; line-height: 1.6; margin: 0; }
.status-msg { margin: 0; font-size: 14px; }
.status-msg.success { color: #52c41a; }
.status-msg.error   { color: #ff4d4f; }
.install-guide { display: flex; flex-direction: column; gap: 10px; }
.download-btn {
  display: inline-block;
  padding: 8px 20px;
  background: var(--color-primary);
  color: white;
  border-radius: 6px;
  text-decoration: none;
  font-size: 14px;
  font-weight: 600;
  text-align: center;
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
.success-icon { font-size: 40px; text-align: center; }
.success-title { font-size: 20px; font-weight: bold; text-align: center; margin: 0; }
.hint-box {
  background: #f0fdf4;
  border: 1px solid #bbf7d0;
  border-radius: 8px;
  padding: 14px 16px;
  font-size: 14px;
  color: #166534;
}
.hint-box p { margin: 0; }
html.dark .hint-box { background: #14532d; border-color: #166534; color: #bbf7d0; }
</style>
