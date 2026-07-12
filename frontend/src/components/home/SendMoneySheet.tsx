import { Send } from 'lucide-react'
import { useState } from 'react'
import { api } from '../../api'
import { ApiError } from '../../api/types'
import { formatMoney, parseAmountToMinor } from '../../lib/money'
import { useWalletStore } from '../../store/wallet.store'
import { Button } from '../ui/Button'
import { Input } from '../ui/Input'
import { BottomSheet } from '../ui/BottomSheet'

interface SendMoneySheetProps {
  open: boolean
  onClose: () => void
}

const E164_PATTERN = /^\+[1-9]\d{7,14}$/

export function SendMoneySheet({ open, onClose }: SendMoneySheetProps) {
  const currency = useWalletStore((s) => s.selectedCurrency)
  const wallets = useWalletStore((s) => s.wallets)
  const fetchAll = useWalletStore((s) => s.fetchAll)

  const [phone, setPhone] = useState('')
  const [amount, setAmount] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const balanceMinor = wallets.find((w) => w.currency === currency)?.balanceMinor ?? 0

  function reset(): void {
    setPhone('')
    setAmount('')
    setError(null)
    setSubmitting(false)
  }

  function handleClose(): void {
    if (submitting) {
      return
    }
    reset()
    onClose()
  }

  async function handleSubmit(): Promise<void> {
    setError(null)
    if (!E164_PATTERN.test(phone.trim())) {
      setError('Ingresa el telefono del destinatario en formato internacional, p.ej. +50688881234.')
      return
    }
    const amountMinor = parseAmountToMinor(amount)
    if (amountMinor === null) {
      setError('Ingresa un monto valido mayor que cero.')
      return
    }
    if (amountMinor > balanceMinor) {
      setError('El monto supera tu saldo disponible.')
      return
    }
    setSubmitting(true)
    try {
      await api.transfer(phone.trim(), amountMinor, currency, crypto.randomUUID())
      await fetchAll()
      reset()
      onClose()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'No se pudo completar el envio.')
      setSubmitting(false)
    }
  }

  return (
    <BottomSheet open={open} onClose={handleClose} title="Enviar dinero">
      <div className="flex flex-col gap-4">
        <p className="text-sm text-fg-secondary">
          Disponible: <span className="font-semibold text-fg">{formatMoney(balanceMinor, currency)}</span>
        </p>
        <Input
          label="Telefono del destinatario"
          type="tel"
          inputMode="tel"
          placeholder="+50688881234"
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
          autoComplete="off"
        />
        <Input
          label={`Monto (${currency})`}
          type="text"
          inputMode="decimal"
          placeholder="0.00"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          error={error ?? undefined}
        />
        <Button onClick={handleSubmit} loading={submitting} leftIcon={<Send size={18} aria-hidden="true" />}>
          Enviar
        </Button>
      </div>
    </BottomSheet>
  )
}
