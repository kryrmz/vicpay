import { useEffect, useId, useRef } from 'react'
import type { ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { useUiStore } from '../../store/ui.store'

export interface BottomSheetProps {
  open: boolean
  onClose: () => void
  title?: string
  children: ReactNode
}

/**
 * Hoja inferior modal. Se registra en la pila de overlays de la app (ver
 * src/store/ui.store.ts) para que el boton atras de Android (ver
 * src/hooks/useBackButton.ts) la cierre en vez de navegar. Tambien se cierra
 * con Escape en web.
 */
export function BottomSheet({ open, onClose, title, children }: BottomSheetProps) {
  const overlayId = useId()
  const overlays = useUiStore((state) => state.overlays)
  const pushOverlay = useUiStore((state) => state.pushOverlay)
  const popOverlay = useUiStore((state) => state.popOverlay)
  /** True una vez que confirmamos que nuestro id quedo registrado en la pila. */
  const registeredRef = useRef(false)

  useEffect(() => {
    if (!open) {
      registeredRef.current = false
      return
    }
    pushOverlay(overlayId)
    return () => {
      registeredRef.current = false
      popOverlay(overlayId)
    }
  }, [open, overlayId, pushOverlay, popOverlay])

  useEffect(() => {
    if (!open) return
    if (overlays.includes(overlayId)) {
      registeredRef.current = true
      return
    }
    if (registeredRef.current) {
      // Estabamos registrados y alguien mas nos saco de la pila (p.ej. el
      // boton atras de Android): sincronizamos el estado "cerrado".
      onClose()
    }
  }, [overlays, open, overlayId, onClose])

  useEffect(() => {
    if (!open) return
    function handleKeyDown(event: KeyboardEvent): void {
      if (event.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [open, onClose])

  if (!open) return null

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-end justify-center">
      <button
        type="button"
        aria-label="Cerrar"
        className="absolute inset-0 h-full w-full bg-black/40"
        onClick={onClose}
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-label={title}
        className="relative z-10 w-full max-w-md rounded-t-xl border border-b-0 border-border bg-surface-raised p-5 shadow-lg"
      >
        <div className="mx-auto mb-4 h-1.5 w-10 rounded-full bg-surface-sunken" />
        {title ? <h2 className="mb-3 text-lg font-semibold text-fg">{title}</h2> : null}
        {children}
      </div>
    </div>,
    document.body,
  )
}
