import { useState } from 'react'
import { Button } from '../ui/Button'
import { OtpInput } from '../ui/OtpInput'

export interface OtpStepProps {
  phone: string
  onVerify: (code: string) => Promise<void> | void
  onChangePhone: () => void
  submitting: boolean
  error: string | null
}

const CODE_LENGTH = 6

export function OtpStep({ phone, onVerify, onChangePhone, submitting, error }: OtpStepProps) {
  const [code, setCode] = useState('')

  function handleSubmit(): void {
    if (code.length !== CODE_LENGTH) return
    void onVerify(code)
  }

  return (
    <div className="flex flex-col gap-4">
      <p className="text-sm text-fg-secondary">
        Enviamos un codigo de {CODE_LENGTH} digitos a <span className="font-semibold text-fg">{phone}</span>.
      </p>
      <OtpInput value={code} onChange={setCode} disabled={submitting} error={error ?? undefined} />
      {error ? <p className="text-sm text-danger">{error}</p> : null}
      <p className="text-xs text-fg-muted">Modo de prueba: el codigo de verificacion es 000000.</p>
      <Button
        type="button"
        variant="primary"
        className="w-full"
        loading={submitting}
        disabled={code.length !== CODE_LENGTH}
        onClick={handleSubmit}
      >
        Verificar
      </Button>
      <Button type="button" variant="ghost" className="w-full" onClick={onChangePhone} disabled={submitting}>
        Cambiar numero
      </Button>
    </div>
  )
}
