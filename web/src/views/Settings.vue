<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api/client'

const settings = ref({})
const newKey = ref('')
const newValue = ref('')
const error = ref('')
const saved = ref(false)

async function load() {
  settings.value = await api.get('/settings')
}

async function save() {
  error.value = ''
  saved.value = false
  try {
    await api.put('/settings', settings.value)
    saved.value = true
    setTimeout(() => (saved.value = false), 2000)
  } catch (e) {
    error.value = e.message
  }
}

function addSetting() {
  if (!newKey.value.trim()) return
  settings.value = { ...settings.value, [newKey.value.trim()]: newValue.value }
  newKey.value = ''
  newValue.value = ''
}

function removeKey(k) {
  const copy = { ...settings.value }
  delete copy[k]
  settings.value = copy
}

onMounted(load)
</script>

<template>
  <div class="page-header">
    <div>
      <h1 class="page-title">Pengaturan</h1>
      <p class="page-subtitle">Key-value setting umum aplikasi.</p>
    </div>
  </div>

  <div v-if="error" class="alert alert-danger">{{ error }}</div>
  <div v-if="saved" class="alert alert-success">Pengaturan tersimpan.</div>

  <div class="card">
    <div v-for="(v, k) in settings" :key="k" class="field-row" style="align-items:flex-end">
      <div class="field">
        <label>{{ k }}</label>
        <input class="input" v-model="settings[k]" />
      </div>
      <button type="button" class="btn btn-sm btn-danger" style="margin-bottom:14px" @click="removeKey(k)">Hapus</button>
    </div>

    <div class="field-row" style="align-items:flex-end; border-top: 1px solid var(--border); padding-top: 14px; margin-top: 4px">
      <div class="field">
        <label>Key baru</label>
        <input class="input" v-model="newKey" placeholder="mis. default_bot_id" />
      </div>
      <div class="field">
        <label>Value</label>
        <input class="input" v-model="newValue" />
      </div>
      <button type="button" class="btn btn-sm" style="margin-bottom:14px" @click="addSetting">+ Tambah</button>
    </div>

    <div class="modal-actions" style="justify-content:flex-start; margin-top: 4px">
      <button class="btn btn-primary" @click="save">Simpan Semua</button>
    </div>
  </div>
</template>

