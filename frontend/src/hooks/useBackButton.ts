import { useEffect, useRef } from 'react'
import { useNavigate, useNavigationType, useLocation } from 'react-router-dom'
import { useUiStore } from '../store/ui.store'

/**
 * Cablea el boton fisico/gesto "atras" de Android (via @capacitor/app) con:
 *  1. La pila de overlays de la app (BottomSheet, dialogos): si hay uno
 *     abierto, atras lo cierra y NO navega.
 *  2. El historial del router: si no hay overlay pero hay historial dentro de
 *     la app (llegamos aqui navegando, no es la pantalla raiz), atras navega
 *     hacia atras en ese historial.
 *  3. La raiz de la app: si no hay overlay ni historial interno, atras sale
 *     de la app (App.exitApp()).
 *
 * En web (no nativo) esto es un no-op: no se registra ningun listener.
 */
export function useBackButton(): void {
  const navigate = useNavigate()
  const location = useLocation()
  const navigationType = useNavigationType()
  const depthRef = useRef(0)

  useEffect(() => {
    if (navigationType === 'PUSH') {
      depthRef.current += 1
    } else if (navigationType === 'POP') {
      depthRef.current = Math.max(0, depthRef.current - 1)
    }
    // REPLACE no cambia la profundidad de historial interno.
  }, [location, navigationType])

  useEffect(() => {
    let removeListener: (() => void) | undefined
    let cancelled = false

    async function register(): Promise<void> {
      const { Capacitor } = await import('@capacitor/core')
      if (!Capacitor.isNativePlatform()) return

      const { App } = await import('@capacitor/app')
      const handle = await App.addListener('backButton', () => {
        const closedOverlay = useUiStore.getState().closeTopOverlay()
        if (closedOverlay) return

        if (depthRef.current > 0) {
          navigate(-1)
          return
        }

        void App.exitApp()
      })

      if (cancelled) {
        handle.remove()
      } else {
        removeListener = () => handle.remove()
      }
    }

    void register()

    return () => {
      cancelled = true
      removeListener?.()
    }
  }, [navigate])
}
