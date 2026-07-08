import { Globe2, ShieldCheck, Wallet } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { Button } from '../components/ui/Button'

const HIGHLIGHTS = [
  { icon: Wallet, text: 'Una billetera, varias monedas: USD y CRC hoy, mas por venir.' },
  { icon: Globe2, text: 'Envia y recibe dinero donde estes, sin fronteras.' },
  { icon: ShieldCheck, text: 'Tu sesion vive solo en tu dispositivo: nunca guardamos datos sensibles de mas.' },
]

export function WelcomePage() {
  const navigate = useNavigate()

  return (
    <div className="flex min-h-dvh flex-col justify-between bg-surface px-6 py-10">
      <header className="flex flex-col items-center gap-2 pt-8 text-center">
        <h1 className="text-3xl font-extrabold tracking-tight text-fg">
          Vic<span className="text-brand-500">Pay</span>
        </h1>
        <p className="max-w-xs text-balance text-fg-secondary">
          Tu dinero, sin fronteras. Billetera multimoneda, pagos con QR y ahorros en un solo lugar.
        </p>
      </header>

      <ul className="mx-auto flex w-full max-w-sm flex-col gap-4 py-10">
        {HIGHLIGHTS.map(({ icon: Icon, text }) => (
          <li key={text} className="flex items-start gap-3 rounded-xl bg-surface-raised p-4 shadow-sm">
            <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-brand-50 text-brand-600">
              <Icon size={18} aria-hidden="true" />
            </span>
            <p className="text-sm text-fg-secondary">{text}</p>
          </li>
        ))}
      </ul>

      <div className="mx-auto flex w-full max-w-sm flex-col gap-3">
        <Button variant="primary" className="w-full" onClick={() => navigate('/register')}>
          Crear cuenta
        </Button>
        <Button variant="secondary" className="w-full" onClick={() => navigate('/login')}>
          Iniciar sesion
        </Button>
      </div>
    </div>
  )
}
