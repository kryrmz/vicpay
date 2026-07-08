import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'

// Vitest no expone globals (describe/it/afterEach) por defecto en este
// proyecto, asi que @testing-library/react no puede auto-registrar su
// limpieza. La registramos aqui para que cada test empiece con un DOM limpio.
afterEach(() => {
  cleanup()
})
