import { getCategoryMeta } from '../../lib/transactionCategory'
import type { Transaction } from '../../api/types'
import { MoneyText } from '../ui/MoneyText'

export interface TransactionRowProps {
  transaction: Transaction
}

function formatDate(iso: string): string {
  return new Intl.DateTimeFormat('es-CR', { day: '2-digit', month: 'short' }).format(new Date(iso))
}

export function TransactionRow({ transaction }: TransactionRowProps) {
  const { label, icon: Icon } = getCategoryMeta(transaction.category)
  const isIncome = transaction.kind === 'in'

  return (
    <li className="flex items-center gap-3 py-3">
      <span
        className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-full ${
          isIncome ? 'bg-success/15 text-success' : 'bg-surface-sunken text-fg-secondary'
        }`}
      >
        <Icon size={18} aria-hidden="true" />
      </span>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-fg">{transaction.counterparty}</p>
        <p className="text-xs text-fg-muted">
          {label} · {formatDate(transaction.createdAt)}
        </p>
      </div>
      <MoneyText
        amountMinor={isIncome ? transaction.amountMinor : -transaction.amountMinor}
        currency={transaction.currency}
        tone={isIncome ? 'in' : 'out'}
        className="shrink-0 text-sm font-semibold"
      />
    </li>
  )
}
