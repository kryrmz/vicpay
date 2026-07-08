import { formatMoney } from '../../lib/money'

export type MoneyTone = 'neutral' | 'in' | 'out'

export interface MoneyTextProps {
  amountMinor: number
  currency: string
  locale?: string
  tone?: MoneyTone
  className?: string
}

// 'neutral' y 'out' no imponen color: heredan currentColor o el className del
// que llama. Solo 'in' fuerza un color semantico (verde de exito), ya que ese
// caso nunca se combina con un className que pinte texto de otro color.
const TONE_CLASSES: Record<MoneyTone, string> = {
  neutral: '',
  in: 'text-success',
  out: '',
}

export function MoneyText({ amountMinor, currency, locale, tone = 'neutral', className = '' }: MoneyTextProps) {
  const formatted = formatMoney(amountMinor, currency, locale)
  const sign = tone === 'in' && amountMinor > 0 ? '+' : ''

  return (
    <span className={`${TONE_CLASSES[tone]} ${className}`}>
      {sign}
      {formatted}
    </span>
  )
}
