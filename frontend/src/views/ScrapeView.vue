<template>
  <div class="page">
    <h1 class="page-title">爬蟲</h1>
    <div class="scrape-card">
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
import { triggerScrape } from '../api/index'

const cookie = ref('')
const loading = ref(false)
const result = ref('')
const dialogVisible = ref(false)

function showConfirm() {
  if (!cookie.value.trim()) return
  dialogVisible.value = true
}

async function doScrape() {
  dialogVisible.value = false
  loading.value = true
  try {
    const res = await triggerScrape(cookie.value)
    result.value = '完成！共爬取 ' + res.data.count + ' 筆資料'
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
.desc { color: #666; margin-bottom: 16px; }
.result { margin-top: 16px; padding: 12px; border-radius: 8px; background: #f5f5f5; }
</style>
