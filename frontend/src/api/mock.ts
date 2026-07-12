import { ApiError } from './types'
import type {
  Api,
  CurrencyCode,
  PendingRegistration,
  Session,
  Transaction,
  TransferResult,
  User,
  Wallet,
} from './types'

/** Codigo de un solo uso fijo para el mock. Nunca se devuelve al cliente. */
const FIXED_OTP_CODE = '000000'

/** Latencia simulada de red, en milisegundos. */
const LATENCY_MS = 350

const E164_PATTERN = /^\+[1-9]\d{7,14}$/

function delay(ms: number = LATENCY_MS): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function generateId(prefix: string): string {
  return `${prefix}_${Math.random().toString(36).slice(2, 11)}${Date.now().toString(36)}`
}

function maskPhone(phone: string): string {
  const visibleDigits = 2
  const tail = phone.slice(-visibleDigits)
  const maskedLength = Math.max(phone.length - visibleDigits - 1, 4)
  return `${phone.slice(0, 1)}${'*'.repeat(maskedLength)}${tail}`
}

interface StoredUser {
  id: string
  phone: string
  password: string
  phoneMasked: string
  kycLevel: 0 | 1 | 2
}

interface PendingRecord {
  pendingUserId: string
  phone: string
  password: string
  code: string
}

function seedWallets(currency: CurrencyCode, balanceMinor: number): Wallet {
  return { currency, balanceMinor }
}

function emptyWalletsFor(): Wallet[] {
  return [seedWallets('USD', 0), seedWallets('CRC', 0)]
}

function demoWalletsFor(): Wallet[] {
  return [seedWallets('USD', 128450), seedWallets('CRC', 645000)]
}

function demoTransactionsFor(): Transaction[] {
  const now = Date.now()
  const day = 24 * 60 * 60 * 1000
  return [
    {
      id: generateId('tx'),
      kind: 'in',
      counterparty: 'Nomina Acme Corp',
      amountMinor: 85000,
      currency: 'USD',
      createdAt: new Date(now - 1 * day).toISOString(),
      category: 'transfer',
    },
    {
      id: generateId('tx'),
      kind: 'out',
      counterparty: 'Super La Colonia',
      amountMinor: 12300,
      currency: 'CRC',
      createdAt: new Date(now - 2 * day).toISOString(),
      category: 'payment',
    },
    {
      id: generateId('tx'),
      kind: 'out',
      counterparty: 'Ahorro meta viaje',
      amountMinor: 20000,
      currency: 'USD',
      createdAt: new Date(now - 3 * day).toISOString(),
      category: 'savings',
    },
    {
      id: generateId('tx'),
      kind: 'in',
      counterparty: 'Recarga tarjeta BCT',
      amountMinor: 50000,
      currency: 'CRC',
      createdAt: new Date(now - 5 * day).toISOString(),
      category: 'topup',
    },
    {
      id: generateId('tx'),
      kind: 'in',
      counterparty: 'Reembolso pedido #4821',
      amountMinor: 4599,
      currency: 'USD',
      createdAt: new Date(now - 6 * day).toISOString(),
      category: 'refund',
    },
    {
      id: generateId('tx'),
      kind: 'out',
      counterparty: 'Envio a Maria Rojas',
      amountMinor: 15000,
      currency: 'USD',
      createdAt: new Date(now - 8 * day).toISOString(),
      category: 'transfer',
    },
  ]
}

/** "Base de datos" en memoria del mock. Se reinicia en cada carga de la app. */
class MockDatabase {
  usersByPhone = new Map<string, StoredUser>()
  usersById = new Map<string, StoredUser>()
  pendingByToken = new Map<string, PendingRecord>()
  walletsByUserId = new Map<string, Wallet[]>()
  transactionsByUserId = new Map<string, Transaction[]>()
  idempotency = new Map<string, TransferResult>()
  currentUserId: string | null = null

  constructor() {
    this.seedDemoUser()
  }

