import { PiggyBank, QrCode, Send, Wallet } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../../api'
import { ApiError } from '../../api/types'
import type { CurrencyCode } from '../../api/types'
import { formatMoney } from '../../lib/money'
import { useWalletStore } from '../../store/wallet.store'
import { BottomSheet } from '../ui/BottomSheet'
import { SendMoneySheet } from './SendMoneySheet'

interface QuickAction {
  key: string
  label: string
  icon: LucideIcon
  comingSoon?: boolean
}

const ACTIONS: QuickAction[] = [
  { key: 'send', label: 'Enviar', icon: Send },
  { key: 'collect', label: 'Cobrar', icon: QrCode },
  { key: 'topup', label: 'Recargar', icon: Wallet },
  { key: 'savings', label: 'Ahorros', icon: PiggyBank, comingSoon: true },
]

// Montos de la recarga de demostracion, por moneda (en unidades menores).
const DEMO_TOPUP_MINOR: Record<CurrencyCode, number> = {
  USD: 10000,
  CRC: 5000000,
}

interface Message {
  title: string
  body: string
}

export function QuickActions() {
  const navigate = useNavigate()
  const currency = useWalletStore((s) => s.selectedCurrency)
  const fetchAll = useWalletStore((s) => s.fetchAll)

  const [sendOpen, setSendOpen] = useState(false)
  const [message, setMessage] = useState<Message | null>(null)
  const [topUpBusy, setTopUpBusy] = useState(false)

  function handleActionClick(action: QuickAction): void {
    switch (action.key) {
      case 'send':
        setSendOpen(true)
        return
      case 'collect':
        navigate('/app/qr')
        return
      case 'topup':
        void handleTopUp()
        return
      default:
        setMessage({
          title: action.label,
          body: 'Esta funcion todavia no esta disponible. Estamos trabajando para traertela pronto.',
        })
    }
  }

  async function handleTopUp(): Promise<void> {
    if (topUpBusy) {
      return
    }
    setTopUpBusy(true)
    const amountMinor = DEMO_TOPUP_MINOR[currency]
    try {
      await api.topUp(amountMinor, currency, crypto.randomUUID())
      await fetchAll()
      setMessage({
        title: 'Recarga demo acreditada',
        body: `Se acreditaron ${formatMoney(amountMinor, currency)} a tu billetera (fondos de demostracion).`,
      })
    } catch (err) {
      setMessage({
        title: 'No se pudo recargar',
        body: err instanceof ApiError ? err.message : 'Intenta de nuevo mas tarde.',
      })
    } finally {
      setTopUpBusy(false)
    }
  }

  return (
    <>
      <div className="grid grid-cols-4 gap-2">
        {ACTIONS.map((action) => (
          <button
            key={action.key}
            type="button"
            onClick={() => handleActionClick(action)}
            disabled={action.key === 'topup' && topUpBusy}
            className="flex flex-col items-center gap-2 rounded-xl bg-surface-raised p-3 text-xs font-medium text-fg-secondary shadow-sm transition-colors hover:bg-surface-sunken disabled:opacity-60"
          >
            <span className="flex h-11 w-11 items-center justify-center rounded-full bg-secondary-50 text-secondary-600">
              <action.icon size={20} aria-hidden="true" />
            </span>
            {action.label}
          </button>
        ))}
      </div>

      <SendMoneySheet open={sendOpen} onClose={() => setSendOpen(false)} />

      <BottomSheet open={message !== null} onClose={() => setMessage(null)} title={message?.title}>
        <p className="text-sm text-fg-secondary">{message?.body}</p>
      </BottomSheet>
    </>
  )
}
