import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  { path: '/login', name: 'login', component: () => import('../views/Login.vue'), meta: { public: true } },
  { path: '/', name: 'dashboard', component: () => import('../views/Dashboard.vue') },
  { path: '/providers', name: 'providers', component: () => import('../views/Providers.vue') },
  { path: '/bots', name: 'bots', component: () => import('../views/Bots.vue') },
  { path: '/playground', name: 'playground', component: () => import('../views/Playground.vue') },
  { path: '/chat-gateway/whatsapp', name: 'gateway-whatsapp', component: () => import('../views/gateway/WhatsAppSessions.vue') },
  { path: '/chat-gateway/telegram', name: 'gateway-telegram', component: () => import('../views/gateway/TelegramBots.vue') },
  { path: '/inbox', name: 'inbox', component: () => import('../views/Inbox.vue') },
  { path: '/settings', name: 'settings', component: () => import('../views/Settings.vue') },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (!to.meta.public && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.name === 'login' && auth.isAuthenticated) {
    return { name: 'dashboard' }
  }
  return true
})

export default router