  private seedDemoUser(): void {
    const phone = '+50688888888'
    const user: StoredUser = {
      id: generateId('usr'),
      phone,
      password: 'VicPay#2026',
      phoneMasked: maskPhone(phone),
      kycLevel: 1,
    }
    this.usersByPhone.set(phone, user)
    this.usersById.set(user.id, user)
    this.walletsByUserId.set(user.id, demoWalletsFor())
    this.transactionsByUserId.set(user.id, demoTransactionsFor())
  }

  toPublicUser(user: StoredUser): User {
    return { id: user.id, phoneMasked: user.phoneMasked, kycLevel: user.kycLevel }
  }
}

const db = new MockDatabase()

function requireSession(): StoredUser {
  if (!db.currentUserId) {
    throw new ApiError('No hay una sesion activa.')
  }
  const user = db.usersById.get(db.currentUserId)
  if (!user) {
    throw new ApiError('No hay una sesion activa.')
  }
  return user
}

function walletBalance(userId: string, currency: CurrencyCode): number {
  return (db.walletsByUserId.get(userId) ?? []).find((w) => w.currency === currency)?.balanceMinor ?? 0
}

function adjustWallet(userId: string, currency: CurrencyCode, deltaMinor: number): number {
  const wallets = db.walletsByUserId.get(userId) ?? []
  const existing = wallets.find((w) => w.currency === currency)
  if (existing) {
    existing.balanceMinor += deltaMinor
    db.walletsByUserId.set(userId, wallets)
    return existing.balanceMinor
  }
  const created: Wallet = { currency, balanceMinor: deltaMinor }
  db.walletsByUserId.set(userId, [...wallets, created])
  return created.balanceMinor
}

function addTransaction(userId: string, tx: Transaction): void {
  const txs = db.transactionsByUserId.get(userId) ?? []
  db.transactionsByUserId.set(userId, [tx, ...txs])
}

/** Ejecuta el movimiento una vez resuelto el destinatario (saldo, asientos, idempotencia). */
function executeTransfer(
  sender: StoredUser,
  recipient: StoredUser,
  amountMinor: number,
  currency: CurrencyCode,
  idempotencyKey?: string,
): TransferResult {
  if (idempotencyKey) {
    const prior = db.idempotency.get(idempotencyKey)
    if (prior) {
      return prior
    }
  }
  if (recipient.id === sender.id) {
    throw new ApiError('No puedes enviarte dinero a ti mismo.')
  }
  if (amountMinor <= 0) {
    throw new ApiError('El monto debe ser mayor que cero.')
  }
  if (walletBalance(sender.id, currency) < amountMinor) {
    throw new ApiError('Saldo insuficiente.')
  }
  const newBalance = adjustWallet(sender.id, currency, -amountMinor)
  adjustWallet(recipient.id, currency, amountMinor)
  const now = new Date().toISOString()
  addTransaction(sender.id, {
    id: generateId('tx'),
    kind: 'out',
    counterparty: recipient.phoneMasked,
    amountMinor,
    currency,
    createdAt: now,
    category: 'transfer',
  })
  addTransaction(recipient.id, {
    id: generateId('tx'),
    kind: 'in',
    counterparty: sender.phoneMasked,
    amountMinor,
    currency,
    createdAt: now,
    category: 'transfer',
  })
  const result: TransferResult = { postingId: generateId('pst'), newBalanceMinor: newBalance, currency }
  if (idempotencyKey) {
    db.idempotency.set(idempotencyKey, result)
  }
  return result
}

