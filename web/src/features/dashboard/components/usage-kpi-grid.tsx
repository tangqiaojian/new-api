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
import type { LucideIcon } from 'lucide-react'
import {
  Activity,
  ArrowDownToLine,
  ArrowUpFromLine,
  Database,
} from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import {
  formatKpiPercent,
  formatKpiTokenCount,
  kpiCacheHitRate,
  kpiInputTokens,
  kpiSuccessRate,
} from '@/features/dashboard/lib/stats'
import { cn } from '@/lib/utils'

export type UsageKpiStats = {
  totalCount: number
  promptTokens: number
  completionTokens: number
  cacheReadTokens: number
  successCount: number
}

interface UsageKpiGridProps {
  stats: UsageKpiStats | null
  loading?: boolean
  className?: string
}

type IconTone =
  | 'violet'
  | 'amber'
  | 'rose'
  | 'sky'

/** Sub2API dark: bg-*-900/30 + text-*-400 */
const TONE_CLASSES: Record<IconTone, string> = {
  violet:
    'bg-violet-100 text-violet-600 dark:bg-violet-900/30 dark:text-violet-400',
  amber: 'bg-amber-100 text-amber-600 dark:bg-amber-900/30 dark:text-amber-400',
  rose: 'bg-rose-100 text-rose-600 dark:bg-rose-900/30 dark:text-rose-400',
  sky: 'bg-sky-100 text-sky-600 dark:bg-sky-900/30 dark:text-sky-400',
}

/**
 * Compact Sub2API-style stat card: icon square left, title / value / subtitle.
 * No giant centered KPI numbers.
 */
function CompactStatCard(props: {
  title: string
  value: ReactNode
  subtitle?: ReactNode
  icon: LucideIcon
  tone: IconTone
  loading?: boolean
}) {
  const Icon = props.icon
  return (
    <div className='bg-card text-card-foreground flex items-start gap-3 rounded-xl border p-4 shadow-sm'>
      <div
        className={cn(
          'flex size-10 shrink-0 items-center justify-center rounded-lg',
          TONE_CLASSES[props.tone]
        )}
      >
        <Icon className='size-5' aria-hidden='true' />
      </div>
      <div className='min-w-0 flex-1 space-y-1'>
        <div className='text-muted-foreground text-xs leading-none'>
          {props.title}
        </div>
        {props.loading ? (
          <Skeleton className='h-5 w-24' />
        ) : (
          <div className='text-xl leading-none font-semibold tracking-tight tabular-nums'>
            {props.value}
          </div>
        )}
        {props.loading ? (
          <Skeleton className='h-3 w-28' />
        ) : props.subtitle ? (
          <div className='text-muted-foreground text-xs leading-snug'>
            {props.subtitle}
          </div>
        ) : null}
      </div>
    </div>
  )
}

/**
 * Frontend-only Sub2API-style usage stats (existing /api/data fields only).
 */
export function UsageKpiGrid(props: UsageKpiGridProps) {
  const { t } = useTranslation()
  const stats = props.stats ?? {
    totalCount: 0,
    promptTokens: 0,
    completionTokens: 0,
    cacheReadTokens: 0,
    successCount: 0,
  }
  const successRate = kpiSuccessRate(stats)
  const cacheHitRate = kpiCacheHitRate(stats)
  const inputTokens = kpiInputTokens(stats)

  return (
    <div
      className={cn(
        'grid grid-cols-2 gap-3 lg:grid-cols-4',
        props.className
      )}
    >
      <CompactStatCard
        title={t('Total Requests')}
        value={formatKpiTokenCount(stats.totalCount)}
        subtitle={
          stats.totalCount > 0
            ? `${t('Success rate')}: ${formatKpiPercent(successRate)}`
            : t('All recorded API calls')
        }
        icon={Activity}
        tone='violet'
        loading={props.loading}
      />
      <CompactStatCard
        title={t('Input TOKEN')}
        value={formatKpiTokenCount(inputTokens)}
        subtitle={t('Includes cache hits')}
        icon={ArrowDownToLine}
        tone='amber'
        loading={props.loading}
      />
      <CompactStatCard
        title={t('Output TOKEN')}
        value={formatKpiTokenCount(stats.completionTokens)}
        subtitle={t('Model generation total')}
        icon={ArrowUpFromLine}
        tone='rose'
        loading={props.loading}
      />
      <CompactStatCard
        title={t('Cache TOKEN')}
        value={formatKpiTokenCount(stats.cacheReadTokens)}
        subtitle={
          stats.promptTokens + stats.cacheReadTokens > 0
            ? `${formatKpiPercent(cacheHitRate)} ${t('Cache read tokens')}`
            : t('Cache read tokens')
        }
        icon={Database}
        tone='sky'
        loading={props.loading}
      />
    </div>
  )
}
