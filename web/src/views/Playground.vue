<script setup>
import { ref, onMounted, computed } from 'vue'
import { api } from '../api/client'

const bots = ref([])
const sessions = ref([])
const activeSessionId = ref('')
const messages = ref([])
const draft = ref('')
const sending = ref(false)
const error = ref('')
const loading = ref(true)
const newBotId = ref('')
const mobileShowThread = ref(false)

const activeSession = computed(() => sessions.value.find((s) => s.id === activeSessionId.value))

async function loadBots() {
  const res = await api.get('/bots')
  bots.value = res.bots || []
  if (bots.value.length) newBotId.value = bots.value[0].id
}

async function loadSessions() {
  const res = await api.get('/playground/sessions')
  sessions.value = res.sessions || []
}

async function selectSession(id) {
  error.value = ''
  activeSessionId.value = id
  mobileShowThread.value = true
  try {
    const res = await api.get(`/playground/sessions/${id}/messages`)
    messages.value = res.messages || []
  } catch (e) {
    error.value = e.message
  }
}

async function createSession() {
  if (!newBotId.value) return
  error.value = ''
  try {
    const bot = bots.value.find((b) => b.id === newBotId.value)
    const res = await api.post('/playground/sessions', { bot_id: newBotId.value, title: `Chat dengan ${bot?.name || 'Bot'}` })
    sessions.value.unshift(res)
    await selectSession(res.id)
  } catch (e) {
    error.value = e.message
  }
}

async function send() {
  if (!draft.value.trim() || !activeSessionId.value) return
  const text = draft.value
  draft.value = ''
  messages.value.push({ id: 'tmp-' + Date.now(), role: 'user', content: text })
  sending.value = true
  error.value = ''
  try {
    const reply = await api.post(`/playground/sessions/${activeSessionId.value}/chat`, { message: text })
    messages.value.push(reply)
  } catch (e) {
    error.value = e.message
  } finally {
    sending.value = false
  }
}

onMounted(async () => {
  try {
    await loadBots()
    await loadSessions()
    if (sessions.value.length) await selectSession(sessions.value[0].id)
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
      <h1 class="page-title">Playground</h1>
      <p class="page-subtitle">Coba chatbot langsung di sini sebelum dihubungkan ke WhatsApp/Telegram.</p>
    </div>
    <div style="display:flex; gap:8px">
      <select class="select" v-model="newBotId" style="width:auto">
        <option v-for="b in bots" :key="b.id" :value="b.id">{{ b.name }}</option>
      </select>
      <button class="btn btn-primary" :disabled="!newBotId" @click="createSession">+ Chat Baru</button>
    </div>
  </div>

  <div v-if="error" class="alert alert-danger">{{ error }}</div>
  <div v-if="!loading && !bots.length" class="alert alert-danger">Belum ada chatbot. Bikin dulu di menu "Chatbot".</div>

  <div v-if="loading" class="loading-state card"><span class="spinner"></span> Memuat playground...</div>

  <div class="chat-panel" :class="{ 'mobile-show-thread': mobileShowThread }" v-else-if="sessions.length">
    <div class="chat-list">
      <div
        v-for="s in sessions" :key="s.id"
        class="chat-list-item" :class="{ active: s.id === activeSessionId }"
        @click="selectSession(s.id)"
      >
        <div class="name">{{ s.title }}</div>
        <div class="preview">{{ new Date(s.created_at).toLocaleString('id-ID') }}</div>
      </div>
    </div>

    <div class="chat-thread">
      <div class="chat-thread-header">
        <div class="title-row">
          <button class="mobile-back-btn" aria-label="Kembali" @click="mobileShowThread = false">←</button>
          <strong>{{ activeSession?.title || 'Pilih percakapan' }}</strong>
        </div>
      </div>
      <div class="chat-messages">
        <div v-for="m in messages" :key="m.id" class="chat-bubble" :class="m.role === 'user' ? 'chat-bubble-out' : 'chat-bubble-in'">
          {{ m.content }}
        </div>
        <div v-if="sending" class="chat-bubble chat-bubble-in text-muted">Mengetik...</div>
      </div>
      <form class="chat-composer" @submit.prevent="send">
        <input class="input" v-model="draft" placeholder="Ketik pesan..." :disabled="!activeSessionId || sending" />
        <button class="btn btn-primary" type="submit" :disabled="!activeSessionId || sending">Kirim</button>
      </form>
    </div>
  </div>

  <div v-else-if="bots.length" class="empty-state card">
    <div class="icon">▶</div>
    <p>Belum ada percakapan playground. Klik "+ Chat Baru" buat mulai.</p>
  </div>
</template>

