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
import {
  Activity,
  Clock3,
  Coins,
  Gauge,
  KeyRound,
  Layers,
  Wallet,
  Zap,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { OpsStatCard } from '@/features/dashboard/components/ops-stat-card'
import {
  formatDurationSeconds,
  formatTokenSplitLine,
  formatTokens,
  type OpsDashboardStats,
} from '@/features/dashboard/lib/stats'
import { formatQuota } from '@/lib/format'
import { cn } from '@/lib/utils'

interface OpsStatsGridProps {
  stats: OpsDashboardStats | null
  loading?: boolean
  className?: string
}

/**
 * Sub2API-style 8 compact stats cards shared by Overview, Dashboard, and user detail.
 * Left icon square + right title / 20px value / 12px subtitle — never giant centered KPIs.
 */
export function OpsStatsGrid(props: OpsStatsGridProps) {
  const { t } = useTranslation()
  const stats = props.stats
  const today = stats?.today
  const lifetime = stats?.lifetime
  const loading = props.loading

  const todayTokenUsed =
    (today?.promptTokens ?? 0) +
    (today?.completionTokens ?? 0) +
    (today?.cacheReadTokens ?? 0)
  const lifetimeTokenUsed =
    (lifetime?.promptTokens ?? 0) +
    (lifetime?.completionTokens ?? 0) +
    (lifetime?.cacheReadTokens ?? 0)

  return (
    <div className={cn('space-y-3', props.className)}>
      <div className='grid grid-cols-2 gap-3 lg:grid-cols-4'>
        <OpsStatCard
          title={t('Balance')}
          icon={Wallet}
          tone='emerald'
          loading={loading}
          value={formatQuota(stats?.balanceQuota ?? 0)}
          subtitle={`${t('Used')}: ${formatQuota(stats?.usedQuota ?? 0)}`}
        />
        <OpsStatCard
          title={t('API Keys')}
          icon={KeyRound}
          tone='blue'
          loading={loading}
          value={stats?.apiKeysTotal ?? 0}
          subtitle={`${stats?.apiKeysActive ?? 0} ${t('enabled')}`}
        />
        <OpsStatCard
          title={t('Today Requests')}
          icon={Activity}
          tone='green'
          loading={loading}
          value={(today?.requests ?? 0).toLocaleString()}
          subtitle={`${t('Total')}: ${(lifetime?.requests ?? 0).toLocaleString()}`}
        />
        <OpsStatCard
          title={t('Today Cost')}
          icon={Coins}
          tone='purple'
          loading={loading}
          value={
            <span>
              {formatQuota(today?.quota ?? 0)}
              <span className='text-muted-foreground ml-1 text-sm font-normal'>
                / {formatQuota(today?.quota ?? 0)}
              </span>
            </span>
          }
          subtitle={`${t('Total')}: ${formatQuota(lifetime?.quota ?? 0)} / ${formatQuota(lifetime?.quota ?? 0)}`}
        />
      </div>

      <div className='grid grid-cols-2 gap-3 lg:grid-cols-4'>
        <OpsStatCard
          title={t('Today Tokens')}
          icon={Zap}
          tone='amber'
          loading={loading}
          value={formatTokens(todayTokenUsed || (today?.tokenUsed ?? 0))}
          subtitle={formatTokenSplitLine(t, {
            promptTokens: today?.promptTokens ?? 0,
            completionTokens: today?.completionTokens ?? 0,
            cacheReadTokens: today?.cacheReadTokens ?? 0,
          })}
        />
        <OpsStatCard
          title={t('Total Tokens')}
          icon={Layers}
          tone='indigo'
          loading={loading}
          value={formatTokens(lifetimeTokenUsed || (lifetime?.tokenUsed ?? 0))}
          subtitle={formatTokenSplitLine(t, {
            promptTokens: lifetime?.promptTokens ?? 0,
            completionTokens: lifetime?.completionTokens ?? 0,
            cacheReadTokens: lifetime?.cacheReadTokens ?? 0,
          })}
        />
        <OpsStatCard
          title={t('Performance')}
          icon={Gauge}
          tone='violet'
          loading={loading}
          value={
            <span>
              {formatTokens(stats?.rpm ?? 0)}{' '}
              <span className='text-muted-foreground text-sm font-normal'>
                RPM
              </span>
            </span>
          }
          subtitle={`${formatTokens(stats?.tpm ?? 0)} TPM`}
        />
        <OpsStatCard
          title={t('Avg Response')}
          icon={Clock3}
          tone='rose'
          loading={loading}
          value={formatDurationSeconds(stats?.avgUseTimeSec ?? 0)}
          subtitle={t('Average time')}
        />
      </div>
    </div>
  )
}
