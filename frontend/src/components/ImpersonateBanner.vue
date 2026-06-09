<template>
  <div v-if="store.user" class="impersonate-banner">
    <span>👁 正在模擬 <strong>{{ store.user.name }}</strong>（{{ store.user.email }}）的畫面</span>
    <button class="exit-btn" @click="exit">退出模擬</button>
  </div>
</template>

<script setup>
import { useImpersonateStore } from '../stores/impersonate'
import { useDataStore } from '../stores/data'
import { useRouter } from 'vue-router'

const store = useImpersonateStore()
const dataStore = useDataStore()
const router = useRouter()

function exit() {
  store.stop()
  dataStore.invalidate()
  router.push('/dashboard')
}
</script>

<style scoped>
.impersonate-banner {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 9999;
  background: #f59e0b;
  color: #1a1a1a;
  padding: 8px 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 14px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.15);
}
.exit-btn {
  padding: 4px 14px;
  background: rgba(0,0,0,0.15);
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 600;
  color: #1a1a1a;
}
.exit-btn:hover { background: rgba(0,0,0,0.25); }
</style>
