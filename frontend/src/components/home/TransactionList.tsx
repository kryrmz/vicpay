import { Link } from 'react-router-dom'
import type { Transaction } from '../../api/types'
import { Card } from '../ui/Card'
import { TransactionRow } from './TransactionRow'

export interface TransactionListProps {
  transactions: Transaction[]
  limit?: number
}

export function TransactionList({ transactions, limit = 5 }: TransactionListProps) {
  const visible = transactions.slice(0, limit)

  return (
    <Card>
      <div className="mb-1 flex items-center justify-between">
        <h2 className="text-base font-semibold text-fg">Movimientos recientes</h2>
        <Link to="/app/activity" className="text-sm font-medium text-secondary-600">
          Ver todo
        </Link>
      </div>
      {visible.length === 0 ? (
        <p className="py-6 text-center text-sm text-fg-muted">Todavia no tienes movimientos.</p>
      ) : (
        <ul className="divide-y divide-border">
          {visible.map((transaction) => (
            <TransactionRow key={transaction.id} transaction={transaction} />
          ))}
        </ul>
      )}
    </Card>
  )
}
