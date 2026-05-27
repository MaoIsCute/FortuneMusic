import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  { path: '/', name: 'Login', component: () => import('../views/LoginView.vue'), meta: { public: true } },
  { path: '/auth/callback', name: 'AuthCallback', component: () => import('../views/AuthCallbackView.vue'), meta: { public: true } },
  { path: '/dashboard', name: 'Dashboard', component: () => import('../views/DashboardView.vue') },
  { path: '/member/:name', name: 'Member', component: () => import('../views/MemberView.vue') },
  { path: '/records', name: 'Records', component: () => import('../views/RecordsView.vue') },
  { path: '/scrape', name: 'Scrape', component: () => import('../views/ScrapeView.vue') },
  { path: '/admin', name: 'Admin', component: () => import('../views/AdminView.vue'), meta: { admin: true } },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (!to.meta.public && !auth.isLoggedIn) {
    localStorage.setItem('redirectAfterLogin', to.fullPath)
    return { name: 'Login' }
  }
  if (to.meta.admin && !auth.user?.is_admin) {
    return { name: 'Dashboard' }
  }
})

export default router
