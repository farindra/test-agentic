<script setup>
import { ref, onMounted, reactive } from 'vue'
import { api } from '../api/client'

const bots = ref([])
const providers = ref([])
const error = ref('')
const loading = ref(true)
const showModal = ref(false)
const editing = ref(null)

const form = reactive({
  name: '', provider_id: '', model: '', system_prompt: '',
  temperature: 0.7, max_tokens: 1024, is_active: true,
})
const modelOptions = ref([])
const modelsError = ref('')
const loadingModels = ref(false)

// Ditaro sebagai konstanta script, BUKAN ditulis langsung di template — Vue
// nutup interpolasi {{ }} di kurung kurawal PERTAMA yang ketemu, jadi
// nulis literal "{{nama_variable}}" langsung di markup bikin parser bingung
// (nutup terlalu awal, unterminated string). Direferensi lewat satu pasang
// {{ }} tunggal, ini aman.
const VARIABLE_PLACEHOLDER_EXAMPLE = '{{nama_variable}}'

function providerName(id) {
  return providers.value.find((p) => p.id === id)?.name || '—'
}

// Ganti provider = daftar model punya provider sebelumnya jadi gak relevan,
// jangan sampai ke-simpen diem-diem buat provider yang baru dipilih.
function onProviderChange() {
  modelOptions.value = []
  modelsError.value = ''
}

async function loadModels() {
  const provider = providers.value.find((p) => p.id === form.provider_id)
  if (!provider) {
    modelsError.value = 'Pilih AI Provider dulu.'
    return
  }
  modelsError.value = ''
  loadingModels.value = true
  try {
    const res = await api.post('/providers/models', {
      type: provider.type,
      base_url: provider.base_url,
      provider_id: provider.id,
    })
    modelOptions.value = res.models || []
    if (modelOptions.value.length === 0) {
      modelsError.value = 'Provider tidak mengembalikan model apa pun.'
    } else if (!form.model) {
      form.model = modelOptions.value[0]
    }
  } catch (e) {
    modelsError.value = e.message
  } finally {
    loadingModels.value = false
  }
}

