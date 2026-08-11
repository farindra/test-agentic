<script setup>
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()

function logout() {
  auth.logout()
  router.push({ name: 'login' })
}
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar-brand">
      <span class="sidebar-brand-dot"></span>
      Test Agentic
    </div>

    <RouterLink to="/" custom v-slot="{ navigate, isExactActive }">
      <div class="sidebar-link" :class="{ active: isExactActive }" @click="navigate">
        <span class="icon">◆</span> Dashboard
      </div>
    </RouterLink>

    <div class="sidebar-section">AI</div>
    <RouterLink to="/providers" custom v-slot="{ navigate, isActive }">
      <div class="sidebar-link" :class="{ active: isActive }" @click="navigate">
        <span class="icon">⚙</span> AI Provider
      </div>
    </RouterLink>
    <RouterLink to="/bots" custom v-slot="{ navigate, isActive }">
      <div class="sidebar-link" :class="{ active: isActive }" @click="navigate">
        <span class="icon">✦</span> Chatbot
      </div>
    </RouterLink>
    <RouterLink to="/variables" custom v-slot="{ navigate, isActive }">
      <div class="sidebar-link" :class="{ active: isActive }" @click="navigate">
        <span class="icon">{ }</span> Variables
      </div>
    </RouterLink>
    <RouterLink to="/playground" custom v-slot="{ navigate, isActive }">
      <div class="sidebar-link" :class="{ active: isActive }" @click="navigate">
        <span class="icon">▶</span> Playground
      </div>
    </RouterLink>

    <div class="sidebar-section">Chat Gateway</div>
    <RouterLink to="/chat-gateway/whatsapp" custom v-slot="{ navigate, isActive }">
      <div class="sidebar-link" :class="{ active: isActive }" @click="navigate">
        <span class="icon">▣</span> WhatsApp
      </div>
    </RouterLink>
    <RouterLink to="/chat-gateway/telegram" custom v-slot="{ navigate, isActive }">
      <div class="sidebar-link" :class="{ active: isActive }" @click="navigate">
        <span class="icon">✈</span> Telegram
      </div>
    </RouterLink>
    <RouterLink to="/inbox" custom v-slot="{ navigate, isActive }">
      <div class="sidebar-link" :class="{ active: isActive }" @click="navigate">
        <span class="icon">☰</span> Inbox
      </div>
    </RouterLink>

    <div class="sidebar-footer">
      <div class="sidebar-link" @click="logout">
        <span class="icon">⏻</span> Keluar
      </div>
    </div>
  </aside>
</template>
