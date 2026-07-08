import { act, fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, useLocation, useNavigate } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useUiStore } from '../store/ui.store'
import { useBackButton } from './useBackButton'

const exitAppMock = vi.fn()
let backButtonHandler: (() => void) | undefined

vi.mock('@capacitor/core', () => ({
  Capacitor: { isNativePlatform: () => true },
}))

vi.mock('@capacitor/app', () => ({
  App: {
    addListener: vi.fn((event: string, callback: () => void) => {
      if (event === 'backButton') backButtonHandler = callback
      return Promise.resolve({ remove: vi.fn() })
    }),
    exitApp: () => {
      exitAppMock()
      return Promise.resolve()
    },
  },
}))

async function flushMicrotasks(): Promise<void> {
  await act(async () => {
    await Promise.resolve()
    await Promise.resolve()
  })
}

function LocationDisplay() {
  const location = useLocation()
  return <div data-testid="loc">{location.pathname}</div>
}

function TestHarness() {
  useBackButton()
  const navigate = useNavigate()
  return (
    <>
      <button onClick={() => navigate('/app/qr')}>ir a qr</button>
      <LocationDisplay />
    </>
  )
}

describe('useBackButton', () => {
  beforeEach(() => {
    exitAppMock.mockClear()
    backButtonHandler = undefined
    useUiStore.setState({ overlays: [] })
  })

  it('cierra el overlay superior y no navega cuando hay uno abierto', async () => {
    useUiStore.getState().pushOverlay('sheet-1')

    render(
      <MemoryRouter initialEntries={['/app/home']}>
        <TestHarness />
      </MemoryRouter>,
    )
    await flushMicrotasks()

    expect(backButtonHandler).toBeDefined()
    act(() => backButtonHandler?.())

    expect(useUiStore.getState().overlays).toEqual([])
    expect(exitAppMock).not.toHaveBeenCalled()
    expect(screen.getByTestId('loc').textContent).toBe('/app/home')
  })

  it('navega hacia atras en el historial interno si no hay overlay pero si historial', async () => {
    render(
      <MemoryRouter initialEntries={['/app/home']}>
        <TestHarness />
      </MemoryRouter>,
    )
    await flushMicrotasks()

    act(() => fireEvent.click(screen.getByText('ir a qr')))
    expect(screen.getByTestId('loc').textContent).toBe('/app/qr')

    act(() => backButtonHandler?.())

    expect(screen.getByTestId('loc').textContent).toBe('/app/home')
    expect(exitAppMock).not.toHaveBeenCalled()
  })

  it('sale de la app cuando esta en la raiz, sin overlays ni historial interno', async () => {
    render(
      <MemoryRouter initialEntries={['/app/home']}>
        <TestHarness />
      </MemoryRouter>,
    )
    await flushMicrotasks()

    act(() => backButtonHandler?.())

    expect(exitAppMock).toHaveBeenCalledTimes(1)
  })
})
