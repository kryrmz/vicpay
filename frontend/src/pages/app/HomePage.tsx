import { useEffect } from 'react'
import { BalanceCard } from '../../components/home/BalanceCard'
import { QuickActions } from '../../components/home/QuickActions'
import { TransactionList } from '../../components/home/TransactionList'
import { useAuthStore } from '../../store/auth.store'
import { useWalletStore } from '../../store/wallet.store'

export function HomePage() {
  const user = useAuthStore((state) => state.user)
  const wallets = useWalletStore((state) => state.wallets)
  const transactions = useWalletStore((state) => state.transactions)
  const selectedCurrency = useWalletStore((state) => state.selectedCurrency)
  const selectCurrency = useWalletStore((state) => state.selectCurrency)
  const fetchAll = useWalletStore((state) => state.fetchAll)
  const status = useWalletStore((state) => state.status)

  useEffect(() => {
    void fetchAll()
  }, [fetchAll])

  return (
    <div className="flex flex-col gap-5 px-4 py-5">
      <header>
        <p className="text-sm text-fg-secondary">Hola,</p>
        <h1 className="text-xl font-bold text-fg">{user?.phoneMasked ?? 'Bienvenido'}</h1>
      </header>

      {status === 'loading' && wallets.length === 0 ? (
        <p className="py-10 text-center text-sm text-fg-muted">Cargando tu billetera...</p>
      ) : (
        <>
          <BalanceCard wallets={wallets} selectedCurrency={selectedCurrency} onSelectCurrency={selectCurrency} />
          <QuickActions />
          <TransactionList transactions={transactions} />
        </>
      )}
    </div>
  )
}
