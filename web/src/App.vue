<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import Sidebar from './components/Sidebar.vue'
import { useOnline } from './composables/useOnline'

const route = useRoute()
const isLoginPage = computed(() => route.name === 'login')
const { isOnline } = useOnline()

const mobileMenuOpen = ref(false)
// Pindah halaman = drawer mobile wajib nutup sendiri, kalau nggak
// nu-user harus nutup manual tiap abis milih menu.
watch(() => route.fullPath, () => { mobileMenuOpen.value = false })
</script>

<template>
  <div v-if="!isOnline" class="offline-banner">
    Kamu sedang offline — perubahan gak bakal tersimpan sampai koneksi balik.
  </div>
  <RouterView v-if="isLoginPage" />
  <div v-else class="layout">
    <div class="mobile-topbar">
      <button class="mobile-menu-btn" aria-label="Buka menu" @click="mobileMenuOpen = true">☰</button>
      <span class="brand">Test Agentic</span>
    </div>
    <div v-if="mobileMenuOpen" class="sidebar-backdrop" @click="mobileMenuOpen = false"></div>
    <Sidebar :class="{ 'sidebar-open': mobileMenuOpen }" />
    <main class="content">
      <RouterView />
    </main>
  </div>
</template>
