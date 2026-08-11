import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from './auth'
import { getToken } from '../api/client'

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    global.fetch = vi.fn()
  })

  it('login stores token and username on success', async () => {
    global.fetch.mockResolvedValue(
      new Response(JSON.stringify({ token: 'tok-123', username: 'admin' }), {
        status: 200,
        headers: { 'content-type': 'application/json' },
      })
    )

    const store = useAuthStore()
    await store.login('admin', 'secret')

    expect(store.token).toBe('tok-123')
    expect(store.isAuthenticated).toBe(true)
    expect(getToken()).toBe('tok-123')
  })

  it('login throws and does not set token on failure', async () => {
    global.fetch.mockResolvedValue(
      new Response(JSON.stringify({ error: 'username atau password salah' }), {
        status: 401,
        headers: { 'content-type': 'application/json' },
      })
    )

    const store = useAuthStore()
    await expect(store.login('admin', 'salah')).rejects.toThrow('username atau password salah')
    expect(store.isAuthenticated).toBe(false)
  })

  it('logout clears token', async () => {
    const store = useAuthStore()
    store.token = 'something'
    store.logout()
    expect(store.isAuthenticated).toBe(false)
    expect(getToken()).toBe('')
  })
})
