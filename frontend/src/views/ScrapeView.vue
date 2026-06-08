<template>
  <div class="page">
    <h1 class="page-title">同步工具設定</h1>
    <!-- 連結前 -->
    <div v-if="statusType !== 'success'" class="setup-card">
      <p class="desc">點擊下方按鈕，自動將你的帳號與同步工具連結，完成一鍵設定。</p>
      <el-button type="primary" size="large" :loading="loading" @click="authorize">
        連結同步工具
      </el-button>
      <p v-if="statusMsg" :class="['status-msg', statusType]">{{ statusMsg }}</p>
    </div>

    <!-- 連結成功後：引導下一步 -->
    <div v-else class="setup-card">
      <div class="success-icon">✅</div>
      <h2 class="success-title">連結成功！</h2>
      <p class="desc">最後一步：點 Chrome 右上角的同步工具圖示，再點「同步」，就會開始抓取你的資料。</p>
      <div class="hint-box">
        <p>抓取完成後，點左上角「總覽」就能看到你的統計資料了。</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { getScrapeToken } from '../api/index'

const EXTENSION_ID = 'gdclpkfeiocedicokoenhconeoocigeh'

const router = useRouter()
const loading   = ref(false)
const statusMsg = ref('')
const statusType = ref('')

async function authorize() {
  loading.value   = true
  statusMsg.value = ''
  try {
    const res = await getScrapeToken()
    const token      = res.data.scrape_token
    const backendUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080'

    await new Promise((resolve, reject) => {
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
    statusMsg.value = msg.includes('Could not establish connection') || msg.includes('Extension')
      ? '找不到同步工具，請確認已安裝並啟用。'
      : msg
    statusType.value = 'error'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
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
