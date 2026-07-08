import { ArrowLeftRight, Banknote, PiggyBank, Receipt, Undo2 } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import type { TransactionCategory } from '../api/types'

export interface CategoryMeta {
  label: string
  icon: LucideIcon
}

const CATEGORY_META: Record<TransactionCategory, CategoryMeta> = {
  transfer: { label: 'Transferencia', icon: ArrowLeftRight },
  topup: { label: 'Recarga', icon: Banknote },
  savings: { label: 'Ahorro', icon: PiggyBank },
  payment: { label: 'Pago', icon: Receipt },
  refund: { label: 'Reembolso', icon: Undo2 },
}

export function getCategoryMeta(category: TransactionCategory): CategoryMeta {
  return CATEGORY_META[category]
}
