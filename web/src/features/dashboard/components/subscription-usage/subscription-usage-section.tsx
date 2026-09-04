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
import { useQuery } from '@tanstack/react-query'
import { VChart } from '@visactor/react-vchart'
import { Hash, Loader2, ArrowLeftRight, AlertCircle } from 'lucide-react'
import { useEffect, useMemo, useState, useRef, useCallback } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useTheme } from '@/context/theme-provider'
import {
  getSelfSubscriptionUsage,
  getSelfSubscriptionModelUsage,
  getSelfSubscriptions,
} from '@/features/dashboard/api'
import { DashboardTimeRangeBar } from '@/features/dashboard/components/ui/dashboard-time-range-bar'
import { useAutoRefresh } from '@/features/dashboard/hooks/use-auto-refresh'
import { resolveUnixTimeRange } from '@/features/dashboard/lib'
import type {
  DashboardTimeWindow,
  SubscriptionUsageDataItem,
  SubscriptionUsageFilters,
} from '@/features/dashboard/types'
import { toIntlLocale } from '@/i18n/languages'
import { formatQuotaWithCurrency } from '@/lib/currency'
import { formatCompactNumber, formatNumber } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { VCHART_OPTION } from '@/lib/vchart'
import { useAuthStore } from '@/stores/auth-store'

import { AdminSubscriptionPlanUsage } from './admin-subscription-plan-usage'

let themeManagerPromise: Promise<
  (typeof import('@visactor/vchart'))['ThemeManager']
> | null = null

const TOKEN_COLORS = [
  '#5B8FF9',
  '#5AD8A6',
  '#F6BD16',
  '#E8684A',
  '#6DC8EC',
  '#9270CA',
  '#FF9D4D',
  '#269A99',
  '#FF99C3',
  '#5D7092',
]

const PAGE_SIZE = 20

interface SubscriptionUsageSectionProps {
  filters: SubscriptionUsageFilters
  onFiltersChange: (filters: SubscriptionUsageFilters) => void
  includeCache?: boolean
}

type TrendSeries = 'prompt' | 'completion' | 'cached'

export function SubscriptionUsageSection(props: SubscriptionUsageSectionProps) {
  const userRole = useAuthStore((state) => state.auth.user?.role)
  const isAdmin = Boolean(userRole && userRole >= ROLE.ADMIN)

  // Admin needs per-user plan usage (quota/token remaining), not log traffic charts.
  if (isAdmin) {
    return <AdminSubscriptionPlanUsage />
  }

  return <SelfSubscriptionUsageSection {...props} />
}

