/**
 * Todo el dinero se representa en unidades menores (minor units) como
 * enteros: centimos para USD, centimos de colon para CRC. Nunca floats.
 * La unica forma de convertir a texto legible es `formatMoney` (src/lib/money.ts).
 */

export type CurrencyCode = 'USD' | 'CRC'

export type KycLevel = 0 | 1 | 2

/** Perfil de usuario expuesto por la API. Nunca incluye PII cruda (sin telefono completo, sin password). */
export interface User {
  id: string
  phoneMasked: string
  kycLevel: KycLevel
}

export interface Session {
  user: User
  accessToken: string
}

export interface PendingRegistration {
  pendingUserId: string
}

export interface Wallet {
  currency: CurrencyCode
  balanceMinor: number
}

export type TransactionKind = 'in' | 'out'

export type TransactionCategory = 'transfer' | 'topup' | 'savings' | 'payment' | 'refund'

export interface Transaction {
  id: string
  kind: TransactionKind
  counterparty: string
  amountMinor: number
  currency: CurrencyCode
  /** Fecha ISO 8601. */
  createdAt: string
  category: TransactionCategory
}

export class ApiError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'ApiError'
  }
}

/**
 * Contrato de la API de VicPay. La app solo conoce esta interfaz; la
 * implementacion real (mock hoy, HTTP manana) vive detras de ella.
 *
 * Nota de diseno: los metodos autenticados (me/listWallets/listTransactions/
 * logout) no reciben el access token como parametro explicito. El token vive
 * en memoria en el auth store y, en un backend real, viajaria en el header
 * Authorization de cada request via un cliente HTTP compartido. El adapter
 * mock (./mock.ts) simula esa sesion "actual" internamente para no filtrar
 * detalles de transporte en este contrato.
 */
export interface Api {
  register(phone: string, password: string): Promise<PendingRegistration>
  verifyPhone(pendingUserId: string, code: string): Promise<Session>
  login(phone: string, password: string): Promise<Session>
  logout(): Promise<void>
  me(): Promise<User>
  listWallets(): Promise<Wallet[]>
  listTransactions(): Promise<Transaction[]>
}
