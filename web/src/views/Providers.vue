<script setup>
import { ref, onMounted, reactive } from 'vue'
import { api } from '../api/client'

const providers = ref([])
const error = ref('')
const showModal = ref(false)
const editing = ref(null)
const testResult = reactive({})
const testing = reactive({})

const form = reactive({ name: '', type: 'openai', base_url: '', api_key: '', default_model: '', is_active: true })
const modelOptions = ref([])
const modelsError = ref('')
const loadingModels = ref(false)

const PROVIDER_TYPES = [
  { value: 'openai', label: 'ChatGPT (OpenAI)' },
  { value: 'deepseek', label: 'DeepSeek' },
  { value: 'gemini', label: 'Gemini (Google)' },
  { value: 'ollama', label: 'Ollama (self-hosted)' },
  { value: 'custom', label: 'Custom (OpenAI-compatible)' },
]

async function load() {
  error.value = ''
  try {
    const res = await api.get('/providers')
    providers.value = res.providers || []
  } catch (e) {
    error.value = e.message
  }
}

function openCreate() {
  editing.value = null
  Object.assign(form, { name: '', type: 'openai', base_url: '', api_key: '', default_model: '', is_active: true })
  modelOptions.value = []
  modelsError.value = ''
  showModal.value = true
}

function openEdit(p) {
  editing.value = p
  Object.assign(form, { name: p.name, type: p.type, base_url: p.base_url, api_key: '', default_model: p.default_model, is_active: p.is_active })
  modelOptions.value = []
  modelsError.value = ''
  showModal.value = true
}

// Ganti tipe provider = daftar model lama (punya provider sebelumnya) jadi
// gak relevan lagi, jangan sampai kepilih diem-diem.
function onTypeChange() {
  modelOptions.value = []
  modelsError.value = ''
}

async function loadModels() {
  modelsError.value = ''
  loadingModels.value = true
  try {
    const res = await api.post('/providers/models', {
      type: form.type,
      base_url: form.base_url,
      api_key: form.api_key,
      provider_id: editing.value ? editing.value.id : null,
    })
    modelOptions.value = res.models || []
    if (modelOptions.value.length === 0) {
      modelsError.value = 'Provider tidak mengembalikan model apa pun.'
    } else if (!form.default_model) {
      form.default_model = modelOptions.value[0]
    }
  } catch (e) {
    modelsError.value = e.message
  } finally {
    loadingModels.value = false
  }
}

async function save() {
  try {
    if (editing.value) {
      await api.put(`/providers/${editing.value.id}`, form)
    } else {
      await api.post('/providers', form)
    }
    showModal.value = false
    await load()
  } catch (e) {
    error.value = e.message
  }
}

async function remove(p) {
  if (!confirm(`Hapus provider "${p.name}"?`)) return
  try {
    await api.del(`/providers/${p.id}`)
    await load()
  } catch (e) {
    error.value = e.message
  }
}

async function test(p) {
  testing[p.id] = true
  testResult[p.id] = null
  try {
    const res = await api.post(`/providers/${p.id}/test`, {})
    testResult[p.id] = { ok: true, message: res.reply }
  } catch (e) {
    testResult[p.id] = { ok: false, message: e.message }
  } finally {
    testing[p.id] = false
  }
}

onMounted(load)
</script>

