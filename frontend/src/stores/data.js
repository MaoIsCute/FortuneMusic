import { defineStore } from 'pinia'
import { getStats } from '../api/index'

export const useDataStore = defineStore('data', {
  state: () => ({
    hasData: null,
  }),
  actions: {
    async check() {
      if (this.hasData !== null) return
      try {
        const res = await getStats()
        this.hasData = (res.data?.total_applied ?? 0) > 0
      } catch {
        // 網路失敗時保持 null，讓頁面自行顯示錯誤狀態
      }
    },
    invalidate() {
      this.hasData = null
    },
  },
})
