import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { OtpStep } from '../components/register/OtpStep'
import type { PhoneStepValues } from '../components/register/PhoneStep'
import { PhoneStep } from '../components/register/PhoneStep'
import { useAuthStore } from '../store/auth.store'

type WizardStep = 'phone' | 'otp'

export function RegisterPage() {
  const navigate = useNavigate()
  const register = useAuthStore((state) => state.register)
  const verifyPhone = useAuthStore((state) => state.verifyPhone)
  const status = useAuthStore((state) => state.status)
  const error = useAuthStore((state) => state.error)
  const clearError = useAuthStore((state) => state.clearError)

  const [step, setStep] = useState<WizardStep>('phone')
  const [phone, setPhone] = useState('')

  async function handlePhoneSubmit(values: PhoneStepValues): Promise<void> {
    try {
      await register(values.phone, values.password)
      setPhone(values.phone)
      setStep('otp')
    } catch {
      // El error ya queda expuesto via el store; no hay nada mas que hacer aqui.
    }
  }

  async function handleVerify(code: string): Promise<void> {
    try {
      await verifyPhone(code)
      navigate('/app/home', { replace: true })
    } catch {
      // El error ya queda expuesto via el store.
    }
  }

  function handleChangePhone(): void {
    clearError()
    setStep('phone')
  }

  return (
    <div className="mx-auto flex min-h-dvh max-w-sm flex-col justify-center gap-6 px-6 py-10">
      <header className="text-center">
        <h1 className="text-2xl font-bold text-fg">Crea tu cuenta VicPay</h1>
        <p className="mt-1 text-sm text-fg-secondary">
          {step === 'phone' ? 'Paso 1 de 2: tus datos de acceso.' : 'Paso 2 de 2: verifica tu telefono.'}
        </p>
      </header>

      {step === 'phone' ? (
        <PhoneStep onSubmit={handlePhoneSubmit} submitting={status === 'loading'} error={error} />
      ) : (
        <OtpStep
          phone={phone}
          onVerify={handleVerify}
          onChangePhone={handleChangePhone}
          submitting={status === 'loading'}
          error={error}
        />
      )}

      <p className="text-center text-sm text-fg-secondary">
        Ya tienes cuenta?{' '}
        <Link to="/login" className="font-semibold text-secondary-600">
          Inicia sesion
        </Link>
      </p>
    </div>
  )
}
