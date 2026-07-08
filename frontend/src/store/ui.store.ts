import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type Theme = 'light' | 'dark'

interface UiState {
  /** Preferencia de tema explicita del usuario. No es PII: se puede persistir. */
  theme: Theme
  setTheme: (theme: Theme) => void
  toggleTheme: () => void
  /**
   * Pila de overlays abiertos (bottom sheets, dialogos). El boton atras de
   * Android/Capacitor cierra el overlay superior en vez de navegar cuando
   * esta pila no esta vacia. Ver src/hooks/useBackButton.ts.
   */
  overlays: string[]
  pushOverlay: (id: string) => void
  popOverlay: (id: string) => void
  /** Cierra el overlay superior. Devuelve true si habia uno para cerrar. */
  closeTopOverlay: () => boolean
}

export const useUiStore = create<UiState>()(
  persist(
    (set, get) => ({
      theme: 'light',

      setTheme(theme) {
        set({ theme })
      },

      toggleTheme() {
        set({ theme: get().theme === 'light' ? 'dark' : 'light' })
      },

      overlays: [],

      pushOverlay(id) {
        set({ overlays: [...get().overlays, id] })
      },

      popOverlay(id) {
        const { overlays } = get()
        if (!overlays.includes(id)) return
        set({ overlays: overlays.filter((overlayId) => overlayId !== id) })
      },

      closeTopOverlay() {
        const { overlays } = get()
        if (overlays.length === 0) return false
        set({ overlays: overlays.slice(0, -1) })
        return true
      },
    }),
    {
      name: 'vicpay-ui-theme',
      partialize: (state) => ({ theme: state.theme }),
    },
  ),
)
