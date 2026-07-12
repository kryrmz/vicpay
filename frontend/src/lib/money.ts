/**
 * Unica utilidad autorizada para convertir dinero (unidades menores enteras)
 * a texto legible. El dinero nunca se representa como float en VicPay: todo
 * monto vive en unidades menores (centimos) como entero.
 */

const DEFAULT_LOCALE_BY_CURRENCY: Record<string, string> = {
  USD: 'en-US',
  CRC: 'es-CR',
}

/**
 * Formatea un monto en unidades menores como texto de moneda localizado.
 *
 * @param amountMinor monto entero en unidades menores (p.ej. centimos). Puede ser negativo.
 * @param currency codigo ISO 4217 de la moneda (p.ej. 'USD', 'CRC').
 * @param locale locale BCP 47 opcional. Por defecto: en-US para USD, es-CR para CRC.
 */
export function formatMoney(amountMinor: number, currency: string, locale?: string): string {
  if (!Number.isInteger(amountMinor)) {
    throw new Error('formatMoney espera un entero en unidades menores, nunca un float.')
  }

  const resolvedLocale = locale ?? DEFAULT_LOCALE_BY_CURRENCY[currency]
  const amountMajor = amountMinor / 100

  return new Intl.NumberFormat(resolvedLocale, {
    style: 'currency',
    currency,
    currencyDisplay: 'symbol',
  }).format(amountMajor)
}

/**
 * Convierte un monto escrito por el usuario en unidades mayores (p.ej. "12.34")
 * a unidades menores enteras (1234). Acepta coma o punto como separador decimal
 * y hasta dos decimales. Devuelve null si la entrada no es un monto valido > 0.
 */
export function parseAmountToMinor(input: string): number | null {
  const normalized = input.trim().replace(',', '.')
  if (!/^\d+(\.\d{1,2})?$/.test(normalized)) {
    return null
  }
  const minor = Math.round(Number(normalized) * 100)
  return minor > 0 ? minor : null
}
