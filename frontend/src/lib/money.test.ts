import { describe, expect, it } from 'vitest'
import { formatMoney, parseAmountToMinor } from './money'

describe('formatMoney', () => {
  it('formatea USD con el locale por defecto (en-US)', () => {
    expect(formatMoney(128450, 'USD')).toBe('$1,284.50')
  })

  it('formatea CRC con el locale por defecto (es-CR)', () => {
    expect(formatMoney(645000, 'CRC')).toBe('₡6 450,00')
  })

  it('conserva el signo negativo', () => {
    expect(formatMoney(-1234, 'USD')).toBe('-$12.34')
  })

  it('permite forzar un locale distinto al por defecto de la moneda', () => {
    const result = formatMoney(100000, 'USD', 'es-CR')
    expect(result).toContain('1')
    expect(result).toContain('000')
    expect(result).toMatch(/US\$|USD|\$/)
  })

  it('redondea a cero decimales sobrantes al dividir entre 100', () => {
    expect(formatMoney(0, 'USD')).toBe('$0.00')
  })

  it('lanza un error si el monto no es un entero (nunca floats en unidades menores)', () => {
    expect(() => formatMoney(12.5, 'USD')).toThrow()
  })
})

describe('parseAmountToMinor', () => {
  it('convierte unidades mayores a menores enteras', () => {
    expect(parseAmountToMinor('12.34')).toBe(1234)
    expect(parseAmountToMinor('100')).toBe(10000)
    expect(parseAmountToMinor('0.05')).toBe(5)
  })

  it('acepta coma como separador decimal', () => {
    expect(parseAmountToMinor('12,34')).toBe(1234)
  })

  it('rechaza entradas invalidas o no positivas', () => {
    for (const bad of ['', '0', '0.00', '-5', 'abc', '1.234', '1.2.3']) {
      expect(parseAmountToMinor(bad)).toBeNull()
    }
  })
})
