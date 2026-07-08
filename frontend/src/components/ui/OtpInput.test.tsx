import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, it } from 'vitest'
import { OtpInput } from './OtpInput'

function ControlledOtp() {
  const [value, setValue] = useState('')
  return (
    <>
      <OtpInput value={value} onChange={setValue} />
      <output data-testid="value">{value}</output>
    </>
  )
}

describe('OtpInput', () => {
  it('renderiza una casilla por digito', () => {
    render(<ControlledOtp />);
    expect(screen.getAllByRole('textbox')).toHaveLength(6)
  })

  it('avanza automaticamente de casilla al escribir y arma el codigo completo', async () => {
    const user = userEvent.setup()
    render(<ControlledOtp />)

    const boxes = screen.getAllByRole('textbox')
    await user.click(boxes[0])
    await user.keyboard('123456')

    // El foco avanza solo, asi que escribir 6 digitos seguidos llena todas las casillas.
    expect(screen.getByTestId('value').textContent).toBe('123456')
    expect(boxes[5]).toHaveFocus()
  })

  it('borra el digito anterior y mueve el foco con Backspace en una casilla vacia', async () => {
    const user = userEvent.setup()
    render(<ControlledOtp />)

    const boxes = screen.getAllByRole('textbox')
    await user.click(boxes[0])
    await user.keyboard('12')
    expect(screen.getByTestId('value').textContent).toBe('12')

    await user.keyboard('{Backspace}')
    expect(screen.getByTestId('value').textContent).toBe('1')
    expect(boxes[1]).toHaveFocus()
  })

  it('ignora caracteres que no sean digitos', async () => {
    const user = userEvent.setup()
    render(<ControlledOtp />)

    const boxes = screen.getAllByRole('textbox')
    await user.click(boxes[0])
    await user.keyboard('a')
    expect(screen.getByTestId('value').textContent).toBe('')
  })
})