<template>
  <div class="page-header">
    <div>
      <h1 class="page-title">AI Provider</h1>
      <p class="page-subtitle">Kelola koneksi ke ChatGPT, DeepSeek, Gemini, Ollama, atau endpoint custom lainnya.</p>
    </div>
    <button class="btn btn-primary" @click="openCreate">+ Tambah Provider</button>
  </div>

  <div v-if="error" class="alert alert-danger">{{ error }}</div>

  <div class="card" style="padding: 0">
    <table class="table" v-if="providers.length">
      <thead>
        <tr>
          <th>Nama</th><th>Tipe</th><th>Model Default</th><th>API Key</th><th>Status</th><th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="p in providers" :key="p.id">
          <td><strong>{{ p.name }}</strong></td>
          <td>{{ p.type }}</td>
          <td>{{ p.default_model || '—' }}</td>
          <td>{{ p.api_key_set ? p.api_key_preview : '—' }}</td>
          <td>
            <span class="badge" :class="p.is_active ? 'badge-connected' : 'badge-neutral'">
              {{ p.is_active ? 'Aktif' : 'Nonaktif' }}
            </span>
          </td>
          <td>
            <div style="display:flex; gap:6px; align-items:center; justify-content:flex-end">
              <span v-if="testResult[p.id]" :style="{ color: testResult[p.id].ok ? 'var(--success)' : 'var(--danger)', fontSize: '12px' }">
                {{ testResult[p.id].ok ? '✓ ' + testResult[p.id].message : '✗ ' + testResult[p.id].message }}
              </span>
              <button class="btn btn-sm" @click="test(p)" :disabled="testing[p.id]">
                {{ testing[p.id] ? 'Menguji...' : 'Test' }}
              </button>
              <button class="btn btn-sm" @click="openEdit(p)">Edit</button>
              <button class="btn btn-sm btn-danger" @click="remove(p)">Hapus</button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty-state">
      <div class="icon">⚙</div>
      <p>Belum ada AI provider. Tambahkan dulu sebelum bikin chatbot.</p>
    </div>
  </div>

  <div v-if="showModal" class="modal-backdrop" @click.self="showModal = false">
    <div class="modal">
      <h2 class="modal-title">{{ editing ? 'Edit' : 'Tambah' }} Provider</h2>
      <form @submit.prevent="save" autocomplete="off">
        <div class="field">
          <label>Nama</label>
          <input class="input" v-model="form.name" required placeholder="mis. OpenAI Produksi" autocomplete="off" />
        </div>
        <div class="field">
          <label>Tipe</label>
          <select class="select" v-model="form.type" @change="onTypeChange">
            <option v-for="t in PROVIDER_TYPES" :key="t.value" :value="t.value">{{ t.label }}</option>
          </select>
        </div>
        <div class="field">
          <label>Base URL {{ ['ollama', 'custom'].includes(form.type) ? '(wajib)' : '(opsional, isi kalau beda dari default)' }}</label>
          <input class="input" v-model="form.base_url" placeholder="https://..." autocomplete="off" />
        </div>
        <div class="field">
          <label>API Key {{ editing ? '(kosongkan kalau tidak diganti)' : '' }}</label>
          <input class="input" type="password" v-model="form.api_key" placeholder="sk-..." autocomplete="new-password" />
        </div>
        <div class="field">
          <div style="display:flex; justify-content:space-between; align-items:center">
            <label style="margin:0">Model Default</label>
            <button type="button" class="btn btn-sm btn-ghost" :disabled="loadingModels" @click="loadModels">
              {{ loadingModels ? 'Mengambil...' : '⟳ Ambil daftar model' }}
            </button>
          </div>
          <select v-if="modelOptions.length" class="select" v-model="form.default_model">
            <option v-for="m in modelOptions" :key="m" :value="m">{{ m }}</option>
          </select>
          <input v-else class="input" v-model="form.default_model" placeholder="mis. gpt-4o-mini" autocomplete="off" />
          <span v-if="modelsError" class="text-muted" style="color: var(--danger); font-size: 12px">{{ modelsError }}</span>
          <span v-else-if="modelOptions.length" class="text-muted" style="font-size: 12px">
            {{ modelOptions.length }} model ditemukan.
            <a href="#" style="color: var(--accent)" @click.prevent="modelOptions = []">Ketik manual</a>
          </span>
          <span v-else class="text-muted" style="font-size: 12px">
            Isi Base URL/API Key dulu (kalau perlu), baru klik "Ambil daftar model" — atau ketik manual.
          </span>
        </div>
        <div class="field" style="flex-direction: row; align-items: center; gap: 10px">
          <label class="toggle">
            <input type="checkbox" v-model="form.is_active" />
            <span class="toggle-track"></span>
          </label>
          <label style="margin: 0">Aktifkan provider ini</label>
        </div>
        <div class="modal-actions">
          <button type="button" class="btn" @click="showModal = false">Batal</button>
          <button type="submit" class="btn btn-primary">Simpan</button>
        </div>
      </form>
    </div>
  </div>
</template>

