<script setup>
import { ref, reactive, onMounted } from 'vue'
import { api } from '../api/client'

const variables = ref({})
const error = ref('')
const loading = ref(true)
const savingKey = ref('')
const savedFlash = reactive({})

const newKey = ref('')
const newValue = ref('')

// Ditaro sebagai konstanta script (bukan langsung ditulis di template) —
// Vue nutup interpolasi {{ }} di kurung kurawal PERTAMA yang ketemu, jadi
// literal "{{...}}" langsung di markup bikin parser salah nutup.
const PLACEHOLDER_SYNTAX_EXAMPLE = '{{nama_variable}}'
const PLACEHOLDER_USAGE_EXAMPLE = '{{jam_buka}}'

async function load() {
  error.value = ''
  try {
    variables.value = await api.get('/variables')
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function saveValue(key) {
  error.value = ''
  savingKey.value = key
  try {
    await api.put(`/variables/${encodeURIComponent(key)}`, { value: variables.value[key] })
    savedFlash[key] = true
    setTimeout(() => (savedFlash[key] = false), 1500)
  } catch (e) {
    error.value = e.message
  } finally {
    savingKey.value = ''
  }
}

async function removeVariable(key) {
  if (!confirm(`Hapus variable "${key}"?`)) return
  error.value = ''
  try {
    await api.del(`/variables/${encodeURIComponent(key)}`)
    const copy = { ...variables.value }
    delete copy[key]
    variables.value = copy
  } catch (e) {
    error.value = e.message
  }
}

async function addVariable() {
  const key = newKey.value.trim()
  if (!key) return
  error.value = ''
  try {
    await api.put(`/variables/${encodeURIComponent(key)}`, { value: newValue.value })
    variables.value = { ...variables.value, [key]: newValue.value }
    newKey.value = ''
    newValue.value = ''
  } catch (e) {
    error.value = e.message
  }
}

onMounted(load)
</script>

<template>
  <div class="page-header">
    <div>
      <h1 class="page-title">Variables</h1>
      <p class="page-subtitle">
        Variable custom (mis. jam buka, alamat) yang bisa dipakai di System Prompt chatbot
        pakai format <code>{{ PLACEHOLDER_SYNTAX_EXAMPLE }}</code>.
      </p>
    </div>
  </div>

  <div v-if="error" class="alert alert-danger">{{ error }}</div>

  <div class="card">
    <div v-if="loading" class="loading-state" style="padding: 24px 0"><span class="spinner"></span> Memuat variables...</div>
    <div v-else-if="Object.keys(variables).length === 0" class="empty-state" style="padding: 24px 0">
      <p>Belum ada variable. Tambahkan di bawah, lalu pakai di System Prompt chatbot.</p>
    </div>

    <div v-for="(v, k) in variables" :key="k" class="field-row" style="align-items:flex-end">
      <div class="field" style="flex: 0 0 200px; min-width: 0">
        <label>Key</label>
        <input class="input" :value="k" disabled />
      </div>
      <div class="field" style="min-width: 0">
        <label>Value</label>
        <input class="input" v-model="variables[k]" @keyup.enter="saveValue(k)" />
      </div>
      <button type="button" class="btn btn-sm" style="margin-bottom:14px" :disabled="savingKey === k" @click="saveValue(k)">
        {{ savedFlash[k] ? '✓ Tersimpan' : savingKey === k ? 'Menyimpan...' : 'Simpan' }}
      </button>
      <button type="button" class="btn btn-sm btn-danger" style="margin-bottom:14px" @click="removeVariable(k)">Hapus</button>
    </div>

    <div class="field-row" style="align-items:flex-end; border-top: 1px solid var(--border); padding-top: 14px; margin-top: 4px">
      <div class="field" style="flex: 0 0 200px; min-width: 0">
        <label>Key baru</label>
        <input class="input" v-model="newKey" placeholder="mis. jam_buka" @keyup.enter="addVariable" />
      </div>
      <div class="field" style="min-width: 0">
        <label>Value</label>
        <input class="input" v-model="newValue" placeholder="mis. 09.00 - 17.00" @keyup.enter="addVariable" />
      </div>
      <button type="button" class="btn btn-sm btn-primary" style="margin-bottom:14px" :disabled="!newKey.trim()" @click="addVariable">
        + Tambah
      </button>
    </div>
  </div>

  <div class="card" style="margin-top: 16px">
    <p class="text-muted mt-0" style="margin-bottom: 0">
      Contoh pemakaian di System Prompt: <em>"Toko kami buka jam <code>{{ PLACEHOLDER_USAGE_EXAMPLE }}</code>."</em>
      — placeholder yang key-nya gak ketemu akan dibiarkan apa adanya di prompt (biar gampang ketauan typo).
    </p>
  </div>
</template>
