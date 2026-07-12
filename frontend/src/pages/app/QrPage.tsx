import { CheckCircle2, QrCode, ScanLine } from 'lucide-react'
import { QRCodeSVG } from 'qrcode.react'
import { useState } from 'react'
import { api } from '../../api'
import { ApiError } from '../../api/types'
import { Button } from '../../components/ui/Button'
import { Card } from '../../components/ui/Card'
import { Input } from '../../components/ui/Input'
import { formatMoney, parseAmountToMinor } from '../../lib/money'
import { decodePaymentRequest, encodePaymentRequest } from '../../lib/paymentRequest'
import type { PaymentRequest } from '../../lib/paymentRequest'
import { useAuthStore } from '../../store/auth.store'
import { useWalletStore } from '../../store/wallet.store'

type Mode = 'receive' | 'pay'

export function QrPage() {
  const [mode, setMode] = useState<Mode>('receive')

  return (
    <div className="flex flex-col gap-6 px-4 py-6">
      <header className="text-center">
        <h1 className="text-xl font-bold text-fg">Pagos con QR</h1>
        <p className="mt-1 text-sm text-fg-secondary">Cobra generando un codigo, o paga leyendo uno.</p>
      </header>

      <div className="mx-auto flex w-full max-w-xs rounded-lg bg-surface-sunken p-1 text-sm font-medium">
        <button
          type="button"
          onClick={() => setMode('receive')}
          className={`flex-1 rounded-md py-2 transition-colors ${
            mode === 'receive' ? 'bg-surface-raised text-fg shadow-sm' : 'text-fg-secondary'
          }`}
        >
          Cobrar
        </button>
        <button
          type="button"
          onClick={() => setMode('pay')}
          className={`flex-1 rounded-md py-2 transition-colors ${
            mode === 'pay' ? 'bg-surface-raised text-fg shadow-sm' : 'text-fg-secondary'
          }`}
        >
          Pagar
        </button>
      </div>

      {mode === 'receive' ? <ReceivePanel /> : <PayPanel />}
    </div>
  )
}

function ReceivePanel() {
  const user = useAuthStore((s) => s.user)
  const currency = useWalletStore((s) => s.selectedCurrency)
  const [amount, setAmount] = useState('')

  const amountMinor = parseAmountToMinor(amount)
  const qrValue =
    amountMinor !== null && user ? encodePaymentRequest({ toUserId: user.id, amountMinor, currency }) : null

  return (
    <div className="flex flex-col items-center gap-4">
      <div className="w-full max-w-xs">
        <Input
          label={`Monto a cobrar (${currency})`}
          type="text"
          inputMode="decimal"
          placeholder="0.00"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
        />
      </div>

      <Card className="flex flex-col items-center gap-4">
        {qrValue ? (
          <>
            <div className="rounded-xl bg-white p-4">
              <QRCodeSVG value={qrValue} size={200} />
            </div>
            <p className="text-lg font-semibold text-fg">{formatMoney(amountMinor ?? 0, currency)}</p>
            <p className="text-sm text-fg-secondary">{user?.phoneMasked}</p>
            <div className="w-full">
              <p className="mb-1 text-xs text-fg-muted">O comparte este codigo si no pueden escanear:</p>
              <code
                data-testid="pay-code"
                className="block w-full select-all break-all rounded-md bg-surface-sunken px-2 py-1.5 text-xs text-fg-secondary"
              >
                {qrValue}
              </code>
            </div>
          </>
        ) : (
          <div className="flex h-[248px] w-[232px] flex-col items-center justify-center gap-2 text-center text-fg-muted">
            <QrCode size={40} aria-hidden="true" />
            <p className="text-sm">Ingresa un monto para generar tu QR de cobro.</p>
          </div>
        )}
      </Card>
    </div>
  )
}

function PayPanel() {
  const fetchAll = useWalletStore((s) => s.fetchAll)
  const [code, setCode] = useState('')
  const [request, setRequest] = useState<PaymentRequest | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [paying, setPaying] = useState(false)
  const [done, setDone] = useState(false)

  function handleDecode(): void {
    const parsed = decodePaymentRequest(code)
    if (!parsed) {
      setError('El codigo de pago no es valido. Copia el codigo completo (empieza con vicpay:pay).')
      setRequest(null)
      return
    }
    setError(null)
    setRequest(parsed)
  }

  async function handlePay(): Promise<void> {
    if (!request) {
      return
    }
    setPaying(true)
    setError(null)
    try {
      await api.payRequest(request.toUserId, request.amountMinor, request.currency, crypto.randomUUID())
      await fetchAll()
      setDone(true)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'No se pudo completar el pago.')
    } finally {
      setPaying(false)
    }
  }

  function reset(): void {
    setCode('')
    setRequest(null)
    setError(null)
    setDone(false)
  }

  if (done && request) {
    return (
      <Card className="flex flex-col items-center gap-3 text-center">
        <CheckCircle2 className="text-success" size={40} aria-hidden="true" />
        <p className="text-lg font-semibold text-fg">Pago enviado</p>
        <p className="text-sm text-fg-secondary">
          Pagaste {formatMoney(request.amountMinor, request.currency)}.
        </p>
        <Button variant="secondary" onClick={reset} className="mt-2">
          Pagar otro
        </Button>
      </Card>
    )
  }

  return (
    <div className="flex flex-col gap-4">
      <label className="flex flex-col gap-1.5">
        <span className="text-sm font-medium text-fg-secondary">Codigo de pago</span>
        <textarea
          value={code}
          onChange={(e) => setCode(e.target.value)}
          rows={3}
          placeholder="vicpay:pay?to=...&amt=...&cur=USD"
          className="rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-fg outline-none transition-colors placeholder:text-fg-muted focus:border-brand-500 focus:ring-2 focus:ring-brand-100"
        />
      </label>

      {request ? (
        <Card className="flex flex-col gap-2">
          <p className="text-sm text-fg-secondary">Vas a pagar</p>
          <p className="text-2xl font-bold text-fg">{formatMoney(request.amountMinor, request.currency)}</p>
          <Button onClick={handlePay} loading={paying} className="mt-2">
            Confirmar pago
          </Button>
          <Button variant="ghost" onClick={() => setRequest(null)} disabled={paying}>
            Usar otro codigo
          </Button>
        </Card>
      ) : (
        <Button variant="secondary" onClick={handleDecode} leftIcon={<ScanLine size={18} aria-hidden="true" />}>
          Leer codigo
        </Button>
      )}

      {error ? <p className="text-sm text-danger">{error}</p> : null}

      <p className="text-xs text-fg-muted">
        El escaneo con camara requiere la app nativa (Capacitor). Por ahora, pega el codigo del cobro.
      </p>
    </div>
  )
}
