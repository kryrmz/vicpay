import { describe, expect, it, vi } from 'vitest'
import type { Api } from './types'

/** Reimporta el modulo mock para que cada test tenga su propia "base de datos" en memoria. */
async function freshApi(): Promise<Api> {
  vi.resetModules()
  const mod = await import('./mock')
  return mod.mockApi
}

describe('mockApi', () => {
  it('register() nunca devuelve el codigo OTP', async () => {
    const api = await freshApi()
    const result = await api.register('+50670009001', 'contrasenaSegura')
    expect(result).toEqual({ pendingUserId: expect.any(String) })
  })

  it('verifyPhone() rechaza un codigo incorrecto y no crea sesion', async () => {
    const api = await freshApi()
    const { pendingUserId } = await api.register('+50670009002', 'contrasenaSegura')
    await expect(api.verifyPhone(pendingUserId, '111111')).rejects.toThrow()
  })

  it('verifyPhone() con el codigo correcto entrega usuario y access token', async () => {
    const api = await freshApi()
    const { pendingUserId } = await api.register('+50670009003', 'contrasenaSegura')
    const session = await api.verifyPhone(pendingUserId, '000000')

    expect(session.accessToken).toBeTruthy()
    expect(session.user.kycLevel).toBe(0)
    expect(session.user.phoneMasked).toContain('*')
  })

  it('login() funciona con el usuario demo sembrado en el mock', async () => {
    const api = await freshApi()
    const session = await api.login('+50688888888', 'VicPay#2026')
    expect(session.accessToken).toBeTruthy()
  })

  it('login() rechaza credenciales incorrectas', async () => {
    const api = await freshApi()
    await expect(api.login('+50688888888', 'contrasena-mala')).rejects.toThrow()
  })

  it('listWallets() y listTransactions() exigen una sesion activa', async () => {
    const api = await freshApi()
    await expect(api.listWallets()).rejects.toThrow()
    await expect(api.listTransactions()).rejects.toThrow()
  })

  it('tras iniciar sesion, listWallets() y listTransactions() devuelven datos en unidades menores enteras', async () => {
    const api = await freshApi()
    await api.login('+50688888888', 'VicPay#2026')

    const wallets = await api.listWallets()
    const transactions = await api.listTransactions()

    expect(wallets.length).toBeGreaterThan(0)
    for (const wallet of wallets) {
      expect(Number.isInteger(wallet.balanceMinor)).toBe(true)
    }
    expect(transactions.length).toBeGreaterThan(0)
    for (const transaction of transactions) {
      expect(Number.isInteger(transaction.amountMinor)).toBe(true)
    }
  })
})
