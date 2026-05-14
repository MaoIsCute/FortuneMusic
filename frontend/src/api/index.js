import axios from 'axios'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080',
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = 'Bearer ' + token
  return config
})

export const getStats = () => api.get('/api/stats/overall')
export const getStatsByDate = () => api.get('/api/stats/by-date')
export const getStatsBySession = () => api.get('/api/stats/by-session')
export const getStatsByMember = () => api.get('/api/stats/by-member')
export const getRecords = () => api.get('/api/records')
export const triggerScrape = (cookie) => api.post('/api/scrape', { cookie })

export default api