/** Ordinary user: personal plan cards + subscription-billed token charts. */
function SelfSubscriptionUsageSection(props: SubscriptionUsageSectionProps) {
  const { t, i18n } = useTranslation()
  const { resolvedTheme } = useTheme()
  const { refetchInterval } = useAutoRefresh()
  const userRole = useAuthStore((state) => state.auth.user?.role)
  const isAdmin = Boolean(userRole && userRole >= ROLE.ADMIN)
  const [themeReady, setThemeReady] = useState(false)
  const themeManagerRef = useRef<
    (typeof import('@visactor/vchart'))['ThemeManager'] | null
  >(null)

  // Number format mode: 'compact' shows 万/亿, 'precise' shows full numbers
  const [compactMode, setCompactMode] = useState(true)
  const includeCache = props.includeCache ?? false

  const modelFilter = props.filters.model
  const selectedRange = props.filters.selectedRange
  const startTimestamp = props.filters.start_timestamp
  const endTimestamp = props.filters.end_timestamp
  const onFiltersChange = props.onFiltersChange
  const [currentPage, setCurrentPage] = useState(1)

  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)

  const timeRange = useMemo(
    () =>
      resolveUnixTimeRange({
        selectedRange,
        start_timestamp: startTimestamp,
        end_timestamp: endTimestamp,
      }),
    [endTimestamp, selectedRange, startTimestamp]
  )

  const handleTimeWindowChange = useCallback(
    (window: DashboardTimeWindow) => {
      onFiltersChange({ ...props.filters, ...window })
      setCurrentPage(1)
    },
    [onFiltersChange, props.filters]
  )

  const handleModelChange = useCallback(
    (value: string) => {
      onFiltersChange({ ...props.filters, model: value })
      setCurrentPage(1)
    },
    [onFiltersChange, props.filters]
  )

  useEffect(() => {
    const updateTheme = async () => {
      setThemeReady(false)
      if (!themeManagerPromise) {
        themeManagerPromise = import('@visactor/vchart').then(
          (m) => m.ThemeManager
        )
      }
      const ThemeManager = await themeManagerPromise
      themeManagerRef.current = ThemeManager
      ThemeManager.setCurrentTheme(resolvedTheme === 'dark' ? 'dark' : 'light')
      setThemeReady(true)
    }
    updateTheme()
  }, [resolvedTheme])

  const queryParams = useMemo(
    () => ({
      ...timeRange,
      ...(modelFilter ? { model: modelFilter } : {}),
      include_cache: includeCache,
    }),
    [timeRange, modelFilter, includeCache]
  )

  const {
    data: dailyData,
    isLoading: dailyLoading,
    isError: dailyError,
  } = useQuery({
    queryKey: [
      'dashboard',
      'subscription-usage',
      timeRange,
      modelFilter,
      includeCache,
    ],
    queryFn: () => getSelfSubscriptionUsage(queryParams),
    select: (res) => (res.success ? res.data : []),
    staleTime: 60_000,
    refetchInterval: refetchInterval || undefined,
  })

  const {
    data: modelData,
    isLoading: modelLoading,
    isError: modelError,
  } = useQuery({
    queryKey: [
      'dashboard',
      'subscription-model-usage',
      timeRange,
      modelFilter,
      includeCache,
    ],
    queryFn: () => getSelfSubscriptionModelUsage(queryParams),
    select: (res) => (res.success ? res.data : []),
    staleTime: 60_000,
    refetchInterval: refetchInterval || undefined,
  })

  const { data: subscriptionInfo } = useQuery({
    queryKey: ['dashboard', 'subscription-self'],
    queryFn: () => getSelfSubscriptions(),
    staleTime: 60_000,
    refetchInterval: refetchInterval || undefined,
  })

  const activeSubscriptions =
    subscriptionInfo?.data?.subscriptions?.filter(
      (s) => s.subscription.status === 'active'
    ) ?? []

  const isLoading = dailyLoading || modelLoading
  const hasError = dailyError || modelError

  // Number formatter: compact mode uses locale-aware compact notation (万/亿 in zh)
  // precise mode uses full number with separators
  const formatInt = useCallback(
    (value: number) =>
      compactMode
        ? Intl.NumberFormat(locale, {
            notation: 'compact',
            maximumFractionDigits: 1,
          }).format(value)
        : Intl.NumberFormat(locale, { maximumFractionDigits: 0 }).format(value),
    [compactMode, locale]
  )

  const formatQuota = (quota: number) => formatQuotaWithCurrency(quota)
  const formatNum = (value: number) =>
    compactMode ? formatCompactNumber(value, locale) : formatNumber(value)

  // Build the daily token usage trend chart spec.
  // Series: prompt_tokens, completion_tokens, and cached_tokens (when includeCache).
  const trendSpec = useMemo(() => {
    const items = isLoading ? [] : (dailyData ?? [])
    const sortedItems = [...items].sort((a, b) => a.date.localeCompare(b.date))

    const series: TrendSeries[] = ['prompt', 'completion']
    if (includeCache) series.push('cached')

    const seriesLabels: Record<TrendSeries, string> = {
      prompt: t('Prompt Tokens'),
      completion: t('Completion Tokens'),
      cached: t('Cached Tokens'),
    }
    const seriesColors: Record<TrendSeries, string> = {
      prompt: TOKEN_COLORS[0],
      completion: TOKEN_COLORS[1],
      cached: TOKEN_COLORS[3],
    }
    const colorMap = series.reduce<Record<string, string>>((acc, s) => {
      acc[seriesLabels[s]] = seriesColors[s]
      return acc
    }, {})

    const values: Array<{
      Date: string
      Series: string
      Tokens: number
    }> = []
    sortedItems.forEach((item) => {
      series.forEach((s) => {
        let tokens = item.cached_tokens
        if (s === 'prompt') {
          tokens = item.prompt_tokens
        } else if (s === 'completion') {
          tokens = item.completion_tokens
        }
        values.push({
          Date: item.date,
          Series: seriesLabels[s],
          Tokens: Number(tokens) || 0,
        })
      })
    })

    const totalTokens = sortedItems.reduce(
      (sum, item) =>
        sum +
        (Number(item.prompt_tokens) || 0) +
        (Number(item.completion_tokens) || 0) +
        (includeCache ? Number(item.cached_tokens) || 0 : 0),
      0
    )

    return {
      type: 'bar',
      data: [{ id: 'subscriptionTrendData', values }],
      xField: 'Date',
      yField: 'Tokens',
      seriesField: 'Series',
      stack: true,
      title: {
        visible: true,
        text: isAdmin
          ? t('Platform Subscription Token Trend')
          : t('Daily Subscription Token Usage Trend'),
        subtext: `${t('Total:')} ${formatInt(totalTokens)}`,
      },
      legends: { visible: true, selectMode: 'single' },
      axes: [
        { orient: 'bottom', type: 'band' },
        {
          orient: 'left',
          type: 'linear',
          label: {
            formatMethod: (value: number) => formatInt(value),
          },
        },
      ],
      tooltip: {
        mark: {
          content: [
            {
              key: (datum: Record<string, unknown>) => datum?.Series,
              value: (datum: Record<string, unknown>) =>
                formatInt(Number(datum?.Tokens) || 0),
            },
          ],
        },
        dimension: {
          content: [
            {
              key: (datum: Record<string, unknown>) => datum?.Series,
              value: (datum: Record<string, unknown>) =>
                Number(datum?.Tokens) || 0,
            },
          ],
          updateContent: (
            array: Array<{ key: string; value: string | number }>
          ) => {
            array.sort(
              (a, b) => (Number(b.value) || 0) - (Number(a.value) || 0)
            )
            let sum = 0
            for (let i = 0; i < array.length; i++) {
              const v = Number(array[i].value) || 0
              sum += v
              array[i].value = formatInt(v)
            }
            array.unshift({
              key: t('Total:'),
              value: formatInt(sum),
            })
            return array
          },
        },
      },
      bar: {
        style: {
          cornerRadius: 2,
        },
      },
      color: { specified: colorMap },
      background: { fill: 'transparent' },
      animation: true,
    }
  }, [dailyData, isLoading, includeCache, isAdmin, t, formatInt])

  // Build the model distribution pie chart spec.
  const modelPieSpec = useMemo(() => {
    const items = isLoading ? [] : (modelData ?? [])
    const sorted = [...items]
      .map((item) => ({
        Model: item.model_name || 'unknown',
        Tokens: Number(item.total_tokens) || 0,
      }))
      .sort((a, b) => b.Tokens - a.Tokens)

    const topModels = sorted.slice(0, 10).map((d) => d.Model)
    const modelColorMap = topModels.reduce<Record<string, string>>(
      (acc, model, i) => {
        acc[model] = TOKEN_COLORS[i % TOKEN_COLORS.length]
        return acc
      },
      {}
    )

    const totalTokens = sorted.reduce((s, d) => s + d.Tokens, 0)

    return {
      type: 'pie',
      data: [{ id: 'subscriptionModelPieData', values: sorted }],
      valueField: 'Tokens',
      categoryField: 'Model',
      outerRadius: 0.8,
      innerRadius: 0.5,
      padAngle: 0.6,
      title: {
        visible: true,
        text: isAdmin
          ? t('Platform Subscription Model Distribution')
          : t('Subscription Model Distribution'),
        subtext: `${t('Total:')} ${formatInt(totalTokens)}`,
      },
      legends: { visible: true, orient: 'left' },
      label: { visible: true },
      color: { specified: modelColorMap },
      tooltip: {
        mark: {
          content: [
            {
              key: (datum: Record<string, unknown>) => datum?.Model,
              value: (datum: Record<string, unknown>) =>
                formatInt(Number(datum?.Tokens) || 0),
            },
          ],
        },
      },
      background: { fill: 'transparent' },
      animation: true,
    }
  }, [modelData, isLoading, isAdmin, t, formatInt])

  // Data fingerprint so VChart remounts when the underlying data changes
  // (react-vchart does not reliably re-render on spec prop changes alone).
  const trendFingerprint = useMemo(() => {
    const items = dailyData ?? []
    let sum = 0
    for (const item of items) sum += item.total_tokens
    return `${items.length}-${sum}-${includeCache}`
  }, [dailyData, includeCache])

  const modelFingerprint = useMemo(() => {
    const items = modelData ?? []
    let sum = 0
    for (const item of items) sum += item.total_tokens
    return `${items.length}-${sum}`
  }, [modelData])

  // Table data with pagination
  const tableData = useMemo(() => dailyData ?? [], [dailyData])
  const totalPages = Math.ceil(tableData.length / PAGE_SIZE)
  const paginatedData = useMemo(
    () =>
      tableData.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE),
    [tableData, currentPage]
  )

  const renderChart = (
    titleKey: string,
    spec: Record<string, unknown>,
    fingerprint: string,
    chartKey: string
  ) => (
    <div className='overflow-hidden rounded-lg border'>
      <div className='flex w-full items-center gap-2 border-b px-3 py-2 sm:px-5 sm:py-3'>
        <Hash className='text-muted-foreground/60 size-4' />
        <div className='text-sm font-semibold'>{t(titleKey)}</div>
      </div>
      <div className='h-[300px] p-1.5 sm:h-96 sm:p-2'>
        {isLoading ? (
          <Skeleton className='h-full w-full' />
        ) : (
          themeReady &&
          spec && (
            <VChart
              key={`subscription-${chartKey}-${resolvedTheme}-${compactMode}-${fingerprint}`}
              spec={{
                ...spec,
                theme: resolvedTheme === 'dark' ? 'dark' : 'light',
                background: 'transparent',
              }}
              option={VCHART_OPTION}
            />
          )
        )}
      </div>
    </div>
  )

  return (
    <div className='space-y-3'>
      {/* Filter controls */}
      <div className='flex items-center gap-1.5 overflow-x-auto pb-1 sm:gap-2'>
        <DashboardTimeRangeBar
          value={props.filters}
          onChange={handleTimeWindowChange}
        />

        {/* Model text input filter */}
        <div className='flex shrink-0 items-center gap-1.5'>
          <Input
            value={modelFilter}
            onChange={(e) => handleModelChange(e.target.value)}
            placeholder={t('Filter by model')}
            className='h-7 w-40 text-xs'
            aria-label={t('Model')}
          />
        </div>

        {/* Number format toggle */}
        <Button
          variant='outline'
          size='sm'
          className='h-7 shrink-0 gap-1 px-2 text-xs'
          onClick={() => setCompactMode(!compactMode)}
          title={t('Number Format')}
        >
          <ArrowLeftRight className='h-3 w-3' />
          {compactMode ? t('Compact') : t('Precise')}
        </Button>

        {isLoading && (
          <Loader2 className='text-muted-foreground size-4 animate-spin' />
        )}
      </div>

      {/* Personal subscription summary cards */}
      {activeSubscriptions.length > 0 && (
        <div className='grid gap-2 sm:grid-cols-2 lg:grid-cols-3'>
          {activeSubscriptions.map((s) => {
            const sub = s.subscription
            const amountPercent =
              sub.amount_total > 0
                ? Math.min(100, (sub.amount_used / sub.amount_total) * 100)
                : 0
            const tokensPercent =
              (sub.tokens_total ?? 0) > 0
                ? Math.min(
                    100,
                    ((sub.tokens_used ?? 0) / (sub.tokens_total ?? 0)) * 100
                  )
                : 0
            const endDate = new Date(sub.end_time * 1000)
            const daysLeft = Math.max(
              0,
              Math.ceil((sub.end_time * 1000 - Date.now()) / 86400000)
            )
            return (
              <div
                key={s.subscription.id}
                className='space-y-2 rounded-lg border p-3'
              >
                <div className='flex items-center justify-between'>
                  <span className='text-sm font-medium'>
                    {t('Plan')} #{sub.plan_id}
                  </span>
                  <span className='text-muted-foreground text-xs'>
                    {daysLeft} {t('days remaining')}
                  </span>
                </div>
                {sub.amount_total > 0 ? (
                  <div className='space-y-1'>
                    <div className='text-muted-foreground flex justify-between text-xs'>
                      <span>{t('Quota')}</span>
                      <span>
                        {formatQuotaWithCurrency(sub.amount_used)} /{' '}
                        {formatQuotaWithCurrency(sub.amount_total)}
                      </span>
                    </div>
                    <div className='bg-muted h-1.5 overflow-hidden rounded-full'>
                      <div
                        className='bg-primary h-full rounded-full'
                        style={{ width: `${amountPercent}%` }}
                      />
                    </div>
                  </div>
                ) : (
                  <div className='text-muted-foreground flex justify-between text-xs'>
                    <span>{t('Quota')}</span>
                    <span>{t('Unlimited')}</span>
                  </div>
                )}
                {(sub.tokens_total ?? 0) > 0 ? (
                  <div className='space-y-1'>
                    <div className='text-muted-foreground flex justify-between text-xs'>
                      <span>{t('Total Tokens')}</span>
                      <span>
                        {formatCompactNumber(sub.tokens_used ?? 0)} /{' '}
                        {formatCompactNumber(sub.tokens_total ?? 0)}
                      </span>
                    </div>
                    <div className='bg-muted h-1.5 overflow-hidden rounded-full'>
                      <div
                        className='h-full rounded-full bg-blue-500'
                        style={{ width: `${tokensPercent}%` }}
                      />
                    </div>
                  </div>
                ) : (
                  <div className='text-muted-foreground flex justify-between text-xs'>
                    <span>{t('Total Tokens')}</span>
                    <span>{t('Unlimited')}</span>
                  </div>
                )}
                {(sub.next_reset_time ?? 0) > 0 && (
                  <div className='text-muted-foreground text-xs'>
                    {t('Quota resets')}:{' '}
                    {new Date(
                      (sub.next_reset_time ?? 0) * 1000
                    ).toLocaleString()}
                  </div>
                )}
                {(sub.token_next_reset_time ?? 0) > 0 && (
                  <div className='text-muted-foreground text-xs'>
                    {t('Token resets')}:{' '}
                    {new Date(
                      (sub.token_next_reset_time ?? 0) * 1000
                    ).toLocaleString()}
                  </div>
                )}
                <div className='text-muted-foreground text-xs'>
                  {t('Expires')}: {endDate.toLocaleDateString()}
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* Error state */}
      {hasError && !isLoading && (
        <div className='border-destructive/30 bg-destructive/5 text-destructive flex items-center gap-2 rounded-lg border px-4 py-3 text-sm'>
          <AlertCircle className='size-4 shrink-0' />
          {t('Failed to load subscription usage data')}
        </div>
      )}

      {/* Charts */}
      <div className='grid gap-3'>
        {renderChart(
          'Daily Subscription Token Usage Trend',
          trendSpec,
          trendFingerprint,
          'trend'
        )}
        {renderChart(
          'Subscription Model Distribution',
          modelPieSpec,
          modelFingerprint,
          'pie'
        )}
      </div>

      {/* Data Table */}
      <div className='overflow-hidden rounded-lg border'>
        <div className='flex w-full items-center gap-2 border-b px-3 py-2 sm:px-5 sm:py-3'>
          <Hash className='text-muted-foreground/60 size-4' />
          <div className='text-sm font-semibold'>
            {t('Subscription Usage Details')}
          </div>
        </div>

        <div className='overflow-x-auto'>
          {isLoading && (
            <div className='p-4'>
              <Skeleton className='h-64 w-full' />
            </div>
          )}
          {!isLoading && tableData.length === 0 && (
            <div className='text-muted-foreground flex items-center justify-center p-8 text-sm'>
              {hasError
                ? t('Failed to load subscription usage data')
                : t('No subscription usage data')}
            </div>
          )}
          {!isLoading && tableData.length > 0 && (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className='whitespace-nowrap'>
                    {t('Date')}
                  </TableHead>
                  <TableHead className='whitespace-nowrap'>
                    {t('Plan')}
                  </TableHead>
                  <TableHead className='text-right whitespace-nowrap'>
                    {t('Prompt Tokens')}
                  </TableHead>
                  <TableHead className='text-right whitespace-nowrap'>
                    {t('Completion Tokens')}
                  </TableHead>
                  <TableHead className='text-right whitespace-nowrap'>
                    {t('Cached Tokens')}
                  </TableHead>
                  <TableHead className='text-right whitespace-nowrap'>
                    {t('Total Tokens')}
                  </TableHead>
                  <TableHead className='text-right whitespace-nowrap'>
                    {t('Requests')}
                  </TableHead>
                  <TableHead className='text-right whitespace-nowrap'>
                    {t('Quota')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {paginatedData.map((item: SubscriptionUsageDataItem) => (
                  <TableRow key={`${item.subscription_id}-${item.date}`}>
                    <TableCell className='whitespace-nowrap'>
                      {item.date}
                    </TableCell>
                    <TableCell className='font-medium whitespace-nowrap'>
                      {item.plan_title}
                    </TableCell>
                    <TableCell className='text-right whitespace-nowrap tabular-nums'>
                      {formatNum(item.prompt_tokens)}
                    </TableCell>
                    <TableCell className='text-right whitespace-nowrap tabular-nums'>
                      {formatNum(item.completion_tokens)}
                    </TableCell>
                    <TableCell className='text-right whitespace-nowrap tabular-nums'>
                      {formatNum(item.cached_tokens)}
                    </TableCell>
                    <TableCell className='text-right font-medium whitespace-nowrap tabular-nums'>
                      {formatNum(item.total_tokens)}
                    </TableCell>
                    <TableCell className='text-right whitespace-nowrap tabular-nums'>
                      {formatNum(item.request_count)}
                    </TableCell>
                    <TableCell className='text-right whitespace-nowrap tabular-nums'>
                      {formatQuota(item.quota)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className='flex items-center justify-between border-t px-3 py-2 sm:px-5'>
            <div className='text-muted-foreground text-xs'>
              {t('{{total}} records', { total: tableData.length })}
            </div>
            <div className='flex items-center gap-1'>
              <Tabs
                value={String(currentPage)}
                onValueChange={(v) => setCurrentPage(Number(v))}
              >
                <TabsList>
                  {Array.from({ length: totalPages }, (_, i) => i + 1).map(
                    (page) => (
                      <TabsTrigger
                        key={page}
                        value={String(page)}
                        className='px-2.5 text-xs'
                      >
                        {page}
                      </TabsTrigger>
                    )
                  )}
                </TabsList>
              </Tabs>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