async function load() {
  error.value = ''
  try {
    const [b, p] = await Promise.all([api.get('/bots'), api.get('/providers')])
    bots.value = b.bots || []
    providers.value = p.providers || []
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = null
  Object.assign(form, { name: '', provider_id: providers.value[0]?.id || '', model: '', system_prompt: '', temperature: 0.7, max_tokens: 1024, is_active: true })
  modelOptions.value = []
  modelsError.value = ''
  showModal.value = true
}

function openEdit(b) {
  editing.value = b
  Object.assign(form, { ...b })
  modelOptions.value = []
  modelsError.value = ''
  showModal.value = true
}

async function save() {
  try {
    const payload = { ...form, temperature: Number(form.temperature), max_tokens: Number(form.max_tokens) }
    if (editing.value) {
      await api.put(`/bots/${editing.value.id}`, payload)
    } else {
      await api.post('/bots', payload)
    }
    showModal.value = false
    await load()
  } catch (e) {
    error.value = e.message
  }
}

async function remove(b) {
  if (!confirm(`Hapus chatbot "${b.name}"?`)) return
  try {
    await api.del(`/bots/${b.id}`)
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
      <h1 class="page-title">Chatbot</h1>
      <p class="page-subtitle">Susun persona bot: system prompt, provider AI, dan parameter model.</p>
    </div>
    <button class="btn btn-primary" :disabled="!providers.length" @click="openCreate">+ Tambah Chatbot</button>
  </div>

  <div v-if="error" class="alert alert-danger">{{ error }}</div>
  <div v-if="!providers.length" class="alert alert-danger">
    Belum ada AI provider. Tambahkan provider dulu di menu "AI Provider".
  </div>

  <div class="card" style="padding: 0">
    <div v-if="loading" class="loading-state"><span class="spinner"></span> Memuat chatbot...</div>
    <div v-else-if="bots.length" class="table-scroll">
    <table class="table">
      <thead>
        <tr><th>Nama</th><th>Provider</th><th>Model</th><th>Status</th><th></th></tr>
      </thead>
      <tbody>
        <tr v-for="b in bots" :key="b.id">
          <td><strong>{{ b.name }}</strong></td>
          <td>{{ providerName(b.provider_id) }}</td>
          <td>{{ b.model || '—' }}</td>
          <td>
            <span class="badge" :class="b.is_active ? 'badge-connected' : 'badge-neutral'">
              {{ b.is_active ? 'Aktif' : 'Nonaktif' }}
            </span>
          </td>
          <td>
            <div style="display:flex; gap:6px; justify-content:flex-end">
              <button class="btn btn-sm" @click="openEdit(b)">Edit</button>
              <button class="btn btn-sm btn-danger" @click="remove(b)">Hapus</button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
    </div>
    <div v-else class="empty-state">
      <div class="icon">✦</div>
      <p>Belum ada chatbot. Bikin satu buat mulai testing di Playground.</p>
    </div>
  </div>

  <div v-if="showModal" class="modal-backdrop" @click.self="showModal = false">
    <div class="modal" style="max-width: 560px">
      <h2 class="modal-title">{{ editing ? 'Edit' : 'Tambah' }} Chatbot</h2>
      <form @submit.prevent="save">
        <div class="field-row">
          <div class="field">
            <label>Nama Bot</label>
            <input class="input" v-model="form.name" required placeholder="mis. CS Bot" />
          </div>
          <div class="field">
            <label>AI Provider</label>
            <select class="select" v-model="form.provider_id" required @change="onProviderChange">
              <option v-for="p in providers" :key="p.id" :value="p.id">{{ p.name }}</option>
            </select>
          </div>
        </div>
        <div class="field">
          <div style="display:flex; justify-content:space-between; align-items:center">
            <label style="margin:0">Model (kosongkan buat pakai default provider)</label>
            <button type="button" class="btn btn-sm btn-ghost" :disabled="loadingModels" @click="loadModels">
              {{ loadingModels ? 'Mengambil...' : '⟳ Ambil daftar model' }}
            </button>
          </div>
          <select v-if="modelOptions.length" class="select" v-model="form.model">
            <option value="">— pakai default provider —</option>
            <option v-for="m in modelOptions" :key="m" :value="m">{{ m }}</option>
          </select>
          <input v-else class="input" v-model="form.model" placeholder="mis. gpt-4o-mini" />
          <span v-if="modelsError" class="text-muted" style="color: var(--danger); font-size: 12px">{{ modelsError }}</span>
          <span v-else-if="modelOptions.length" class="text-muted" style="font-size: 12px">
            {{ modelOptions.length }} model ditemukan dari provider ini.
            <a href="#" style="color: var(--accent)" @click.prevent="modelOptions = []">Ketik manual</a>
          </span>
          <span v-else class="text-muted" style="font-size: 12px">
            Pilih provider, lalu klik "Ambil daftar model" — atau ketik manual.
          </span>
        </div>
        <div class="field">
          <label>System Prompt</label>
          <textarea class="textarea" v-model="form.system_prompt" rows="5" placeholder="Kamu adalah asisten customer service yang ramah..."></textarea>
          <span class="text-muted" style="font-size: 12px">
            Bisa pakai <code>{{ VARIABLE_PLACEHOLDER_EXAMPLE }}</code> — kelola nilainya di menu Variables.
          </span>
        </div>
        <div class="field-row">
          <div class="field">
            <label>Temperature</label>
            <input class="input" type="number" min="0" max="2" step="0.1" v-model="form.temperature" />
          </div>
          <div class="field">
            <label>Max Tokens</label>
            <input class="input" type="number" min="1" v-model="form.max_tokens" />
          </div>
        </div>
        <div class="field" style="flex-direction: row; align-items: center; gap: 10px">
          <label class="toggle">
            <input type="checkbox" v-model="form.is_active" />
            <span class="toggle-track"></span>
          </label>
          <label style="margin: 0">Aktifkan chatbot ini</label>
        </div>
        <div class="modal-actions">
          <button type="button" class="btn" @click="showModal = false">Batal</button>
          <button type="submit" class="btn btn-primary">Simpan</button>
        </div>
      </form>
    </div>
  </div>
</template>

