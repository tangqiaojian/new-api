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
import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  ArrowDownToLine,
  ArrowUpFromLine,
  CalendarDays,
  Cpu,
  Hash,
  Sparkles,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { IconBadge, type IconBadgeTone } from '@/components/ui/icon-badge'
import { Skeleton } from '@/components/ui/skeleton'
import {
  getDailyModelTokenData,
  getSelfDailyModelTokenData,
} from '@/features/dashboard/api'
import { useAutoRefresh } from '@/features/dashboard/hooks/use-auto-refresh'
import { aggregateTodayModelTokens } from '@/features/dashboard/lib'
import { toIntlLocale } from '@/i18n/languages'
import {
  formatCompactNumber,
  formatNumber,
  formatQuota,
} from '@/lib/format'
import { ROLE } from '@/lib/roles'
import {
  dateToUnixTimestamp,
  getEndOfDay,
  getStartOfDay,
} from '@/lib/time'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

const MODEL_BAR_TONES = [
  'bg-chart-1',
  'bg-chart-2',
  'bg-chart-3',
  'bg-chart-4',
  'bg-chart-5',
] as const

const MAX_VISIBLE_MODELS = 12

interface TodayModelTokensPanelProps {
  includeCache?: boolean
}

function formatStatNumber(
  value: number,
  locale: Intl.LocalesArgument,
  compact = true
) {
  const fullValue = formatNumber(value, locale)
  const displayValue = compact
    ? formatCompactNumber(value, locale)
    : fullValue
  return { displayValue, fullValue }
}

