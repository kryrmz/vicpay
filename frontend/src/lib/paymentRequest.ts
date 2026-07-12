/**
 * Solicitud de pago codificada en un QR. Lleva el id de usuario del cobrador
 * (nunca su telefono, para no exponer PII), el monto en unidades menores y la
 * moneda. Formato: vicpay:pay?to=<userId>&amt=<minor>&cur=<USD|CRC>
 */
import type { CurrencyCode } from '../api/types'

export interface PaymentRequest {
  toUserId: string
  amountMinor: number
  currency: CurrencyCode
}

const SCHEME = 'vicpay:pay'
const CURRENCIES: readonly CurrencyCode[] = ['USD', 'CRC']

export function encodePaymentRequest(req: PaymentRequest): string {
  const params = new URLSearchParams({
    to: req.toUserId,
    amt: String(req.amountMinor),
    cur: req.currency,
  })
  return `${SCHEME}?${params.toString()}`
}

/** Decodifica un codigo de pago; devuelve null si es invalido. */
export function decodePaymentRequest(input: string): PaymentRequest | null {
  const trimmed = input.trim()
  const prefix = `${SCHEME}?`
  if (!trimmed.startsWith(prefix)) {
    return null
  }
  const params = new URLSearchParams(trimmed.slice(prefix.length))
  const to = params.get('to')
  const amt = params.get('amt')
  const cur = params.get('cur')
  if (!to || !amt || !cur) {
    return null
  }
  if (!/^\d+$/.test(amt)) {
    return null
  }
  const amountMinor = Number(amt)
  if (!Number.isInteger(amountMinor) || amountMinor <= 0) {
    return null
  }
  if (!CURRENCIES.includes(cur as CurrencyCode)) {
    return null
  }
  return { toUserId: to, amountMinor, currency: cur as CurrencyCode }
}
