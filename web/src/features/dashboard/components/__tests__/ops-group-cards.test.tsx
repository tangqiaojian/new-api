import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

import { OpsGroupCards } from '@/features/dashboard/components/ops-group-cards'
import { OpsStatCard } from '@/features/dashboard/components/ops-stat-card'
import { Wallet } from 'lucide-react'

describe('OpsStatCard dark icon tones', () => {
  it('uses Sub2API dark bg-*-900/30 and text-*-400 classes', () => {
    const { container } = render(
      <OpsStatCard
        title='Balance'
        value='1'
        icon={Wallet}
        tone='emerald'
      />
    )
    const iconWrap = container.querySelector('.bg-emerald-100')
    expect(iconWrap?.className).toContain('dark:bg-emerald-900/30')
    expect(iconWrap?.className).toContain('dark:text-emerald-400')
  })
})

describe('OpsGroupCards Sub2API layout', () => {
  it('shows name, header total, today cost, requests, tokens, and never reset', () => {
    render(
      <OpsGroupCards
        items={[
          {
            name: 'grok',
            totalCost: 200,
            todayCost: 40,
            requests: 3,
            tokens: 1500,
            limitPercent: 20,
            resetAt: 0,
          },
        ]}
      />
    )
    expect(screen.getByText('grok')).toBeInTheDocument()
    expect(screen.getByText('Today Cost')).toBeInTheDocument()
    expect(screen.getByText('Requests')).toBeInTheDocument()
    expect(screen.getByText('Tokens')).toBeInTheDocument()
    expect(screen.getByText('Never reset')).toBeInTheDocument()
  })
})
