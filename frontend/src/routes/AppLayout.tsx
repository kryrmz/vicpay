import { Outlet } from 'react-router-dom'
import { TabBar } from '../components/ui/TabBar'

/** Shell de las pantallas autenticadas: contenido con scroll + tab bar fija abajo. */
export function AppLayout() {
  return (
    <div className="flex h-dvh flex-col bg-surface">
      <main className="flex-1 overflow-y-auto">
        <Outlet />
      </main>
      <TabBar />
    </div>
  )
}
