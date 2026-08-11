<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import Sidebar from './components/Sidebar.vue'

const route = useRoute()
const isLoginPage = computed(() => route.name === 'login')
</script>

<template>
  <RouterView v-if="isLoginPage" />
  <div v-else class="layout">
    <Sidebar />
    <main class="content">
      <RouterView />
    </main>
  </div>
</template>
