import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { useAuthStore } from '../store/auth.store'

export function LoginPage() {
  const navigate = useNavigate()
  const login = useAuthStore((state) => state.login)
  const status = useAuthStore((state) => state.status)
  const error = useAuthStore((state) => state.error)

  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')

  async function handleSubmit(event: FormEvent): Promise<void> {
    event.preventDefault()
    try {
      await login(phone, password)
      navigate('/app/home', { replace: true })
    } catch {
      // El error ya queda expuesto via el store.
    }
  }

  return (
    <div className="mx-auto flex min-h-dvh max-w-sm flex-col justify-center gap-6 px-6 py-10">
      <header className="text-center">
        <h1 className="text-2xl font-bold text-fg">
          Bienvenido de nuevo a Vic<span className="text-brand-500">Pay</span>
        </h1>
        <p className="mt-1 text-sm text-fg-secondary">Inicia sesion para continuar.</p>
      </header>

      <form className="flex flex-col gap-4" onSubmit={handleSubmit} noValidate>
        <Input
          label="Numero de telefono"
          type="tel"
          placeholder="+50688881234"
          autoComplete="tel"
          value={phone}
          onChange={(event) => setPhone(event.target.value.trim())}
        />
        <Input
          label="Contrasena"
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
        />
        {error ? <p className="text-sm text-danger">{error}</p> : null}
        <Button type="submit" variant="primary" className="w-full" loading={status === 'loading'}>
          Iniciar sesion
        </Button>
      </form>

      <p className="text-center text-xs text-fg-muted">
        Modo de prueba: +50688888888 / VicPay#2026
      </p>

      <p className="text-center text-sm text-fg-secondary">
        No tienes cuenta?{' '}
        <Link to="/register" className="font-semibold text-secondary-600">
          Crea una
        </Link>
      </p>
    </div>
  )
}
