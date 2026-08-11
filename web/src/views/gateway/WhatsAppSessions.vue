<script setup>
import { ref, onMounted, reactive, onUnmounted } from 'vue'
import { api } from '../../api/client'

const sessions = ref([])
const bots = ref([])
const error = ref('')
const loading = ref(true)
const showCreate = ref(false)
const newLabel = ref('')

const qrModal = reactive({ show: false, sessionId: '', image: '', ttl: 0 })
let qrTimer = null
let statusTimer = null

async function load() {
  error.value = ''
  try {
    const [s, b] = await Promise.all([api.get('/gateway/whatsapp/sessions'), api.get('/bots')])
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
    await api.post('/gateway/whatsapp/sessions', { label: newLabel.value })
    newLabel.value = ''
    showCreate.value = false
    await load()
  } catch (e) {
    error.value = e.message
  }
}

async function openPairing(s) {
  qrModal.sessionId = s.id
  qrModal.show = true
  await fetchQR()
  pollStatus()
}

async function fetchQR() {
  try {
    const res = await api.get(`/gateway/whatsapp/sessions/${qrModal.sessionId}/qr`)
    qrModal.image = res.qr
    qrModal.ttl = res.ttl_sec
    clearTimeout(qrTimer)
    qrTimer = setTimeout(fetchQR, Math.max((res.ttl_sec - 3) * 1000, 5000))
  } catch (e) {
    // "sesi sudah tersambung" itu tandanya HP-nya BARU AJA berhasil connect
    // pas kita lagi nunggu QR — bukan error beneran, jadi jangan ditampilin
    // sebagai alert merah. Tabel tetap wajib di-refresh di dua-duanya biar
    // status "Disconnected" yang basi nggak nyangkut di UI.
    if (!e.message.includes('tersambung')) {
      error.value = e.message
    }
    closeQR()
  }
}

// Dipanggil tiap 3 detik selama modal QR kebuka: begitu status sesi jadi
// "connected", modal ditutup otomatis + tabel di-refresh. Tanpa ini, FE cuma
// nunggu sampai fetchQR() gagal (kode QR lama ditolak backend) buat nyadar
// sesi udah tersambung — bisa nunggu lama tergantung sisa TTL kode terakhir.
function pollStatus() {
  clearTimeout(statusTimer)
  statusTimer = setTimeout(async () => {
    if (!qrModal.show) return
    try {
      const res = await api.get(`/gateway/whatsapp/sessions/${qrModal.sessionId}/status`)
      if (res.state === 'connected') {
        closeQR()
        return
      }
    } catch (e) {
      // diemin — biar fetchQR() yang nanganin error fatal (sesi kehapus, dst)
    }
    pollStatus()
  }, 3000)
}

function closeQR() {
  qrModal.show = false
  clearTimeout(qrTimer)
  clearTimeout(statusTimer)
  load()
}

async function updateBinding(s) {
  try {
    await api.patch(`/gateway/whatsapp/sessions/${s.id}`, { bot_id: s.bot_id || '', auto_reply: s.auto_reply })
  } catch (e) {
    error.value = e.message
  }
}

async function remove(s) {
  if (!confirm(`Hapus sesi WhatsApp "${s.label || s.id}"?`)) return
  try {
    await api.del(`/gateway/whatsapp/sessions/${s.id}`)
    await load()
  } catch (e) {
    error.value = e.message
  }
}

onMounted(load)
onUnmounted(() => {
  clearTimeout(qrTimer)
  clearTimeout(statusTimer)
})
</script>

<template>
  <div class="page-header">
    <div>
      <h1 class="page-title">Chat Gateway — WhatsApp</h1>
      <p class="page-subtitle">Kelola sesi WhatsApp (whatsmeow) dan binding-nya ke chatbot.</p>
    </div>
    <button class="btn btn-primary" @click="showCreate = true">+ Sesi Baru</button>
  </div>

  <div v-if="error" class="alert alert-danger">{{ error }}</div>

  <div class="card" style="padding: 0">
    <div v-if="loading" class="loading-state"><span class="spinner"></span> Memuat sesi WhatsApp...</div>
    <div v-else-if="sessions.length" class="table-scroll">
    <table class="table">
      <thead>
        <tr><th>Label</th><th>Nomor</th><th>Status</th><th>Bot</th><th>Auto-Reply</th><th></th></tr>
      </thead>
      <tbody>
        <tr v-for="s in sessions" :key="s.id">
          <td><strong>{{ s.label || '(tanpa label)' }}</strong></td>
          <td>{{ s.wa_jid || '—' }}</td>
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
              <button class="btn btn-sm" v-if="s.status !== 'connected'" @click="openPairing(s)">Scan QR</button>
              <button class="btn btn-sm btn-danger" @click="remove(s)">Hapus</button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
    </div>
    <div v-else class="empty-state">
      <div class="icon">▣</div>
      <p>Belum ada sesi WhatsApp. Bikin sesi baru lalu scan QR dari HP kamu.</p>
    </div>
  </div>

  <div v-if="showCreate" class="modal-backdrop" @click.self="showCreate = false">
    <div class="modal">
      <h2 class="modal-title">Sesi WhatsApp Baru</h2>
      <form @submit.prevent="createSession">
        <div class="field">
          <label>Label (opsional)</label>
          <input class="input" v-model="newLabel" placeholder="mis. CS Toko A" />
        </div>
        <div class="modal-actions">
          <button type="button" class="btn" @click="showCreate = false">Batal</button>
          <button type="submit" class="btn btn-primary">Buat Sesi</button>
        </div>
      </form>
    </div>
  </div>

  <div v-if="qrModal.show" class="modal-backdrop" @click.self="closeQR">
    <div class="modal">
      <h2 class="modal-title">Scan QR WhatsApp</h2>
      <div class="qr-box">
        <img v-if="qrModal.image" :src="qrModal.image" alt="QR pairing" />
        <p class="text-muted">Buka WhatsApp di HP → Perangkat Tertaut → Tautkan Perangkat, lalu scan kode ini. Kode diganti otomatis tiap kedaluwarsa.</p>
      </div>
      <div class="modal-actions">
        <button class="btn btn-primary" @click="closeQR">Selesai</button>
      </div>
    </div>
  </div>
</template>

