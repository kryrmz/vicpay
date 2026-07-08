import { useRef } from 'react'
import type { ClipboardEvent, KeyboardEvent } from 'react'

export interface OtpInputProps {
  length?: number
  value: string
  onChange: (value: string) => void
  disabled?: boolean
  error?: string
}

const DIGIT_PATTERN = /^\d$/

export function OtpInput({ length = 6, value, onChange, disabled = false, error }: OtpInputProps) {
  const inputRefs = useRef<Array<HTMLInputElement | null>>([])
  const digits = Array.from({ length }, (_, index) => value[index] ?? '')

  function setDigitAt(index: number, digit: string): void {
    const nextDigits = [...digits]
    nextDigits[index] = digit
    onChange(nextDigits.join(''))
  }

  function focusInput(index: number): void {
    inputRefs.current[index]?.focus()
  }

  function handleChange(index: number, rawValue: string): void {
    const incoming = rawValue.slice(-1)
    if (incoming && !DIGIT_PATTERN.test(incoming)) return

    setDigitAt(index, incoming)
    if (incoming && index < length - 1) {
      focusInput(index + 1)
    }
  }

  function handleKeyDown(index: number, event: KeyboardEvent<HTMLInputElement>): void {
    if (event.key === 'Backspace' && !digits[index] && index > 0) {
      focusInput(index - 1)
      setDigitAt(index - 1, '')
    } else if (event.key === 'ArrowLeft' && index > 0) {
      focusInput(index - 1)
    } else if (event.key === 'ArrowRight' && index < length - 1) {
      focusInput(index + 1)
    }
  }

  function handlePaste(event: ClipboardEvent<HTMLInputElement>): void {
    const pasted = event.clipboardData.getData('text').replace(/\D/g, '').slice(0, length)
    if (!pasted) return
    event.preventDefault()
    onChange(pasted.padEnd(length, '').slice(0, length))
    focusInput(Math.min(pasted.length, length - 1))
  }

  return (
    <div className="flex justify-between gap-2" role="group" aria-label="Codigo de verificacion">
      {digits.map((digit, index) => (
        <input
          key={index}
          ref={(element) => {
            inputRefs.current[index] = element
          }}
          value={digit}
          onChange={(event) => handleChange(index, event.target.value)}
          onKeyDown={(event) => handleKeyDown(index, event)}
          onPaste={handlePaste}
          disabled={disabled}
          inputMode="numeric"
          autoComplete="one-time-code"
          aria-label={`Digito ${index + 1}`}
          aria-invalid={Boolean(error)}
          maxLength={1}
          className={`h-12 w-11 rounded-lg border text-center text-lg font-semibold text-fg outline-none transition-colors focus:border-brand-500 focus:ring-2 focus:ring-brand-100 ${
            error ? 'border-danger' : 'border-border'
          }`}
        />
      ))}
    </div>
  )
}
