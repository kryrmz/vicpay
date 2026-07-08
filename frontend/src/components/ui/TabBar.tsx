import { Activity, Home, QrCode, User } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { NavLink } from 'react-router-dom'

interface TabDefinition {
  to: string
  label: string
  icon: LucideIcon
}

const TABS: TabDefinition[] = [
  { to: '/app/home', label: 'Inicio', icon: Home },
  { to: '/app/qr', label: 'QR', icon: QrCode },
  { to: '/app/activity', label: 'Actividad', icon: Activity },
  { to: '/app/profile', label: 'Perfil', icon: User },
]

export function TabBar() {
  return (
    <nav
      aria-label="Navegacion principal"
      className="sticky bottom-0 flex border-t border-border bg-surface-raised pb-[env(safe-area-inset-bottom)]"
    >
      {TABS.map(({ to, label, icon: Icon }) => (
        <NavLink
          key={to}
          to={to}
          className={({ isActive }) =>
            `flex flex-1 flex-col items-center gap-1 py-2.5 text-xs font-medium transition-colors ${
              isActive ? 'text-brand-600' : 'text-fg-muted hover:text-fg-secondary'
            }`
          }
        >
          <Icon size={22} strokeWidth={2} aria-hidden="true" />
          {label}
        </NavLink>
      ))}
    </nav>
  )
}
