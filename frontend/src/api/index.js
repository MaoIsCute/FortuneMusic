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
let networkErrorShown = false

api.interceptors.response.use(
  (res) => res,
  async (error) => {
    if (!error.response && !networkErrorShown) {
      networkErrorShown = true
      ElMessage({ type: 'error', message: '無法連線到伺服器，請稍後再試', duration: 4000,
        onClose: () => { networkErrorShown = false } })
    }
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
export const getAdminPurchaseTitleIssues = () => api.get('/api/admin/purchase-title-issues')
export const fixPurchaseTitle = (single_number, single_name) => api.put('/api/admin/purchase-title', { single_number, single_name })
export const getAdminUsers = () => api.get('/api/admin/users')
export const deleteUserRecords = (id, params = {}) => api.delete(`/api/admin/users/${id}/records`, { params })
export const deleteUserFullRecords = (id, params = {}) => api.delete(`/api/admin/users/${id}/full-records`, { params })
export const deleteUserPurchases = (id, params = {}) => api.delete(`/api/admin/users/${id}/purchases`, { params })
export const getFullRecords = (params = {}) => api.get('/api/full/records', { params: { page_size: 50, ...params } })
export const getFullOverallStats = () => api.get('/api/full/stats/overall')
export const getFullStatsByMember = () => api.get('/api/full/stats/by-member')
export const getFullStatsBySingle = () => api.get('/api/full/stats/by-single')
export const getPurchases = (params = {}) => api.get('/api/purchases', { params: { page_size: 50, ...params } })
export const getPurchaseTree = () => api.get('/api/purchases/tree')
export const getPurchaseOverallStats = () => api.get('/api/purchases/stats/overall')
export const getPurchaseStatsBySingle = () => api.get('/api/purchases/stats/by-single')
export const getPurchaseStatsByMember = () => api.get('/api/purchases/stats/by-member')
export const getAdminScrapeLogs = () => api.get('/api/admin/scrape-logs')

export default api
