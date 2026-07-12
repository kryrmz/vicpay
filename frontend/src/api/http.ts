/**
 * Adapter HTTP real contra el backend de VicPay. Implementa el mismo contrato
 * `Api` que el mock. Decisiones de seguridad:
 *  - El access token vive SOLO en memoria de este modulo (nunca en localStorage).
 *  - El refresh token viaja en una cookie httpOnly que el navegador adjunta solo
 *    en same-origin; por eso cada request usa `credentials: 'include'` y la app se
 *    sirve tras un proxy que expone el backend en la misma raiz `/api` (patron BFF).
 *  - Ante un 401 se intenta un unico refresh (rota la sesion) y se reintenta.
 */
import { ApiError } from './types'
import type {
  Api,
  CurrencyCode,
  PendingRegistration,
  Session,
  Transaction,
  TransactionCategory,
  TransferResult,
  User,
  Wallet,
} from './types'

interface Envelope<T> {
  data?: T
  error?: { code: string; message: string }
}

interface SessionPayload {
  user: User
  accessToken: string
}

// Token de acceso en memoria. Se pierde al recargar; la sesion se restaura via
// la cookie de refresh (bootstrap), no persistiendo el token.
let accessToken: string | null = null

/** Categorias validas que el backend puede enviar; el resto cae a 'transfer'. */
const KNOWN_CATEGORIES: readonly TransactionCategory[] = [
  'transfer',
  'topup',
  'savings',
  'payment',
  'refund',
]

function normalizeCategory(value: unknown): TransactionCategory {
  return KNOWN_CATEGORIES.includes(value as TransactionCategory)
    ? (value as TransactionCategory)
    : 'transfer'
}

export function createHttpApi(baseUrl: string): Api {
  const base = baseUrl.replace(/\/$/, '')

  async function parse<T>(res: Response): Promise<T> {
    if (res.status === 204) {
      return undefined as T
    }
    let body: Envelope<T>
    try {
      body = (await res.json()) as Envelope<T>
    } catch {
      throw new ApiError('Respuesta invalida del servidor.')
    }
    if (!res.ok || body.error) {
      throw new ApiError(body.error?.message ?? 'Ocurrio un error inesperado.')
    }
    return body.data as T
  }

  function request(path: string, init: RequestInit, withAuth: boolean): Promise<Response> {
    const headers = new Headers(init.headers)
    if (init.body) {
      headers.set('Content-Type', 'application/json')
    }
    if (withAuth && accessToken) {
      headers.set('Authorization', `Bearer ${accessToken}`)
    }
    return fetch(`${base}${path}`, { ...init, headers, credentials: 'include' })
  }

  async function refresh(): Promise<boolean> {
    const res = await request('/auth/refresh', { method: 'POST' }, false)
    if (!res.ok) {
      accessToken = null
      return false
    }
    const session = await parse<SessionPayload>(res)
    accessToken = session.accessToken
    return true
  }

  /** Ejecuta una request autenticada, reintentando una vez tras refrescar en 401. */
  async function authed<T>(path: string, init: RequestInit): Promise<T> {
    let res = await request(path, init, true)
    if (res.status === 401 && (await refresh())) {
      res = await request(path, init, true)
    }
    return parse<T>(res)
  }

  return {
    async register(phone, password): Promise<PendingRegistration> {
      const res = await request(
        '/auth/register',
        { method: 'POST', body: JSON.stringify({ phone, password }) },
        false,
      )
      return parse<PendingRegistration>(res)
    },

    async verifyPhone(pendingUserId, code): Promise<Session> {
      const res = await request(
        '/auth/verify-phone',
        { method: 'POST', body: JSON.stringify({ pendingUserId, code }) },
        false,
      )
      const payload = await parse<SessionPayload>(res)
      accessToken = payload.accessToken
      return payload
    },

    async login(phone, password): Promise<Session> {
      const res = await request(
        '/auth/login',
        { method: 'POST', body: JSON.stringify({ phone, password }) },
        false,
      )
      const payload = await parse<SessionPayload>(res)
      accessToken = payload.accessToken
      return payload
    },

    async logout(): Promise<void> {
      try {
        await request('/auth/logout', { method: 'POST' }, false)
      } finally {
        accessToken = null
      }
    },

    async me(): Promise<User> {
      const payload = await authed<{ user: User }>('/me', { method: 'GET' })
      return payload.user
    },

    async listWallets(): Promise<Wallet[]> {
      const payload = await authed<{ wallets: Wallet[] }>('/wallets', { method: 'GET' })
      return (payload.wallets ?? []).map((w) => ({
        currency: w.currency as CurrencyCode,
        balanceMinor: w.balanceMinor,
      }))
    },

    async listTransactions(): Promise<Transaction[]> {
      const payload = await authed<{ transactions: RawTransaction[] }>('/transactions', {
        method: 'GET',
      })
      return (payload.transactions ?? []).map((t) => ({
        id: t.id,
        kind: t.kind,
        counterparty: t.counterparty,
        amountMinor: t.amountMinor,
        currency: t.currency as CurrencyCode,
        createdAt: t.createdAt,
        category: normalizeCategory(t.category),
      }))
    },

    async transfer(toPhone, amountMinor, currency, idempotencyKey): Promise<TransferResult> {
      return authed<TransferResult>('/transfers', {
        method: 'POST',
        body: JSON.stringify({ toPhone, amountMinor, currency, idempotencyKey }),
      })
    },

    async topUp(amountMinor, currency, idempotencyKey): Promise<TransferResult> {
      return authed<TransferResult>('/wallets/topup', {
        method: 'POST',
        body: JSON.stringify({ amountMinor, currency, idempotencyKey }),
      })
    },
  }
}

/** Forma cruda de una transaccion tal como llega del backend (category opcional). */
interface RawTransaction {
  id: string
  kind: 'in' | 'out'
  counterparty: string
  amountMinor: number
  currency: string
  createdAt: string
  category?: string
}
