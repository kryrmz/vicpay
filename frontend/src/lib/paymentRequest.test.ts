import { describe, expect, it } from 'vitest'
import { decodePaymentRequest, encodePaymentRequest } from './paymentRequest'

describe('paymentRequest', () => {
  it('round-trip: decode(encode(x)) === x', () => {
    const req = { toUserId: 'usr_abc123', amountMinor: 2500, currency: 'USD' as const }
    const decoded = decodePaymentRequest(encodePaymentRequest(req))
    expect(decoded).toEqual(req)
  })

  it('codifica con el esquema vicpay:pay', () => {
    const encoded = encodePaymentRequest({ toUserId: 'u1', amountMinor: 100, currency: 'CRC' })
    expect(encoded).toContain('vicpay:pay?')
    expect(encoded).toContain('to=u1')
    expect(encoded).toContain('amt=100')
    expect(encoded).toContain('cur=CRC')
  })

  it('rechaza codigos invalidos', () => {
    const bad = [
      '',
      'https://example.com',
      'vicpay:pay?to=u1&amt=abc&cur=USD',
      'vicpay:pay?to=u1&amt=0&cur=USD',
      'vicpay:pay?to=u1&amt=-5&cur=USD',
      'vicpay:pay?to=u1&amt=100&cur=EUR',
      'vicpay:pay?amt=100&cur=USD',
    ]
    for (const code of bad) {
      expect(decodePaymentRequest(code)).toBeNull()
    }
  })
})