export const mockApi: Api = {
  async register(phone, password) {
    await delay()
    if (!E164_PATTERN.test(phone)) {
      throw new ApiError('El telefono debe estar en formato internacional E.164, por ejemplo +50688881234.')
    }
    if (password.length < 8) {
      throw new ApiError('La contrasena debe tener al menos 8 caracteres.')
    }
    if (db.usersByPhone.has(phone)) {
      throw new ApiError('Ya existe una cuenta con ese numero de telefono.')
    }

    const pendingUserId = generateId('pending')
    db.pendingByToken.set(pendingUserId, { pendingUserId, phone, password, code: FIXED_OTP_CODE })

    const result: PendingRegistration = { pendingUserId }
    return result
  },

  async verifyPhone(pendingUserId, code) {
    await delay()
    const pending = db.pendingByToken.get(pendingUserId)
    if (!pending) {
      throw new ApiError('El registro pendiente expiro o no existe. Inicia el registro de nuevo.')
    }
    if (code !== pending.code) {
      throw new ApiError('El codigo ingresado es incorrecto.')
    }

    const user: StoredUser = {
      id: generateId('usr'),
      phone: pending.phone,
      password: pending.password,
      phoneMasked: maskPhone(pending.phone),
      kycLevel: 0,
    }
    db.usersByPhone.set(user.phone, user)
    db.usersById.set(user.id, user)
    db.walletsByUserId.set(user.id, emptyWalletsFor())
    db.transactionsByUserId.set(user.id, [])
    db.pendingByToken.delete(pendingUserId)
    db.currentUserId = user.id

    const session: Session = { user: db.toPublicUser(user), accessToken: generateId('tok') }
    return session
  },

  async login(phone, password) {
    await delay()
    const user = db.usersByPhone.get(phone)
    if (!user || user.password !== password) {
      throw new ApiError('Telefono o contrasena incorrectos.')
    }
    db.currentUserId = user.id
    const session: Session = { user: db.toPublicUser(user), accessToken: generateId('tok') }
    return session
  },

  async logout() {
    await delay(120)
    db.currentUserId = null
  },

  async me() {
    await delay(150)
    return db.toPublicUser(requireSession())
  },

  async listWallets() {
    await delay()
    const user = requireSession()
    return [...(db.walletsByUserId.get(user.id) ?? [])]
  },

  async listTransactions() {
    await delay()
    const user = requireSession()
    const transactions = db.transactionsByUserId.get(user.id) ?? []
    return [...transactions].sort((a, b) => b.createdAt.localeCompare(a.createdAt))
  },

  async transfer(toPhone, amountMinor, currency, idempotencyKey) {
    await delay()
    const sender = requireSession()
    if (!E164_PATTERN.test(toPhone)) {
      throw new ApiError('El telefono del destinatario debe estar en formato E.164.')
    }
    const recipient = db.usersByPhone.get(toPhone)
    if (!recipient) {
      throw new ApiError('No hay ninguna cuenta con ese numero.')
    }
    return executeTransfer(sender, recipient, amountMinor, currency, idempotencyKey)
  },

  async payRequest(toUserId, amountMinor, currency, idempotencyKey) {
    await delay()
    const sender = requireSession()
    const recipient = db.usersById.get(toUserId)
    if (!recipient) {
      throw new ApiError('El cobro apunta a una cuenta que no existe.')
    }
    return executeTransfer(sender, recipient, amountMinor, currency, idempotencyKey)
  },

  async topUp(amountMinor, currency, idempotencyKey) {
    await delay()
    const user = requireSession()
    if (amountMinor <= 0) {
      throw new ApiError('El monto debe ser mayor que cero.')
    }
    if (idempotencyKey) {
      const prior = db.idempotency.get(idempotencyKey)
      if (prior) {
        return prior
      }
    }
    const newBalance = adjustWallet(user.id, currency, amountMinor)
    addTransaction(user.id, {
      id: generateId('tx'),
      kind: 'in',
      counterparty: 'Recarga demo',
      amountMinor,
      currency,
      createdAt: new Date().toISOString(),
      category: 'topup',
    })
    const result: TransferResult = { postingId: generateId('pst'), newBalanceMinor: newBalance, currency }
    if (idempotencyKey) {
      db.idempotency.set(idempotencyKey, result)
    }
    return result
  },
}
