import { PiggyBank, QrCode, Send, Wallet } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { BottomSheet } from '../ui/BottomSheet'

interface QuickAction {
  key: string
  label: string
  icon: LucideIcon
  comingSoon?: boolean
}

const ACTIONS: QuickAction[] = [
  { key: 'send', label: 'Enviar', icon: Send, comingSoon: true },
  { key: 'collect', label: 'Cobrar', icon: QrCode },
  { key: 'topup', label: 'Recargar', icon: Wallet, comingSoon: true },
  { key: 'savings', label: 'Ahorros', icon: PiggyBank, comingSoon: true },
]

export function QuickActions() {
  const navigate = useNavigate()
  const [comingSoonAction, setComingSoonAction] = useState<QuickAction | null>(null)

  function handleActionClick(action: QuickAction): void {
    if (action.key === 'collect') {
      navigate('/app/qr')
      return
    }
    setComingSoonAction(action)
  }

  return (
    <>
      <div className="grid grid-cols-4 gap-2">
        {ACTIONS.map((action) => (
          <button
            key={action.key}
            type="button"
            onClick={() => handleActionClick(action)}
            className="flex flex-col items-center gap-2 rounded-xl bg-surface-raised p-3 text-xs font-medium text-fg-secondary shadow-sm transition-colors hover:bg-surface-sunken"
          >
            <span className="flex h-11 w-11 items-center justify-center rounded-full bg-secondary-50 text-secondary-600">
              <action.icon size={20} aria-hidden="true" />
            </span>
            {action.label}
          </button>
        ))}
      </div>

      <BottomSheet
        open={comingSoonAction !== null}
        onClose={() => setComingSoonAction(null)}
        title={comingSoonAction?.label}
      >
        <p className="text-sm text-fg-secondary">
          Esta funcion todavia no esta disponible. Estamos trabajando para traertela pronto.
        </p>
      </BottomSheet>
    </>
  )
}
