import { defineStore } from 'pinia'

export const useImpersonateStore = defineStore('impersonate', {
  state: () => ({ user: null }),
  actions: {
    start(user) { this.user = user },
    stop()       { this.user = null },
  },
})
