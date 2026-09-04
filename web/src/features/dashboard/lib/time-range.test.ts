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
import { afterEach, describe, expect, test, vi } from 'vitest'

import { dateToUnixTimestamp } from '@/lib/time'

import {
  applyCustomTimeBound,
  buildTimeWindow,
  detectQuickRangeDays,
  resolveUnixTimeRange,
} from './time-range'

describe('dashboard time window', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  test('buildTimeWindow stores a rolling preset range', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-09-04T12:00:00+08:00'))

    const window = buildTimeWindow(7)

    expect(window.selectedRange).toBe(7)
    expect(window.start_timestamp).toBeInstanceOf(Date)
    expect(window.end_timestamp).toBeInstanceOf(Date)
    const start = window.start_timestamp
    const end = window.end_timestamp
    expect(start).toBeDefined()
    expect(end).toBeDefined()
    if (!start || !end) return
    expect(Math.round((end.getTime() - start.getTime()) / 86_400_000)).toBe(7)
  })

  test('resolveUnixTimeRange prefers explicit start and end over the preset', () => {
    const start = new Date('2026-08-01T08:00:00+08:00')
    const end = new Date('2026-08-03T18:30:00+08:00')

    expect(
      resolveUnixTimeRange({
        selectedRange: 7,
        start_timestamp: start,
        end_timestamp: end,
      })
    ).toEqual({
      start_timestamp: dateToUnixTimestamp(start),
      end_timestamp: dateToUnixTimestamp(end),
    })
  })

  test('resolveUnixTimeRange swaps inverted custom bounds', () => {
    const start = new Date('2026-08-10T00:00:00+08:00')
    const end = new Date('2026-08-01T00:00:00+08:00')

    expect(
      resolveUnixTimeRange({
        selectedRange: 0,
        start_timestamp: start,
        end_timestamp: end,
      })
    ).toEqual({
      start_timestamp: dateToUnixTimestamp(end),
      end_timestamp: dateToUnixTimestamp(start),
    })
  })

  test('resolveUnixTimeRange falls back to the rolling preset when dates are missing', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-09-04T12:00:00+08:00'))
    const expected = buildTimeWindow(1)
    const start = expected.start_timestamp
    const end = expected.end_timestamp
    expect(start).toBeDefined()
    expect(end).toBeDefined()
    if (!start || !end) return

    expect(resolveUnixTimeRange({ selectedRange: 1 })).toEqual({
      start_timestamp: dateToUnixTimestamp(start),
      end_timestamp: dateToUnixTimestamp(end),
    })
  })

  test('detectQuickRangeDays highlights an exact preset duration', () => {
    const window = buildTimeWindow(14)
    expect(
      detectQuickRangeDays(window.start_timestamp, window.end_timestamp)
    ).toBe(14)
  })

  test('detectQuickRangeDays leaves custom durations unselected', () => {
    expect(
      detectQuickRangeDays(
        new Date('2026-08-01T00:00:00+08:00'),
        new Date('2026-08-04T12:00:00+08:00')
      )
    ).toBeNull()
  })

  test('applyCustomTimeBound marks a non-preset window as custom', () => {
    const next = applyCustomTimeBound(
      buildTimeWindow(7),
      'end_timestamp',
      new Date('2026-08-20T00:00:00+08:00')
    )
    expect(next.selectedRange).toBe(0)
    expect(next.end_timestamp).toEqual(new Date('2026-08-20T00:00:00+08:00'))
  })
})
