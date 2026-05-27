import axios from 'axios'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores/auth'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080',
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = 'Bearer ' + token
  return config
})

let isRefreshing = false

api.interceptors.response.use(
  (res) => res,
  async (error) => {
    const isAuthEndpoint = error.config?.url?.includes('/auth/')
    if (error.response?.status === 401 && !error.config._retry && !isAuthEndpoint) {
      const rt = localStorage.getItem('refreshToken')
      if (rt && !isRefreshing) {
        error.config._retry = true
        isRefreshing = true
        try {
          const res = await api.post('/auth/refresh', { refresh_token: rt })
          const newToken = res.data.token
          const auth = useAuthStore()
          auth.setToken(newToken)
          error.config.headers.Authorization = 'Bearer ' + newToken
          isRefreshing = false
          return api(error.config)
        } catch {
          isRefreshing = false
        }
      }
      const auth = useAuthStore()
      auth.logout()
      ElMessage.warning('登入已過期，請重新登入')
      setTimeout(() => { window.location.href = '/' }, 1500)
    }
    return Promise.reject(error)
  }
)

export const exchangeToken = (code) => api.post('/auth/token', { code })
export const getStats = () => api.get('/api/stats/overall')
export const getStatsByDate = () => api.get('/api/stats/by-date')
export const getStatsBySession = () => api.get('/api/stats/by-session')
export const getStatsByMember = () => api.get('/api/stats/by-member')
export const getDetailStats = () => api.get('/api/stats/detail')
export const getOrderSequenceStats = (params = {}) => api.get('/api/stats/order-sequence', { params })
export const getRecords = (params = {}) => api.get('/api/records', { params: { page_size: 100, ...params } })
export const getMe = () => api.get('/api/me')
export const getScrapeToken = () => api.get('/api/scrape-token')
export const triggerScrape = (cookie) => api.post('/api/scrape', { cookie })
export const getAdminTitleIssues = () => api.get('/api/admin/title-issues')
export const fixSingleTitle = (single_number, single_name) => api.put('/api/admin/title', { single_number, single_name })
export const getAdminUsers = () => api.get('/api/admin/users')
export const deleteUserRecords = (id) => api.delete(`/api/admin/users/${id}/records`)

export default api
