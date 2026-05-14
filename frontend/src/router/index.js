import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  { path: '/', name: 'Login', component: () => import('../views/LoginView.vue'), meta: { public: true } },
  { path: '/dashboard', name: 'Dashboard', component: () => import('../views/DashboardView.vue') },
  { path: '/member/:name', name: 'Member', component: () => import('../views/MemberView.vue') },
  { path: '/records', name: 'Records', component: () => import('../views/RecordsView.vue') },
  { path: '/scrape', name: 'Scrape', component: () => import('../views/ScrapeView.vue') },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (!to.meta.public && !auth.isLoggedIn) return { name: 'Login' }
})

export default router
