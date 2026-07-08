import { beforeEach, describe, expect, it } from 'vitest'
import { useAuthStore } from './auth.store'

function resetAuthStore(): void {
  useAuthStore.setState({
    user: null,
    accessToken: null,
    pendingUserId: null,
    status: 'idle',
    error: null,
  })
}

describe('useAuthStore', () => {
  beforeEach(() => {
    resetAuthStore()
  })

  it('register() guarda un pendingUserId sin crear sesion', async () => {
    await useAuthStore.getState().register('+50670001111', 'contrasenaSegura')

    const state = useAuthStore.getState()
    expect(state.pendingUserId).toBeTruthy()
    expect(state.accessToken).toBeNull()
    expect(state.user).toBeNull()
  })

  it('el wizard no avanza si verifyPhone() recibe un codigo incorrecto', async () => {
    await useAuthStore.getState().register('+50670001112', 'contrasenaSegura')
    await expect(useAuthStore.getState().verifyPhone('111111')).rejects.toThrow()

    const state = useAuthStore.getState()
    expect(state.accessToken).toBeNull()
    expect(state.user).toBeNull()
    expect(state.error).toBeTruthy()
  })

  it('verifyPhone() con el codigo correcto crea la sesion solo en memoria', async () => {
    await useAuthStore.getState().register('+50670001113', 'contrasenaSegura')
    await useAuthStore.getState().verifyPhone('000000')

    const state = useAuthStore.getState()
    expect(state.accessToken).toBeTruthy()
    expect(state.user).not.toBeNull()
    expect(state.pendingUserId).toBeNull()
  })

  it('logout() limpia el usuario y el access token en memoria', async () => {
    await useAuthStore.getState().register('+50670001114', 'contrasenaSegura')
    await useAuthStore.getState().verifyPhone('000000')
    await useAuthStore.getState().logout()

    const state = useAuthStore.getState()
    expect(state.user).toBeNull()
    expect(state.accessToken).toBeNull()
  })
})
