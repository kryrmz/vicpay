import type { CurrencyCode, Transaction } from '../api/types'

export interface CurrencySummary {
  currency: CurrencyCode
  incomeMinor: number
  expenseMinor: number
  netMinor: number
}

/**
 * Agrupa ingresos/egresos/neto por moneda. Nunca se suman montos de distintas
 * monedas entre si (serian unidades incompatibles), por eso el resultado es
 * una lista, una entrada por cada moneda presente en `transactions`.
 */
export function summarizeByCurrency(transactions: Transaction[]): CurrencySummary[] {
  const summaryByCurrency = new Map<CurrencyCode, CurrencySummary>()

  for (const transaction of transactions) {
    const existing = summaryByCurrency.get(transaction.currency) ?? {
      currency: transaction.currency,
      incomeMinor: 0,
      expenseMinor: 0,
      netMinor: 0,
    }

    if (transaction.kind === 'in') {
      existing.incomeMinor += transaction.amountMinor
      existing.netMinor += transaction.amountMinor
    } else {
      existing.expenseMinor += transaction.amountMinor
      existing.netMinor -= transaction.amountMinor
    }

    summaryByCurrency.set(transaction.currency, existing)
  }

  return [...summaryByCurrency.values()]
}
