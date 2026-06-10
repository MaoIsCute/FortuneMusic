import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  { path: '/', name: 'Login', component: () => import('../views/LoginView.vue'), meta: { public: true } },
  { path: '/auth/callback', name: 'AuthCallback', component: () => import('../views/AuthCallbackView.vue'), meta: { public: true } },
  { path: '/setup', name: 'Setup', component: () => import('../views/SetupView.vue') },
  { path: '/dashboard', name: 'Dashboard', component: () => import('../views/DashboardView.vue') },
  { path: '/member/:name', name: 'Member', component: () => import('../views/MemberView.vue') },
  { path: '/records', name: 'Records', component: () => import('../views/RecordsView.vue') },
  { path: '/full', name: 'Full', component: () => import('../views/FullView.vue') },
  { path: '/spending', name: 'Spending', component: () => import('../views/SpendingView.vue') },
  { path: '/scrape', name: 'Scrape', component: () => import('../views/ScrapeView.vue') },
  { path: '/admin', name: 'Admin', component: () => import('../views/AdminView.vue'), meta: { admin: true } },
  { path: '/:pathMatch(.*)*', redirect: '/dashboard' },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

const DATA_ROUTES = new Set(['Dashboard', 'Records', 'Spending'])

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!to.meta.public && !auth.isLoggedIn) {
    localStorage.setItem('redirectAfterLogin', to.fullPath)
    return { name: 'Login' }
  }
  if (to.meta.admin && !auth.user?.is_admin) {
    return { name: 'Dashboard' }
  }
  if (auth.isLoggedIn && DATA_ROUTES.has(to.name)) {
    const { useDataStore } = await import('../stores/data')
    await useDataStore().check()
  }
})

export default router
