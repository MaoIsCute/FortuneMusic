import axios from 'axios'
import { ElMessage } from 'element-plus'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080',
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = 'Bearer ' + token
  return config
})

api.interceptors.response.use(
  (res) => res,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      ElMessage.warning('登入已過期，請重新登入')
      setTimeout(() => { window.location.href = '/' }, 1500)
    }
    return Promise.reject(error)
  }
)

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

export default api
