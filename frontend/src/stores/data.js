import { defineStore } from 'pinia'
import { getStats } from '../api/index'

export const useDataStore = defineStore('data', {
  state: () => ({
    hasData: null,
  }),
  actions: {
    // 每次進站都重新查，不能只查第一次就快取——擴充功能同步資料是在完全獨立的瀏覽器分頁/情境
    // 發生的，網站這邊沒有任何管道知道「使用者剛剛才同步完」，如果快取住第一次查到的結果（很可能
    // 是使用者還沒同步過的 false），之後不管實際同步了多少筆，這裡永遠不會重新確認，頁面就會一直
    // 卡在 EmptyState，即使 GetRecords 等實際查詢早就有資料
    async check() {
      try {
        const res = await getStats()
        this.hasData = (res.data?.total_applied ?? 0) > 0
      } catch {
        // 網路失敗時保持既有值，讓頁面自行顯示錯誤狀態
      }
    },
    invalidate() {
      this.hasData = null
    },
  },
})
