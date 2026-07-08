import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it } from 'vitest'
import { useAuthStore } from '../../store/auth.store'
import { useWalletStore } from '../../store/wallet.store'
import { HomePage } from './HomePage'

const mockTransactions = [
  {
    id: 'tx_1',
    kind: 'in' as const,
    counterparty: 'Nomina Acme Corp',
    amountMinor: 85000,
    currency: 'USD' as const,
    createdAt: new Date().toISOString(),
    category: 'transfer' as const,
  },
  {
    id: 'tx_2',
    kind: 'out' as const,
    counterparty: 'Super La Colonia',
    amountMinor: 12300,
    currency: 'USD' as const,
    createdAt: new Date().toISOString(),
    category: 'payment' as const,
  },
]

describe('HomePage', () => {
  beforeEach(() => {
    useAuthStore.setState({ user: { id: 'usr_1', phoneMasked: '+506****1234', kycLevel: 0 } })
    useWalletStore.setState({
      wallets: [
        { currency: 'USD', balanceMinor: 128450 },
        { currency: 'CRC', balanceMinor: 645000 },
      ],
      transactions: mockTransactions,
      selectedCurrency: 'USD',
      status: 'idle',
      error: null,
    })
  })

  it('muestra el saldo de la moneda seleccionada y los movimientos recientes', () => {
    render(
      <MemoryRouter>
        <HomePage />
      </MemoryRouter>,
    )

    expect(screen.getByText('+506****1234')).toBeInTheDocument()
    expect(screen.getByText('$1,284.50')).toBeInTheDocument()
    expect(screen.getByText('Nomina Acme Corp')).toBeInTheDocument()
    expect(screen.getByText('Super La Colonia')).toBeInTheDocument()
  })

  it('permite cambiar entre monedas con el switch del balance', async () => {
    const user = userEvent.setup()
    render(
      <MemoryRouter>
        <HomePage />
      </MemoryRouter>,
    )

    await user.click(screen.getByRole('button', { name: 'CRC' }))
    expect(useWalletStore.getState().selectedCurrency).toBe('CRC')
  })
})
