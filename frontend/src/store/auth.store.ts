import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { api } from '../api'
import type { User } from '../api/types'

function toMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'Ocurrio un error inesperado.'
}

interface AuthState {
  /** Usuario autenticado. Solo en memoria. */
  user: User | null
  /** Access token de sesion. Solo en memoria: NUNCA se persiste (cero PII en localStorage). */
  accessToken: string | null
  /** Registro de telefono en curso, en espera del codigo OTP. */
  pendingUserId: string | null
  status: 'idle' | 'loading'
  error: string | null
  /**
   * Unico dato persistido de este store: un booleano sin PII que solo indica
   * si el dispositivo ya tuvo una sesion antes (para, por ejemplo, ajustar
   * copy de bienvenida). Nunca contiene el usuario, el telefono ni el token.
   */
  hasSignedInBefore: boolean
  register: (phone: string, password: string) => Promise<void>
  verifyPhone: (code: string) => Promise<void>
  login: (phone: string, password: string) => Promise<void>
  logout: () => Promise<void>
  clearError: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      accessToken: null,
      pendingUserId: null,
      status: 'idle',
      error: null,
      hasSignedInBefore: false,

      async register(phone, password) {
        set({ status: 'loading', error: null })
        try {
          const { pendingUserId } = await api.register(phone, password)
          set({ pendingUserId, status: 'idle' })
        } catch (error) {
          set({ status: 'idle', error: toMessage(error) })
          throw error
        }
      },

      async verifyPhone(code) {
        const { pendingUserId } = get()
        if (!pendingUserId) {
          const message = 'No hay un registro pendiente de verificacion.'
          set({ error: message })
          throw new Error(message)
        }
        set({ status: 'loading', error: null })
        try {
          const session = await api.verifyPhone(pendingUserId, code)
          set({
            user: session.user,
            accessToken: session.accessToken,
            pendingUserId: null,
            status: 'idle',
            hasSignedInBefore: true,
          })
        } catch (error) {
          set({ status: 'idle', error: toMessage(error) })
          throw error
        }
      },

      async login(phone, password) {
        set({ status: 'loading', error: null })
        try {
          const session = await api.login(phone, password)
          set({
            user: session.user,
            accessToken: session.accessToken,
            status: 'idle',
            hasSignedInBefore: true,
          })
        } catch (error) {
          set({ status: 'idle', error: toMessage(error) })
          throw error
        }
      },

      async logout() {
        await api.logout()
        set({ user: null, accessToken: null, pendingUserId: null })
      },

      clearError() {
        set({ error: null })
      },
    }),
    {
      name: 'vicpay-auth-flags',
      partialize: (state) => ({ hasSignedInBefore: state.hasSignedInBefore }),
    },
  ),
)
