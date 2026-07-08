import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it } from 'vitest'
import { useAuthStore } from '../store/auth.store'
import { ProtectedRoute } from './ProtectedRoute'

function renderProtected(initialPath: string) {
  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <Routes>
        <Route path="/welcome" element={<div>pantalla de bienvenida</div>} />
        <Route
          path="/app/home"
          element={
            <ProtectedRoute>
              <div>panel privado</div>
            </ProtectedRoute>
          }
        />
      </Routes>
    </MemoryRouter>,
  )
}

describe('ProtectedRoute', () => {
  afterEach(() => {
    useAuthStore.setState({ user: null, accessToken: null, pendingUserId: null })
  })

  it('redirige a /welcome cuando no hay sesion en memoria', () => {
    renderProtected('/app/home')

    expect(screen.getByText('pantalla de bienvenida')).toBeInTheDocument()
    expect(screen.queryByText('panel privado')).not.toBeInTheDocument()
  })

  it('renderiza los hijos cuando hay una sesion activa', () => {
    useAuthStore.setState({
      user: { id: 'usr_1', phoneMasked: '+5****34', kycLevel: 0 },
      accessToken: 'tok_abc',
    })

    renderProtected('/app/home')

    expect(screen.getByText('panel privado')).toBeInTheDocument()
  })
})
