<template>
  <div class="page">
    <h1 class="page-title">爬蟲</h1>

    <div class="scrape-card">
      <h2 class="section-title">瀏覽器擴充功能（推薦）</h2>
      <p class="desc">
        安裝擴充功能後，在 Fortune Music 網站登入，點擊擴充功能圖示即可自動同步。
        首次使用需在擴充功能內輸入下方的 Token。
      </p>
      <div class="token-row">
        <el-input
          v-model="scrapeToken"
          readonly
          placeholder="點擊「取得 Token」生成"
        />
        <el-button @click="fetchToken" :loading="tokenLoading">取得 Token</el-button>
        <el-button v-if="scrapeToken" @click="copyToken">複製</el-button>
      </div>
      <p class="hint">後端網址：{{ apiUrl }}</p>
    </div>

    <div class="scrape-card" style="margin-top: 20px">
      <h2 class="section-title">手動輸入 Cookie</h2>
      <p class="desc">請從瀏覽器複製 Fortune Music 的 Cookie 並貼上：</p>
      <el-input
        v-model="cookie"
        type="textarea"
        :rows="4"
        placeholder="貼上 Cookie..."
      />
      <el-button
        type="primary"
        :loading="loading"
        @click="showConfirm"
        style="margin-top: 16px; width: 100%"
      >
        執行爬蟲
      </el-button>
      <div v-if="result" class="result">{{ result }}</div>
    </div>

    <el-dialog v-model="dialogVisible" title="確認" width="400px">
      <p>Cookie 將傳送至伺服器用於爬蟲，執行完畢後不會永久保存。確定要繼續嗎？</p>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="doScrape">確認執行</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { triggerScrape, getScrapeToken } from '../api/index'

const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080'

const cookie = ref('')
const loading = ref(false)
const result = ref('')
const dialogVisible = ref(false)

const scrapeToken = ref('')
const tokenLoading = ref(false)

async function fetchToken() {
  tokenLoading.value = true
  try {
    const res = await getScrapeToken()
    scrapeToken.value = res.data.scrape_token
  } catch {
    ElMessage.error('取得 Token 失敗')
  } finally {
    tokenLoading.value = false
  }
}

function copyToken() {
  navigator.clipboard.writeText(scrapeToken.value)
  ElMessage.success('已複製')
}

function showConfirm() {
  if (!cookie.value.trim()) return
  dialogVisible.value = true
}

async function doScrape() {
  dialogVisible.value = false
  loading.value = true
  try {
    const res = await triggerScrape(cookie.value)
    result.value = res.data.message
  } catch {
    result.value = '爬蟲失敗，請確認 Cookie 是否正確'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.scrape-card {
  background: white;
  border-radius: 12px;
  padding: 32px;
  max-width: 600px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
}
.section-title { font-size: 16px; font-weight: bold; margin: 0 0 12px; }
.desc { color: #666; margin-bottom: 16px; }
.token-row { display: flex; gap: 8px; align-items: center; }
.hint { color: #aaa; font-size: 12px; margin-top: 8px; }
.result { margin-top: 16px; padding: 12px; border-radius: 8px; background: #f5f5f5; }
</style>
