import type { CurrencyCode, Wallet } from '../../api/types'
import { Card } from '../ui/Card'
import { MoneyText } from '../ui/MoneyText'

export interface BalanceCardProps {
  wallets: Wallet[]
  selectedCurrency: CurrencyCode
  onSelectCurrency: (currency: CurrencyCode) => void
}

export function BalanceCard({ wallets, selectedCurrency, onSelectCurrency }: BalanceCardProps) {
  const selectedWallet = wallets.find((wallet) => wallet.currency === selectedCurrency)

  return (
    <Card className="bg-brand-500 text-white" padded={false}>
      <div className="flex flex-col gap-4 p-5">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium text-white/80">Saldo disponible</span>
          <div className="flex rounded-full bg-white/15 p-1">
            {wallets.map((wallet) => (
              <button
                key={wallet.currency}
                type="button"
                onClick={() => onSelectCurrency(wallet.currency)}
                aria-pressed={wallet.currency === selectedCurrency}
                className={`rounded-full px-3 py-1 text-xs font-semibold transition-colors ${
                  wallet.currency === selectedCurrency ? 'bg-white text-brand-700' : 'text-white/80'
                }`}
              >
                {wallet.currency}
              </button>
            ))}
          </div>
        </div>
        <MoneyText
          amountMinor={selectedWallet?.balanceMinor ?? 0}
          currency={selectedCurrency}
          className="text-3xl font-bold text-white"
        />
      </div>
    </Card>
  )
}
