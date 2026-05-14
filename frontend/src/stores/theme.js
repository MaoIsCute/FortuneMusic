import { defineStore } from 'pinia'
import { applyTheme } from '../styles/theme'

export const useThemeStore = defineStore('theme', {
  state: () => ({
    currentMember: 'default',
    isDark: false,
  }),
  actions: {
    setMember(name) {
      this.currentMember = name
      applyTheme(name, this.isDark)
    },
    toggleDark() {
      this.isDark = !this.isDark
      applyTheme(this.currentMember, this.isDark)
      document.documentElement.classList.toggle('dark', this.isDark)
    },
  },
})
