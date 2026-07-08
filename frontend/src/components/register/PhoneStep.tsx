import { useState } from 'react'
import type { FormEvent } from 'react'
import { Button } from '../ui/Button'
import { Input } from '../ui/Input'

export interface PhoneStepValues {
  phone: string
  password: string
}

export interface PhoneStepProps {
  onSubmit: (values: PhoneStepValues) => Promise<void> | void
  submitting: boolean
  error: string | null
}

const E164_PATTERN = /^\+[1-9]\d{7,14}$/

export function PhoneStep({ onSubmit, submitting, error }: PhoneStepProps) {
  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')
  const [touched, setTouched] = useState(false)

  const phoneError = touched && !E164_PATTERN.test(phone) ? 'Usa formato internacional, ej. +50688881234.' : undefined
  const passwordError = touched && password.length < 8 ? 'Minimo 8 caracteres.' : undefined

  function handleSubmit(event: FormEvent): void {
    event.preventDefault()
    setTouched(true)
    if (!E164_PATTERN.test(phone) || password.length < 8) return
    void onSubmit({ phone, password })
  }

  return (
    <form className="flex flex-col gap-4" onSubmit={handleSubmit} noValidate>
      <Input
        label="Numero de telefono"
        type="tel"
        placeholder="+50688881234"
        autoComplete="tel"
        value={phone}
        onChange={(event) => setPhone(event.target.value.trim())}
        error={phoneError}
        hint={!phoneError ? 'Formato internacional E.164, con el signo +.' : undefined}
      />
      <Input
        label="Contrasena"
        type="password"
        autoComplete="new-password"
        value={password}
        onChange={(event) => setPassword(event.target.value)}
        error={passwordError}
        hint={!passwordError ? 'Minimo 8 caracteres.' : undefined}
      />
      {error ? <p className="text-sm text-danger">{error}</p> : null}
      <Button type="submit" variant="primary" className="w-full" loading={submitting}>
        Continuar
      </Button>
    </form>
  )
}
