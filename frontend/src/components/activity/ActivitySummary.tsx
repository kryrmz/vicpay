import type { Transaction } from '../../api/types'
import { summarizeByCurrency } from '../../lib/transactionSummary'
import { Card } from '../ui/Card'
import { MoneyText } from '../ui/MoneyText'

export interface ActivitySummaryProps {
  transactions: Transaction[]
}

export function ActivitySummary({ transactions }: ActivitySummaryProps) {
  const summaries = summarizeByCurrency(transactions)

  if (summaries.length === 0) {
    return null
  }

  return (
    <div className="flex flex-col gap-3">
      {summaries.map((summary) => (
        <Card key={summary.currency} className="grid grid-cols-3 gap-2 text-center">
          <div>
            <p className="text-xs text-fg-muted">Ingresos</p>
            <MoneyText amountMinor={summary.incomeMinor} currency={summary.currency} tone="in" className="text-sm font-semibold" />
          </div>
          <div>
            <p className="text-xs text-fg-muted">Egresos</p>
            <MoneyText amountMinor={summary.expenseMinor} currency={summary.currency} className="text-sm font-semibold text-fg" />
          </div>
          <div>
            <p className="text-xs text-fg-muted">Neto ({summary.currency})</p>
            <MoneyText
              amountMinor={summary.netMinor}
              currency={summary.currency}
              className="text-sm font-semibold text-fg"
            />
          </div>
        </Card>
      ))}
    </div>
  )
}
