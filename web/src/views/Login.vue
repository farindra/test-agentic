<script setup>
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

async function submit() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(username.value, password.value)
    router.push(route.query.redirect || { name: 'dashboard' })
  } catch (e) {
    error.value = e.message || 'Login gagal'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-screen">
    <div class="login-card">
      <h1>Test Agentic</h1>
      <p>Masuk buat kelola chatbot AI &amp; chat gateway.</p>
      <div v-if="error" class="alert alert-danger">{{ error }}</div>
      <form @submit.prevent="submit">
        <div class="field">
          <label>Username</label>
          <input class="input" v-model="username" autocomplete="username" required />
        </div>
        <div class="field">
          <label>Password</label>
          <input class="input" type="password" v-model="password" autocomplete="current-password" required />
        </div>
        <button class="btn btn-primary" style="width: 100%; justify-content: center" :disabled="loading">
          {{ loading ? 'Memproses...' : 'Masuk' }}
        </button>
      </form>
    </div>
  </div>
</template>

