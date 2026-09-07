import { describe, expect, it } from 'vitest'

import { formatResetTimestamp } from '../format-reset-time'

const t = (key: string) => key

describe('formatResetTimestamp', () => {
  it('returns Never reset for missing or zero timestamps', () => {
    expect(formatResetTimestamp(t as never, undefined)).toBe('Never reset')
    expect(formatResetTimestamp(t as never, null)).toBe('Never reset')
    expect(formatResetTimestamp(t as never, 0)).toBe('Never reset')
  })

  it('formats positive unix seconds', () => {
    const formatted = formatResetTimestamp(t as never, 1700000000)
    expect(formatted).not.toBe('Never reset')
    expect(formatted).toMatch(/\d{4}-\d{2}-\d{2}/)
  })
})
