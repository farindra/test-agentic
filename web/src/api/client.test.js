import { describe, it, expect, beforeEach, vi } from 'vitest'
import { api, getToken, setToken, ApiError } from './client'

describe('api client', () => {
  beforeEach(() => {
    localStorage.clear()
    global.fetch = vi.fn()
  })

  it('sends Authorization header when token is set', async () => {
    setToken('abc123')
    global.fetch.mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), { status: 200, headers: { 'content-type': 'application/json' } })
    )

    await api.get('/bots')

    const [, opts] = global.fetch.mock.calls[0]
    expect(opts.headers['Authorization']).toBe('Bearer abc123')
  })

  it('omits Authorization header when no token', async () => {
    setToken('')
    global.fetch.mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), { status: 200, headers: { 'content-type': 'application/json' } })
    )

    await api.get('/bots')

    const [, opts] = global.fetch.mock.calls[0]
    expect(opts.headers['Authorization']).toBeUndefined()
  })

  it('throws ApiError with server message on non-ok response', async () => {
    global.fetch.mockResolvedValue(
      new Response(JSON.stringify({ error: 'nama wajib' }), { status: 400, headers: { 'content-type': 'application/json' } })
    )

    await expect(api.post('/bots', {})).rejects.toThrow('nama wajib')
  })

  it('clears token on 401 response', async () => {
    setToken('expired-token')
    global.fetch.mockResolvedValue(
      new Response(JSON.stringify({ error: 'unauthorized' }), { status: 401, headers: { 'content-type': 'application/json' } })
    )

    await expect(api.get('/bots')).rejects.toBeInstanceOf(ApiError)
    expect(getToken()).toBe('')
  })

  it('throws a clear message when the network fails (no internet)', async () => {
    global.fetch.mockRejectedValue(new TypeError('Failed to fetch'))

    await expect(api.get('/bots')).rejects.toThrow('periksa koneksi internet')
  })

  it('throws a timeout-specific message when fetch is aborted', async () => {
    const abortError = new Error('aborted')
    abortError.name = 'AbortError'
    global.fetch.mockRejectedValue(abortError)

    await expect(api.get('/bots')).rejects.toThrow('Waktu tunggu habis')
  })

  it('passes an AbortSignal so requests can be cancelled on timeout', async () => {
    global.fetch.mockResolvedValue(
      new Response(JSON.stringify({}), { status: 200, headers: { 'content-type': 'application/json' } })
    )

    await api.get('/bots')

    const [, opts] = global.fetch.mock.calls[0]
    expect(opts.signal).toBeInstanceOf(AbortSignal)
  })
})
