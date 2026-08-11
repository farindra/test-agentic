<script setup>
import { ref, onMounted, reactive } from 'vue'
import { api } from '../../api/client'

const sessions = ref([])
const bots = ref([])
const error = ref('')
const loading = ref(true)
const showCreate = ref(false)
const busy = reactive({})

const form = reactive({ label: '', token: '' })

async function load() {
  error.value = ''
  try {
    const [s, b] = await Promise.all([api.get('/gateway/telegram/sessions'), api.get('/bots')])
    sessions.value = s.sessions || []
    bots.value = b.bots || []
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function createSession() {
  try {
    await api.post('/gateway/telegram/sessions', { label: form.label, token: form.token })
    form.label = ''
    form.token = ''
    showCreate.value = false
    await load()
  } catch (e) {
    error.value = e.message
  }
}

async function activate(s) {
  busy[s.id] = true
  try {
    await api.post(`/gateway/telegram/sessions/${s.id}/activate`, {})
    await load()
  } catch (e) {
    error.value = e.message
  } finally {
    busy[s.id] = false
  }
}

async function deactivate(s) {
  busy[s.id] = true
  try {
    await api.post(`/gateway/telegram/sessions/${s.id}/deactivate`, {})
    await load()
  } catch (e) {
    error.value = e.message
  } finally {
    busy[s.id] = false
  }
}

async function updateBinding(s) {
  try {
    await api.patch(`/gateway/telegram/sessions/${s.id}`, { bot_id: s.bot_id || '', auto_reply: s.auto_reply })
  } catch (e) {
    error.value = e.message
  }
}

async function remove(s) {
  if (!confirm(`Hapus bot Telegram "${s.label || s.id}"?`)) return
  try {
    await api.del(`/gateway/telegram/sessions/${s.id}`)
    await load()
  } catch (e) {
    error.value = e.message
  }
}

onMounted(load)
</script>

<template>
  <div class="page-header">
    <div>
      <h1 class="page-title">Chat Gateway — Telegram</h1>
      <p class="page-subtitle">Kelola bot Telegram (via BotFather token) dan binding-nya ke chatbot.</p>
    </div>
    <button class="btn btn-primary" @click="showCreate = true">+ Bot Baru</button>
  </div>

  <div v-if="error" class="alert alert-danger">{{ error }}</div>

  <div class="card" style="padding: 0">
    <div v-if="loading" class="loading-state"><span class="spinner"></span> Memuat bot Telegram...</div>
    <div v-else-if="sessions.length" class="table-scroll">
    <table class="table">
      <thead>
        <tr><th>Label</th><th>Username</th><th>Status</th><th>Bot AI</th><th>Auto-Reply</th><th></th></tr>
      </thead>
      <tbody>
        <tr v-for="s in sessions" :key="s.id">
          <td><strong>{{ s.label || '(tanpa label)' }}</strong></td>
          <td>{{ s.telegram_username ? '@' + s.telegram_username : '—' }}</td>
          <td><span class="badge" :class="'badge-' + s.status">{{ s.status }}</span></td>
          <td>
            <select class="select" style="width:auto" v-model="s.bot_id" @change="updateBinding(s)">
              <option :value="null">— pilih bot —</option>
              <option v-for="b in bots" :key="b.id" :value="b.id">{{ b.name }}</option>
            </select>
          </td>
          <td>
            <label class="toggle">
              <input type="checkbox" v-model="s.auto_reply" @change="updateBinding(s)" />
              <span class="toggle-track"></span>
            </label>
          </td>
          <td>
            <div style="display:flex; gap:6px; justify-content:flex-end">
              <button class="btn btn-sm" v-if="s.status !== 'connected'" :disabled="busy[s.id]" @click="activate(s)">Aktifkan</button>
              <button class="btn btn-sm" v-else :disabled="busy[s.id]" @click="deactivate(s)">Nonaktifkan</button>
              <button class="btn btn-sm btn-danger" @click="remove(s)">Hapus</button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
    </div>
    <div v-else class="empty-state">
      <div class="icon">✈</div>
      <p>Belum ada bot Telegram. Buat bot lewat @BotFather, lalu tempel token-nya di sini.</p>
    </div>
  </div>

  <div v-if="showCreate" class="modal-backdrop" @click.self="showCreate = false">
    <div class="modal">
      <h2 class="modal-title">Bot Telegram Baru</h2>
      <form @submit.prevent="createSession" autocomplete="off">
        <div class="field">
          <label>Label (opsional)</label>
          <input class="input" v-model="form.label" placeholder="mis. CS Bot Telegram" autocomplete="off" />
        </div>
        <div class="field">
          <label>Bot Token (dari @BotFather)</label>
          <input class="input" type="password" v-model="form.token" required placeholder="123456:ABC-DEF..." autocomplete="new-password" />
        </div>
        <div class="modal-actions">
          <button type="button" class="btn" @click="showCreate = false">Batal</button>
          <button type="submit" class="btn btn-primary">Simpan</button>
        </div>
      </form>
    </div>
  </div>
</template>

