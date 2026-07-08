import { useEffect, useMemo, useState } from 'react'
import { ActivityFilters } from '../../components/activity/ActivityFilters'
import type { ActivityFilter } from '../../components/activity/ActivityFilters'
import { ActivitySummary } from '../../components/activity/ActivitySummary'
import { TransactionRow } from '../../components/home/TransactionRow'
import { useWalletStore } from '../../store/wallet.store'

export function ActivityPage() {
  const transactions = useWalletStore((state) => state.transactions)
  const fetchAll = useWalletStore((state) => state.fetchAll)
  const [filter, setFilter] = useState<ActivityFilter>('all')

  useEffect(() => {
    void fetchAll()
  }, [fetchAll])

  const filteredTransactions = useMemo(() => {
    if (filter === 'all') return transactions
    return transactions.filter((transaction) => transaction.kind === filter)
  }, [transactions, filter])

  return (
    <div className="flex flex-col gap-5 px-4 py-5">
      <header>
        <h1 className="text-xl font-bold text-fg">Actividad</h1>
      </header>

      <ActivityFilters value={filter} onChange={setFilter} />
      <ActivitySummary transactions={filteredTransactions} />

      {filteredTransactions.length === 0 ? (
        <p className="py-10 text-center text-sm text-fg-muted">No hay movimientos para este filtro.</p>
      ) : (
        <ul className="divide-y divide-border rounded-xl border border-border bg-surface-raised px-4">
          {filteredTransactions.map((transaction) => (
            <TransactionRow key={transaction.id} transaction={transaction} />
          ))}
        </ul>
      )}
    </div>
  )
}
