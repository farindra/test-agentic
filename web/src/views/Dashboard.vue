<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api/client'

const summary = ref(null)
const error = ref('')
const loading = ref(true)

onMounted(async () => {
  try {
    summary.value = await api.get('/dashboard/summary')
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="page-header">
    <div>
      <h1 class="page-title">Dashboard</h1>
      <p class="page-subtitle">Ringkasan chatbot AI dan chat gateway kamu.</p>
    </div>
  </div>

  <div v-if="error" class="alert alert-danger">{{ error }}</div>

  <div v-if="loading" class="loading-state"><span class="spinner"></span> Memuat dashboard...</div>

  <div v-if="summary" class="grid grid-4">
    <div class="card stat-card">
      <span class="stat-label">AI Provider</span>
      <span class="stat-value">{{ summary.providers_total }}</span>
      <span class="stat-sub">provider terdaftar</span>
    </div>
    <div class="card stat-card">
      <span class="stat-label">Chatbot</span>
      <span class="stat-value">{{ summary.bots_total }}</span>
      <span class="stat-sub">profile bot aktif</span>
    </div>
    <div class="card stat-card">
      <span class="stat-label">WhatsApp</span>
      <span class="stat-value">{{ summary.whatsapp_connected }}/{{ summary.whatsapp_sessions }}</span>
      <span class="stat-sub">sesi tersambung</span>
    </div>
    <div class="card stat-card">
      <span class="stat-label">Telegram</span>
      <span class="stat-value">{{ summary.telegram_connected }}/{{ summary.telegram_sessions }}</span>
      <span class="stat-sub">bot tersambung</span>
    </div>
  </div>

  <div v-if="summary" class="card" style="margin-top: 16px">
    <span class="stat-label">Total Percakapan</span>
    <div class="stat-value" style="margin-top: 6px">{{ summary.conversations_total }}</div>
    <p class="text-muted" style="margin: 8px 0 0">
      Percakapan dari semua gateway (WhatsApp + Telegram) yang tercatat di inbox.
    </p>
  </div>
</template>

