import { afterEach, describe, expect, it, vi } from 'vitest'
import { createHttpApi } from './http'
import { ApiError } from './types'

interface FakeResponse {
  ok: boolean
  status: number
  json: () => Promise<unknown>
}

function ok(data: unknown): FakeResponse {
  return { ok: true, status: 200, json: async () => ({ data }) }
}

function fail(status: number, message: string): FakeResponse {
  return { ok: false, status, json: async () => ({ error: { code: 'x', message } }) }
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('createHttpApi', () => {
  it('stores the access token from login and sends it as a Bearer header', async () => {
    const calls: Array<{ url: string; headers: Headers }> = []
    const fetchMock = vi.fn(async (url: string, init: RequestInit) => {
      calls.push({ url, headers: new Headers(init.headers) })
      if (url.endsWith('/auth/login')) {
        return ok({ user: { id: 'u1', phoneMasked: '+506****34', kycLevel: 0 }, accessToken: 'tok-123' })
      }
      return ok({ wallets: [{ currency: 'USD', balanceMinor: 5000 }] })
    })
    vi.stubGlobal('fetch', fetchMock)

    const api = createHttpApi('/api')
    await api.login('+50688881234', 'hunter2go1234')
    const wallets = await api.listWallets()

    expect(wallets).toEqual([{ currency: 'USD', balanceMinor: 5000 }])
    const walletCall = calls.find((c) => c.url.endsWith('/wallets'))
    expect(walletCall?.headers.get('Authorization')).toBe('Bearer tok-123')
  })

  it('refreshes once on 401 and retries the original request', async () => {
    let walletHits = 0
    const fetchMock = vi.fn(async (url: string) => {
      if (url.endsWith('/wallets')) {
        walletHits += 1
        if (walletHits === 1) {
          return fail(401, 'expired')
        }
        return ok({ wallets: [] })
      }
      if (url.endsWith('/auth/refresh')) {
        return ok({ user: { id: 'u1', phoneMasked: 'x', kycLevel: 0 }, accessToken: 'tok-new' })
      }
      return ok({})
    })
    vi.stubGlobal('fetch', fetchMock)

    const api = createHttpApi('/api')
    const wallets = await api.listWallets()

    expect(wallets).toEqual([])
    expect(walletHits).toBe(2) // failed once, retried after refresh
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/auth/refresh'),
      expect.objectContaining({ credentials: 'include' }),
    )
  })

  it('throws ApiError with the server message on an error envelope', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => fail(409, 'Ya existe una cuenta con ese numero.')),
    )
    const api = createHttpApi('/api')
    await expect(api.register('+50688881234', 'hunter2go1234')).rejects.toBeInstanceOf(ApiError)
    await expect(api.register('+50688881234', 'hunter2go1234')).rejects.toThrow(
      'Ya existe una cuenta con ese numero.',
    )
  })
})
