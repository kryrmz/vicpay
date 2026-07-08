import { create } from 'zustand'
import { api } from '../api'
import type { CurrencyCode, Transaction, Wallet } from '../api/types'

function toMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'Ocurrio un error inesperado.'
}

interface WalletState {
  wallets: Wallet[]
  transactions: Transaction[]
  selectedCurrency: CurrencyCode
  status: 'idle' | 'loading'
  error: string | null
  selectCurrency: (currency: CurrencyCode) => void
  fetchAll: () => Promise<void>
  reset: () => void
}

export const useWalletStore = create<WalletState>()((set, get) => ({
  wallets: [],
  transactions: [],
  selectedCurrency: 'USD',
  status: 'idle',
  error: null,

  selectCurrency(currency) {
    set({ selectedCurrency: currency })
  },

  async fetchAll() {
    set({ status: 'loading', error: null })
    try {
      const [wallets, transactions] = await Promise.all([api.listWallets(), api.listTransactions()])
      const stillSelected = wallets.some((wallet) => wallet.currency === get().selectedCurrency)
      set({
        wallets,
        transactions,
        selectedCurrency: stillSelected ? get().selectedCurrency : (wallets[0]?.currency ?? 'USD'),
        status: 'idle',
      })
    } catch (error) {
      set({ status: 'idle', error: toMessage(error) })
    }
  },

  reset() {
    set({ wallets: [], transactions: [], selectedCurrency: 'USD', status: 'idle', error: null })
  },
}))
