<script setup>
import { ref, onMounted, computed } from 'vue'
import { api } from '../api/client'

const conversations = ref([])
const activeId = ref('')
const messages = ref([])
const error = ref('')

const active = computed(() => conversations.value.find((c) => c.id === activeId.value))

async function load() {
  const res = await api.get('/conversations')
  conversations.value = res.conversations || []
}

async function select(id) {
  activeId.value = id
  const res = await api.get(`/conversations/${id}/messages`)
  messages.value = res.messages || []
}

async function toggleAutoReply() {
  if (!active.value) return
  try {
    await api.patch(`/conversations/${active.value.id}`, { auto_reply: !active.value.auto_reply })
    active.value.auto_reply = !active.value.auto_reply
  } catch (e) {
    error.value = e.message
  }
}

onMounted(load)
</script>

<template>
  <div class="page-header">
    <div>
      <h1 class="page-title">Inbox</h1>
      <p class="page-subtitle">Semua percakapan dari WhatsApp &amp; Telegram dalam satu tempat.</p>
    </div>
  </div>

  <div v-if="error" class="alert alert-danger">{{ error }}</div>

  <div class="chat-panel" v-if="conversations.length">
    <div class="chat-list">
      <div
        v-for="c in conversations" :key="c.id"
        class="chat-list-item" :class="{ active: c.id === activeId }"
        @click="select(c.id)"
      >
        <div class="name">{{ c.contact_name || c.contact_id }}</div>
        <div class="preview">{{ c.contact_id }} · {{ c.last_message_at ? new Date(c.last_message_at).toLocaleString('id-ID') : '' }}</div>
      </div>
    </div>

    <div class="chat-thread">
      <div class="chat-thread-header">
        <strong>{{ active?.contact_name || active?.contact_id || 'Pilih percakapan' }}</strong>
        <div v-if="active" style="display:flex; align-items:center; gap:8px">
          <span class="text-muted" style="font-size:12px">Auto-reply bot</span>
          <label class="toggle">
            <input type="checkbox" :checked="active.auto_reply" @change="toggleAutoReply" />
            <span class="toggle-track"></span>
          </label>
        </div>
      </div>
      <div class="chat-messages">
        <div v-for="m in messages" :key="m.id" class="chat-bubble" :class="m.direction === 'in' ? 'chat-bubble-in' : 'chat-bubble-out'">
          {{ m.content }}
          <div class="chat-bubble-meta">{{ m.sender }} · {{ new Date(m.created_at).toLocaleTimeString('id-ID') }}</div>
        </div>
      </div>
      <div class="chat-composer">
        <p class="text-muted" style="margin: 0; font-size: 12px">
          Balasan manual belum didukung di versi ini — matikan auto-reply buat handover, lalu bales langsung dari WhatsApp/Telegram.
        </p>
      </div>
    </div>
  </div>

  <div v-else class="empty-state card">
    <div class="icon">☰</div>
    <p>Belum ada percakapan masuk.</p>
  </div>
</template>