export function TodayModelTokensPanel(props: TodayModelTokensPanelProps) {
  const { t, i18n } = useTranslation()
  const { refetchInterval } = useAutoRefresh()
  const user = useAuthStore((state) => state.auth.user)
  const isAdmin = !!(user?.role && user.role >= ROLE.ADMIN)
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)

  const timeRange = useMemo(() => {
    const now = new Date()
    return {
      start_timestamp: dateToUnixTimestamp(getStartOfDay(now)),
      end_timestamp: dateToUnixTimestamp(getEndOfDay(now)),
    }
  }, [])

  const todayLabel = useMemo(() => {
    return new Intl.DateTimeFormat(locale, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      weekday: 'short',
    }).format(new Date())
  }, [locale])

  const query = useQuery({
    queryKey: [
      'dashboard',
      'models',
      'today-model-tokens',
      timeRange,
      isAdmin,
      props.includeCache,
    ],
    queryFn: () =>
      isAdmin
        ? getDailyModelTokenData({
            ...timeRange,
            include_cache: props.includeCache,
          })
        : getSelfDailyModelTokenData({
            ...timeRange,
            include_cache: props.includeCache,
          }),
    select: (res) => (res.success ? res.data : []),
    staleTime: 60_000,
    refetchInterval: refetchInterval || undefined,
  })

  const summary = useMemo(
    () => aggregateTodayModelTokens(query.data ?? []),
    [query.data]
  )

  const visibleModels = summary.models.slice(0, MAX_VISIBLE_MODELS)
  const hiddenModelCount = Math.max(
    0,
    summary.models.length - visibleModels.length
  )
  const loading = query.isLoading
  const error = query.isError

  const summaryCards: Array<{
    key: string
    title: string
    value: string
    fullValue: string
    icon: typeof Hash
    iconTone: IconBadgeTone
    barClass: string
    ratio: number
  }> = (() => {
    const total = Math.max(summary.totalTokens, 1)
    const totalFmt = formatStatNumber(summary.totalTokens, locale)
    const promptFmt = formatStatNumber(summary.promptTokens, locale)
    const completionFmt = formatStatNumber(summary.completionTokens, locale)
    const requestFmt = formatStatNumber(summary.requestCount, locale, false)
    return [
      {
        key: 'total',
        title: t('Total Tokens'),
        value: totalFmt.displayValue,
        fullValue: totalFmt.fullValue,
        icon: Hash,
        iconTone: 'success',
        barClass: 'bg-success',
        ratio: 1,
      },
      {
        key: 'prompt',
        title: t('Input Tokens'),
        value: promptFmt.displayValue,
        fullValue: promptFmt.fullValue,
        icon: ArrowDownToLine,
        iconTone: 'info',
        barClass: 'bg-info',
        ratio: summary.promptTokens / total,
      },
      {
        key: 'completion',
        title: t('Output Tokens'),
        value: completionFmt.displayValue,
        fullValue: completionFmt.fullValue,
        icon: ArrowUpFromLine,
        iconTone: 'chart-2',
        barClass: 'bg-chart-2',
        ratio: summary.completionTokens / total,
      },
      {
        key: 'requests',
        title: t('Requests'),
        value: requestFmt.displayValue,
        fullValue: requestFmt.fullValue,
        icon: Sparkles,
        iconTone: 'chart-4',
        barClass: 'bg-chart-4',
        ratio: summary.models.length > 0 ? 1 : 0,
      },
    ]
  })()

  return (
    <div className='overflow-hidden rounded-xl border bg-gradient-to-br from-background via-background to-muted/30 shadow-sm'>
      <div className='flex flex-col gap-2 border-b px-3 py-3 sm:flex-row sm:items-center sm:justify-between sm:px-5 sm:py-3.5'>
        <div className='flex min-w-0 items-center gap-2.5'>
          <IconBadge tone='chart-1' size='sm' className='shrink-0'>
            <CalendarDays />
          </IconBadge>
          <div className='min-w-0'>
            <div className='text-sm font-semibold tracking-tight'>
              {t("Today's Model Tokens")}
            </div>
            <div className='text-muted-foreground truncate text-xs'>
              {isAdmin
                ? t('Platform-wide usage for today')
                : t('Your model token usage for today')}
            </div>
          </div>
        </div>
        <div className='bg-muted/70 text-muted-foreground inline-flex w-fit items-center gap-1.5 rounded-full border px-2.5 py-1 text-[11px] font-medium tabular-nums'>
          <CalendarDays className='size-3.5 opacity-70' />
          {todayLabel}
        </div>
      </div>

      <div className='divide-border/60 grid min-w-0 grid-cols-2 divide-x divide-y sm:grid-cols-4 sm:divide-y-0'>
        {summaryCards.map((card, idx) => {
          const Icon = card.icon
          const barWidth = `${Math.max(0, Math.min(card.ratio, 1)) * 100}%`
          return (
            <div
              key={card.key}
              className={cn(
                'min-w-0 px-3 py-3 sm:px-5 sm:py-4',
                idx === summaryCards.length - 1 &&
                  summaryCards.length % 2 !== 0 &&
                  'col-span-2 sm:col-span-1'
              )}
            >
              <div className='flex min-w-0 items-center gap-1.5 sm:gap-2'>
                <IconBadge
                  tone={card.iconTone}
                  size='stat'
                  className='size-4 rounded-sm sm:size-7 sm:rounded-md [&>svg]:size-2.5 sm:[&>svg]:size-3.5'
                >
                  <Icon />
                </IconBadge>
                <div className='text-muted-foreground truncate text-[11px] leading-4 font-medium tracking-wide uppercase sm:text-xs sm:tracking-wider'>
                  {card.title}
                </div>
              </div>
              {loading ? (
                <div className='mt-2 flex flex-col gap-1.5'>
                  <Skeleton className='h-6 w-16 sm:h-7 sm:w-20' />
                  <Skeleton className='h-1.5 w-full' />
                </div>
              ) : error ? (
                <div className='text-muted-foreground mt-2 font-mono text-base font-bold sm:text-2xl'>
                  --
                </div>
              ) : (
                <>
                  <div
                    className='text-foreground mt-1.5 max-w-full truncate font-mono text-base leading-tight font-bold tracking-tight tabular-nums sm:mt-2 sm:text-2xl sm:leading-normal'
                    title={card.fullValue}
                  >
                    {card.value}
                  </div>
                  <div className='bg-muted mt-2 h-1.5 w-full overflow-hidden rounded-full'>
                    <div
                      className={cn(
                        'h-full rounded-full transition-all duration-500',
                        card.barClass
                      )}
                      style={{ width: barWidth }}
                    />
                  </div>
                </>
              )}
            </div>
          )
        })}
      </div>

      <div className='border-t px-3 py-3 sm:px-5 sm:py-4'>
        <div className='mb-3 flex items-center justify-between gap-2'>
          <div className='flex items-center gap-2'>
            <IconBadge tone='chart-3' size='sm'>
              <Cpu />
            </IconBadge>
            <div className='text-sm font-semibold'>{t('By model')}</div>
            {!loading && !error && (
              <span className='text-muted-foreground text-xs tabular-nums'>
                {t('{{count}} models', { count: summary.models.length })}
              </span>
            )}
          </div>
          {!loading && !error && summary.quota > 0 && (
            <div className='text-muted-foreground text-xs tabular-nums'>
              {t('Quota')}: {formatQuota(summary.quota)}
            </div>
          )}
        </div>

        {loading ? (
          <div className='space-y-3'>
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className='space-y-1.5'>
                <div className='flex justify-between gap-2'>
                  <Skeleton className='h-4 w-36' />
                  <Skeleton className='h-4 w-16' />
                </div>
                <Skeleton className='h-2 w-full' />
              </div>
            ))}
          </div>
        ) : error ? (
          <div className='text-muted-foreground py-6 text-center text-sm'>
            {t('Failed to load today model tokens')}
          </div>
        ) : summary.models.length === 0 ? (
          <div className='text-muted-foreground flex flex-col items-center justify-center gap-1 py-10 text-center'>
            <Cpu className='text-muted-foreground/50 mb-1 size-8' />
            <div className='text-sm font-medium'>{t('No token usage today')}</div>
            <div className='max-w-sm text-xs leading-relaxed'>
              {t(
                'Model token stats will appear here after you make API calls today.'
              )}
            </div>
          </div>
        ) : (
          <div className='space-y-3'>
            {visibleModels.map((row, index) => {
              const tokensFmt = formatStatNumber(row.totalTokens, locale)
              const sharePct = Math.round(row.share * 1000) / 10
              const barWidth = `${Math.max(row.share * 100, row.totalTokens > 0 ? 1.5 : 0)}%`
              const barTone = MODEL_BAR_TONES[index % MODEL_BAR_TONES.length]
              return (
                <div
                  key={row.modelName}
                  className='group rounded-lg border border-transparent px-1 py-1.5 transition-colors hover:border-border/70 hover:bg-muted/30 sm:px-2'
                >
                  <div className='mb-1.5 flex min-w-0 items-center justify-between gap-3'>
                    <div className='flex min-w-0 items-center gap-2'>
                      <span className='text-muted-foreground/70 w-5 shrink-0 text-right font-mono text-[11px] tabular-nums'>
                        {index + 1}
                      </span>
                      <span
                        className='truncate text-sm font-medium'
                        title={row.modelName}
                      >
                        {row.modelName}
                      </span>
                      <span className='text-muted-foreground hidden shrink-0 text-[11px] tabular-nums sm:inline'>
                        {t('{{count}} requests', {
                          count: row.requestCount,
                        })}
                      </span>
                    </div>
                    <div className='flex shrink-0 items-baseline gap-2'>
                      <span
                        className='font-mono text-sm font-semibold tabular-nums'
                        title={tokensFmt.fullValue}
                      >
                        {tokensFmt.displayValue}
                      </span>
                      <span className='text-muted-foreground w-12 text-right text-xs tabular-nums'>
                        {sharePct}%
                      </span>
                    </div>
                  </div>
                  <div className='bg-muted ml-7 h-2 overflow-hidden rounded-full sm:ml-7'>
                    <div
                      className={cn(
                        'h-full rounded-full transition-all duration-500 ease-out',
                        barTone
                      )}
                      style={{ width: barWidth }}
                    />
                  </div>
                  <div className='text-muted-foreground/70 ml-7 mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-[11px] tabular-nums'>
                    <span>
                      {t('In')}:{' '}
                      {
                        formatStatNumber(row.promptTokens, locale)
                          .displayValue
                      }
                    </span>
                    <span>
                      {t('Out')}:{' '}
                      {
                        formatStatNumber(row.completionTokens, locale)
                          .displayValue
                      }
                    </span>
                    {row.cachedTokens > 0 && (
                      <span>
                        {t('Cache')}:{' '}
                        {
                          formatStatNumber(row.cachedTokens, locale)
                            .displayValue
                        }
                      </span>
                    )}
                  </div>
                </div>
              )
            })}
            {hiddenModelCount > 0 && (
              <div className='text-muted-foreground pt-1 text-center text-xs'>
                {t('+{{count}} more models', { count: hiddenModelCount })}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
