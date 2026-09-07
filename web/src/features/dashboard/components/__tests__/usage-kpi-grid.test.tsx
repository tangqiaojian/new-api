import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

import { UsageKpiGrid } from '@/features/dashboard/components/usage-kpi-grid'

describe('UsageKpiGrid Sub2API compact style', () => {
  it('renders four compact stats without giant KPI layout', () => {
    const { container } = render(
      <UsageKpiGrid
        stats={{
          totalCount: 10,
          promptTokens: 100,
          completionTokens: 40,
          cacheReadTokens: 20,
          successCount: 9,
        }}
      />
    )

    expect(screen.getByText('Total Requests')).toBeInTheDocument()
    expect(screen.getByText('Input TOKEN')).toBeInTheDocument()
    expect(screen.getByText('Output TOKEN')).toBeInTheDocument()
    expect(screen.getByText('Cache TOKEN')).toBeInTheDocument()

    const icon = container.querySelector('.dark\\:bg-violet-900\\/30')
    expect(icon?.className).toContain('dark:bg-violet-900/30')
    expect(icon?.className).toContain('dark:text-violet-400')
    expect(container.querySelector('.text-4xl')).toBeNull()
  })
})
