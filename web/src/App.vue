<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import Sidebar from './components/Sidebar.vue'
import { useOnline } from './composables/useOnline'

const route = useRoute()
const isLoginPage = computed(() => route.name === 'login')
const { isOnline } = useOnline()
</script>

<template>
  <div v-if="!isOnline" class="offline-banner">
    Kamu sedang offline — perubahan gak bakal tersimpan sampai koneksi balik.
  </div>
  <RouterView v-if="isLoginPage" />
  <div v-else class="layout">
    <Sidebar />
    <main class="content">
      <RouterView />
    </main>
  </div>
</template>
