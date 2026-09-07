import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

import { OpsStatsGrid } from '@/features/dashboard/components/ops-stats-grid'
import type { OpsDashboardStats } from '@/features/dashboard/lib/stats'

const fixture: OpsDashboardStats = {
  balanceQuota: 1_000_000,
  usedQuota: 250_000,
  apiKeysTotal: 3,
  apiKeysActive: 2,
  today: {
    requests: 12,
    quota: 50,
    standardQuota: 100,
    promptTokens: 1000,
    completionTokens: 200,
    cacheReadTokens: 50,
    cacheWriteTokens: 10,
    tokenUsed: 1250,
  },
  lifetime: {
    requests: 900,
    quota: 5000,
    standardQuota: 8000,
    promptTokens: 10_000,
    completionTokens: 2000,
    cacheReadTokens: 500,
    cacheWriteTokens: 100,
    tokenUsed: 12_500,
  },
  rpm: 4.2,
  tpm: 1500,
  avgUseTimeSec: 1.25,
}

describe('OpsStatsGrid Sub2API layout', () => {
  it('renders eight compact stats titles and actual/standard cost', () => {
    render(<OpsStatsGrid stats={fixture} />)

    for (const title of [
      'Balance',
      'API Keys',
      'Today Requests',
      'Today Cost',
      'Today Tokens',
      'Total Tokens',
      'Performance',
      'Avg Response',
    ]) {
      expect(screen.getByText(title)).toBeInTheDocument()
    }

    expect(screen.getByText(/RPM/)).toBeInTheDocument()
    expect(screen.getByText(/TPM/)).toBeInTheDocument()
    expect(screen.getAllByText(/Input:/).length).toBeGreaterThanOrEqual(1)
    expect(screen.getAllByText(/Output:/).length).toBeGreaterThanOrEqual(1)
    expect(screen.getAllByText(/Cache:/).length).toBeGreaterThanOrEqual(1)
  })
})
