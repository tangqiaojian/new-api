/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, test, vi } from 'vitest'

import { DateTimePicker } from '@/components/datetime-picker'

function StatefulPicker(props: { initialValue: Date }) {
  const [value, setValue] = useState<Date | undefined>(props.initialValue)
  return <DateTimePicker value={value} onChange={setValue} />
}

async function openPicker(label: RegExp | string) {
  const user = userEvent.setup()
  await user.click(screen.getByRole('button', { name: label }))
  await waitFor(() => {
    expect(screen.getByLabelText('Hour')).toBeVisible()
  })
  return user
}

function getInMonthDayButton(day: string) {
  const grid = screen.getByRole('grid')
  const buttons = within(grid)
    .getAllByRole('button')
    .filter((button) => button.textContent?.trim() === day)
  const inMonth = buttons.find(
    (button) => button.getAttribute('data-outside') !== 'true'
  )
  const dayButton = inMonth ?? buttons[0]
  if (!dayButton) {
    throw new Error(`calendar day ${day} was not found`)
  }
  return dayButton
}

describe('DateTimePicker', () => {
  test('shows YYYY-MM-DD HH:mm on the trigger instead of a date-only label', () => {
    render(<DateTimePicker value={new Date(2026, 8, 4, 14, 30)} />)

    expect(
      screen.getByRole('button', { name: /2026-09-04 14:30/ })
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: /^2026-09-04$/ })
    ).not.toBeInTheDocument()
  })

  test('lets the user pick hour and minute by clicking selects in the calendar popover', async () => {
    const onChange = vi.fn()
    render(
      <DateTimePicker
        value={new Date(2026, 8, 4, 9, 0)}
        onChange={onChange}
      />
    )

    const user = await openPicker(/2026-09-04 09:00/)
    expect(document.querySelector('input[type="time"]')).toBeNull()

    await user.selectOptions(screen.getByLabelText('Hour'), '14')
    await user.selectOptions(screen.getByLabelText('Minute'), '45')

    const lastCall = onChange.mock.calls.at(-1)?.[0] as Date | undefined
    expect(lastCall).toBeInstanceOf(Date)
    expect(lastCall?.getHours()).toBe(14)
    expect(lastCall?.getMinutes()).toBe(45)
  })

  test('keeps the popover open after picking a day so time can still be clicked', async () => {
    render(<StatefulPicker initialValue={new Date(2026, 8, 4, 9, 0)} />)

    const user = await openPicker(/2026-09-04 09:00/)
    const trigger = screen.getByRole('button', { name: /2026-09-04 09:00/ })
    expect(trigger).toHaveAttribute('aria-expanded', 'true')

    await user.click(getInMonthDayButton('15'))

    expect(trigger).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByLabelText('Hour')).toBeVisible()
    expect(screen.getByLabelText('Minute')).toBeVisible()
    await user.selectOptions(screen.getByLabelText('Hour'), '18')
    await user.selectOptions(screen.getByLabelText('Minute'), '05')

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: /2026-09-15 18:05/ })
      ).toBeInTheDocument()
    })
    expect(screen.getByLabelText('Hour')).toBeVisible()
  })
})
