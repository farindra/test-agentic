const TOKEN_KEY = 'test_agentic_token'

export function getToken() {
  return localStorage.getItem(TOKEN_KEY) || ''
}

export function setToken(token) {
  if (token) localStorage.setItem(TOKEN_KEY, token)
  else localStorage.removeItem(TOKEN_KEY)
}

class ApiError extends Error {
  constructor(message, status) {
    super(message)
    this.status = status
  }
}

// DEFAULT_TIMEOUT_MS: fetch() browser TIDAK punya timeout bawaan — kalau
// koneksi lambat/putus di tengah jalan, request bisa nggantung tanpa batas
// dan halaman kelihatan blank/gak respon selamanya tanpa pesan apa pun.
// AbortController di bawah ini yang mastiin request selalu berujung (sukses,
// gagal, atau timeout) dalam waktu wajar.
const DEFAULT_TIMEOUT_MS = 15000

async function request(method, path, body) {
  const headers = { 'Content-Type': 'application/json' }
  const token = getToken()
  if (token) headers['Authorization'] = `Bearer ${token}`

  const controller = new AbortController()
  const timeoutID = setTimeout(() => controller.abort(), DEFAULT_TIMEOUT_MS)

  let res
  try {
    res = await fetch(`/api${path}`, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
      signal: controller.signal,
    })
  } catch (e) {
    if (e.name === 'AbortError') {
      throw new ApiError('Waktu tunggu habis — periksa koneksi internet kamu.', 0)
    }
    throw new ApiError('Gagal terhubung ke server — periksa koneksi internet kamu.', 0)
  } finally {
    clearTimeout(timeoutID)
  }

  const isJSON = res.headers.get('content-type')?.includes('application/json')
  const data = isJSON ? await res.json().catch(() => ({})) : null

  if (!res.ok) {
    if (res.status === 401) setToken('')
    throw new ApiError(data?.error || `Request gagal (${res.status})`, res.status)
  }
  return data
}

export const api = {
  get: (path) => request('GET', path),
  post: (path, body) => request('POST', path, body),
  put: (path, body) => request('PUT', path, body),
  patch: (path, body) => request('PATCH', path, body),
  del: (path) => request('DELETE', path),
}

export { ApiError }
