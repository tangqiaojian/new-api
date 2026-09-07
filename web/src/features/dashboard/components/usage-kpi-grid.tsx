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
  ActivityIcon,
  ArrowDownToLineIcon,
  ArrowUpFromLineIcon,
  DatabaseIcon,
} from 'lucide-react'
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

function Pill(props: { value: string; tone: 'success' | 'info' }) {
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium tabular-nums',
        props.tone === 'success' &&
          'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300',
        props.tone === 'info' &&
          'bg-sky-500/15 text-sky-700 dark:text-sky-300'
      )}
    >
      {props.value}
    </span>
  )
}

function KpiCard(props: {
  title: string
  value: string
  subtitle: string
  icon: typeof ActivityIcon
  pill?: { value: string; tone: 'success' | 'info' }
  iconClassName: string
  loading?: boolean
}) {
  const Icon = props.icon
  return (
    <div className='bg-card text-card-foreground rounded-2xl border border-border/60 p-4 shadow-sm sm:p-5'>
      <div className='mb-3 flex items-start justify-between gap-3'>
        <div
          className={cn(
            'flex size-10 items-center justify-center rounded-full',
            props.iconClassName
          )}
        >
          <Icon className='size-5' aria-hidden='true' />
        </div>
        {props.pill ? (
          <Pill value={props.pill.value} tone={props.pill.tone} />
        ) : null}
      </div>
      <div className='text-muted-foreground text-sm'>{props.title}</div>
      {props.loading ? (
        <Skeleton className='mt-2 h-9 w-28' />
      ) : (
        <div className='mt-1 text-3xl font-semibold tracking-tight tabular-nums sm:text-4xl'>
          {props.value}
        </div>
      )}
      <div className='text-muted-foreground mt-1 text-xs'>{props.subtitle}</div>
    </div>
  )
}

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
        'grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4',
        props.className
      )}
    >
      <KpiCard
        title={t('Total Requests')}
        value={formatKpiTokenCount(stats.totalCount)}
        subtitle={t('All recorded API calls')}
        icon={ActivityIcon}
        iconClassName='bg-violet-500/10 text-violet-600 dark:text-violet-300'
        pill={
          stats.totalCount > 0
            ? { value: formatKpiPercent(successRate), tone: 'success' }
            : undefined
        }
        loading={props.loading}
      />
      <KpiCard
        title={t('Input TOKEN')}
        value={formatKpiTokenCount(inputTokens)}
        subtitle={t('Includes cache hits')}
        icon={ArrowDownToLineIcon}
        iconClassName='bg-amber-500/10 text-amber-700 dark:text-amber-300'
        loading={props.loading}
      />
      <KpiCard
        title={t('Output TOKEN')}
        value={formatKpiTokenCount(stats.completionTokens)}
        subtitle={t('Model generation total')}
        icon={ArrowUpFromLineIcon}
        iconClassName='bg-rose-500/10 text-rose-600 dark:text-rose-300'
        loading={props.loading}
      />
      <KpiCard
        title={t('Cache TOKEN')}
        value={formatKpiTokenCount(stats.cacheReadTokens)}
        subtitle={t('Cache read tokens')}
        icon={DatabaseIcon}
        iconClassName='bg-sky-500/10 text-sky-700 dark:text-sky-300'
        pill={
          stats.promptTokens + stats.cacheReadTokens > 0
            ? { value: formatKpiPercent(cacheHitRate), tone: 'info' }
            : undefined
        }
        loading={props.loading}
      />
    </div>
  )
}
