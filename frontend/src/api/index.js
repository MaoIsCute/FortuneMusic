import axios from 'axios'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores/auth'
import { useImpersonateStore } from '../stores/impersonate'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080',
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = 'Bearer ' + token
  const impersonateStore = useImpersonateStore()
  if (impersonateStore.user) {
    config.headers['X-Impersonate-User'] = String(impersonateStore.user.id)
  }
  return config
})

let isRefreshing = false
let networkErrorShown = false
let pendingRequests = []

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
      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          pendingRequests.push({ resolve, reject })
        }).then(() => {
          error.config.headers.Authorization = 'Bearer ' + localStorage.getItem('token')
          return api(error.config)
        })
      }

      error.config._retry = true
      isRefreshing = true
      const rt = localStorage.getItem('refreshToken')
      if (rt) {
        try {
          const res = await api.post('/auth/refresh', { refresh_token: rt })
          const newToken = res.data.token
          const auth = useAuthStore()
          auth.setToken(newToken)
          error.config.headers.Authorization = 'Bearer ' + newToken
          pendingRequests.forEach(p => p.resolve())
          pendingRequests = []
          isRefreshing = false
          return api(error.config)
        } catch {
          pendingRequests.forEach(p => p.reject())
          pendingRequests = []
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
export const getStatsByMember = (params = {}) => api.get('/api/stats/by-member', { params })
export const getDetailStats = (params = {}) => api.get('/api/stats/detail', { params })
export const getOrderSequenceStats = (params = {}) => api.get('/api/stats/order-sequence', { params })
export const getRecords = (params = {}) => api.get('/api/records', { params: { page_size: 100, ...params } })
export const getMe = () => api.get('/api/me')
export const getScrapeToken = () => api.get('/api/scrape-token')
export const triggerScrape = (cookie) => api.post('/api/scrape', { cookie })
export const getAdminTitleIssues = () => api.get('/api/admin/title-issues')
export const getAdminKnownTitles = () => api.get('/api/admin/titles')
export const fixSingleTitle = (group, single_number, single_name, org_album_name = '', release_date = '') => api.put('/api/admin/title', { group, single_number, single_name, org_album_name, release_date })
export const bulkSetTitles = (titles) => api.post('/api/admin/titles/bulk', { titles })
export const getAdminVenueIssues = () => api.get('/api/admin/venue-issues')
export const fixVenue = (group, single_number, event_date, venue) => api.put('/api/admin/venue', { group, single_number, event_date, venue })
export const bulkSetVenues = (venues) => api.post('/api/admin/venues/bulk', { venues })
export const getAdminKnownVenues = () => api.get('/api/admin/venues')
export const getAdminPurchaseTitleIssues = () => api.get('/api/admin/purchase-title-issues')
export const fixPurchaseTitle = (single_number, single_name) => api.put('/api/admin/purchase-title', { single_number, single_name })
export const normalizeMemberNames = () => api.post('/api/admin/normalize-member-names')
export const getAdminUsers = () => api.get('/api/admin/users')
export const previewUserRecords     = (id, params = {}) => api.get(`/api/admin/users/${id}/records/preview`, { params })
export const previewUserFullRecords = (id, params = {}) => api.get(`/api/admin/users/${id}/full-records/preview`, { params })
export const previewUserPurchases   = (id, params = {}) => api.get(`/api/admin/users/${id}/purchases/preview`, { params })
export const deleteUserRecords      = (id, params = {}) => api.delete(`/api/admin/users/${id}/records`, { params })
export const deleteUserFullRecords  = (id, params = {}) => api.delete(`/api/admin/users/${id}/full-records`, { params })
export const deleteUserPurchases    = (id, params = {}) => api.delete(`/api/admin/users/${id}/purchases`, { params })
export const previewUserSignEvents  = (id, params = {}) => api.get(`/api/admin/users/${id}/sign-events/preview`, { params })
export const deleteUserSignEvents   = (id, params = {}) => api.delete(`/api/admin/users/${id}/sign-events`, { params })
export const previewUserPrizes      = (id, params = {}) => api.get(`/api/admin/users/${id}/prizes/preview`, { params })
export const deleteUserPrizes       = (id, params = {}) => api.delete(`/api/admin/users/${id}/prizes`, { params })
export const getFullRecords = (params = {}) => api.get('/api/full/records', { params: { page_size: 50, ...params } })
export const getFullOverallStats = () => api.get('/api/full/stats/overall')
export const getFullStatsByMember = (params = {}) => api.get('/api/full/stats/by-member', { params })
export const getFullStatsBySingle = (params = {}) => api.get('/api/full/stats/by-single', { params })
export const getFullDetailStats = (params = {}) => api.get('/api/full/stats/detail', { params })
export const getFullStatsByRegion = (params = {}) => api.get('/api/full/stats/by-region', { params })
export const getPurchases = (params = {}) => api.get('/api/purchases', { params: { page_size: 50, ...params } })
export const getPurchaseTree = () => api.get('/api/purchases/tree')
export const getPurchaseOverallStats = () => api.get('/api/purchases/stats/overall')
export const getPurchaseStatsBySingle = () => api.get('/api/purchases/stats/by-single')
export const getPurchaseStatsByMember = () => api.get('/api/purchases/stats/by-member')
export const getAdminScrapeLogs = () => api.get('/api/admin/scrape-logs')
export const getAdminSignEvents = (params = {}) => api.get('/api/admin/sign-events', { params })
export const getAdminPrizes = (params = {}) => api.get('/api/admin/prizes', { params })
export const getSignEvents = (params = {}) => api.get('/api/sign-events', { params })
export const getPrizes = (params = {}) => api.get('/api/prizes', { params })
export const updatePrizeResult = (id, wonStatus) => api.put(`/api/prizes/${id}/result`, { won_status: wonStatus })
export const getGlobalOverallStats = () => api.get('/api/global/stats/overall')
export const getGlobalDetailStats = (params = {}) => api.get('/api/global/stats/detail', { params })
export const getGlobalOrderSequenceStats = (params = {}) => api.get('/api/global/stats/order-sequence', { params })
export const getGlobalStatsByMember = (params = {}) => api.get('/api/global/stats/by-member', { params })
export const getGlobalStatsBySingle = (params = {}) => api.get('/api/global/stats/by-single', { params })
export const getGlobalFullOverallStats = () => api.get('/api/global/full/stats/overall')
export const getGlobalFullStatsByMember = (params = {}) => api.get('/api/global/full/stats/by-member', { params })
export const getGlobalFullStatsByRegion = (params = {}) => api.get('/api/global/full/stats/by-region', { params })
export const getGlobalFullDetailStats = (params = {}) => api.get('/api/global/full/stats/detail', { params })

export default api
