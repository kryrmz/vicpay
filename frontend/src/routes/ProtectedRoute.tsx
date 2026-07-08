import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '../store/auth.store'

export interface ProtectedRouteProps {
  children: ReactNode
}

/** Si no hay sesion en memoria, redirige a /welcome preservando el destino original. */
export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const accessToken = useAuthStore((state) => state.accessToken)
  const location = useLocation()

  if (!accessToken) {
    return <Navigate to="/welcome" replace state={{ from: location }} />
  }

  return <>{children}</>
}
