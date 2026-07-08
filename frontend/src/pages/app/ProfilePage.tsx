import { LogOut, Moon, ShieldCheck, Sun } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { Card } from '../../components/ui/Card'
import { useAuthStore } from '../../store/auth.store'
import { useUiStore } from '../../store/ui.store'

const KYC_EXPLANATIONS: Record<number, string> = {
  0: 'Nivel basico: enviar, recibir y pagar con QR sin fricciones, con limites moderados.',
  1: 'Nivel intermedio: limites mas altos al verificar tu identidad.',
  2: 'Nivel avanzado: limites maximos y acceso a todos los productos de VicPay.',
}

export function ProfilePage() {
  const navigate = useNavigate()
  const user = useAuthStore((state) => state.user)
  const logout = useAuthStore((state) => state.logout)
  const theme = useUiStore((state) => state.theme)
  const toggleTheme = useUiStore((state) => state.toggleTheme)

  async function handleLogout(): Promise<void> {
    await logout()
    navigate('/welcome', { replace: true })
  }

  return (
    <div className="flex flex-col gap-5 px-4 py-5">
      <header>
        <h1 className="text-xl font-bold text-fg">Perfil</h1>
      </header>

      <Card className="flex flex-col gap-3">
        <div>
          <p className="text-xs text-fg-muted">Telefono</p>
          <p className="text-base font-semibold text-fg">{user?.phoneMasked}</p>
        </div>
        <div>
          <div className="flex items-center gap-2">
            <Badge variant="secondary">
              <ShieldCheck size={14} aria-hidden="true" />
              Nivel KYC {user?.kycLevel ?? 0}
            </Badge>
          </div>
          <p className="mt-2 text-sm text-fg-secondary">{KYC_EXPLANATIONS[user?.kycLevel ?? 0]}</p>
        </div>
      </Card>

      <Card className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-fg">Tema</p>
          <p className="text-xs text-fg-muted">Claro u oscuro, a tu gusto.</p>
        </div>
        <Button variant="ghost" onClick={toggleTheme} aria-label="Cambiar tema">
          {theme === 'dark' ? <Sun size={20} aria-hidden="true" /> : <Moon size={20} aria-hidden="true" />}
        </Button>
      </Card>

      <Button variant="secondary" className="w-full" onClick={() => void handleLogout()}>
        <LogOut size={18} aria-hidden="true" />
        Cerrar sesion
      </Button>
    </div>
  )
}
