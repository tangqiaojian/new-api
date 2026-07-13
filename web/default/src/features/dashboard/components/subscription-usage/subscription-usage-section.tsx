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
import { useEffect, useMemo, useState, useRef, useCallback } from 'react'
import { useQuery } from '@tanstack/react-query'
import { VChart } from '@visactor/react-vchart'
import { Hash, Loader2, ArrowLeftRight, AlertCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toIntlLocale } from '@/i18n/languages'
import { getRollingDateRange } from '@/lib/time'
import { useAutoRefresh } from '@/features/dashboard/hooks/use-auto-refresh'
import { VCHART_OPTION } from '@/lib/vchart'
import { useTheme } from '@/context/theme-provider'
import { formatQuotaWithCurrency } from '@/lib/currency'
import { formatCompactNumber, formatNumber } from '@/lib/format'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  getSelfSubscriptionUsage,
  getSelfSubscriptionModelUsage,
} from '@/features/dashboard/api'
import { TIME_RANGE_PRESETS } from '@/features/dashboard/constants'
import type {
  SubscriptionUsageDataItem,
  SubscriptionUsageFilters,
} from '@/features/dashboard/types'

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
  const { t, i18n } = useTranslation()
  const { resolvedTheme } = useTheme()
  const { refetchInterval } = useAutoRefresh()
  const [themeReady, setThemeReady] = useState(false)
  const themeManagerRef = useRef<
    (typeof import('@visactor/vchart'))['ThemeManager'] | null
  >(null)

  // Number format mode: 'compact' shows 万/亿, 'precise' shows full numbers
  const [compactMode, setCompactMode] = useState(true)
  const includeCache = props.includeCache ?? false

  const selectedRange = props.filters.selectedRange
  const modelFilter = props.filters.model
  const onFiltersChange = props.onFiltersChange
  const [currentPage, setCurrentPage] = useState(1)

  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)

  const timeRange = useMemo(() => {
    const { start, end } = getRollingDateRange(selectedRange)
    return {
      start_timestamp: Math.floor(start.getTime() / 1000),
      end_timestamp: Math.floor(end.getTime() / 1000),
    }
  }, [selectedRange])

  const handleRangeChange = useCallback(
    (days: number) => {
      onFiltersChange({ ...props.filters, selectedRange: days })
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
    const sortedItems = [...items].sort((a, b) =>
      a.date.localeCompare(b.date)
    )

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
        const tokens =
          s === 'prompt'
            ? item.prompt_tokens
            : s === 'completion'
              ? item.completion_tokens
              : item.cached_tokens
        values.push({
          Date: item.date,
          Series: seriesLabels[s],
          Tokens: Number(tokens) || 0,
        })
      })
    })

    const totalTokens = sortedItems.reduce(
      (sum, item) => sum + (Number(item.total_tokens) || 0),
      0
    )

    return {
      type: 'area',
      data: [{ id: 'subscriptionTrendData', values }],
      xField: 'Date',
      yField: 'Tokens',
      seriesField: 'Series',
      stack: false,
      title: {
        visible: true,
        text: t('Daily Subscription Token Usage Trend'),
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
      area: {
        style: {
          fillOpacity: 0.15,
          curveType: 'monotone',
        },
      },
      line: {
        style: {
          lineWidth: 2,
          curveType: 'monotone',
        },
      },
      point: { visible: false },
      color: { specified: colorMap },
      background: { fill: 'transparent' },
      animation: true,
    }
  }, [dailyData, isLoading, includeCache, t, formatInt])

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
        text: t('Subscription Model Distribution'),
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
  }, [modelData, isLoading, t, formatInt])

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
        <Tabs
          value={String(selectedRange)}
          onValueChange={(value) => handleRangeChange(Number(value))}
          className='shrink-0'
        >
          <TabsList>
            {TIME_RANGE_PRESETS.map((preset) => (
              <TabsTrigger
                key={preset.days}
                value={String(preset.days)}
                className='px-2.5 text-xs'
              >
                {t(preset.label)}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>

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

        {/* Cache token toggle (controlled by parent DashboardAutoRefreshControls) */}
        <div className='flex shrink-0 items-center gap-1.5'>
          <Switch
            id='subscription-usage-include-cache'
            checked={includeCache}
            disabled
            className='scale-90'
          />
          <Label
            htmlFor='subscription-usage-include-cache'
            className='text-muted-foreground cursor-pointer text-xs font-normal'
          >
            {t('Include cache')}
          </Label>
        </div>

        {/* Number format toggle */}
        <Button
          variant='outline'
          size='sm'
          className='shrink-0 h-7 px-2 text-xs gap-1'
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

      {/* Error state */}
      {hasError && !isLoading && (
        <div className='flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive'>
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
          {isLoading ? (
            <div className='p-4'>
              <Skeleton className='h-64 w-full' />
            </div>
          ) : tableData.length === 0 ? (
            <div className='text-muted-foreground flex items-center justify-center p-8 text-sm'>
              {hasError
                ? t('Failed to load subscription usage data')
                : t('No subscription usage data')}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className='whitespace-nowrap'>
                    {t('Date')}
                  </TableHead>
                  <TableHead className='whitespace-nowrap'>
                    {t('Plan')}
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    {t('Prompt Tokens')}
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    {t('Completion Tokens')}
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    {t('Cached Tokens')}
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    {t('Total Tokens')}
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    {t('Requests')}
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    {t('Quota')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {paginatedData.map(
                  (item: SubscriptionUsageDataItem, idx: number) => (
                    <TableRow
                      key={`${item.subscription_id}-${item.date}-${idx}`}
                    >
                      <TableCell className='whitespace-nowrap'>
                        {item.date}
                      </TableCell>
                      <TableCell className='whitespace-nowrap font-medium'>
                        {item.plan_title}
                      </TableCell>
                      <TableCell className='whitespace-nowrap text-right tabular-nums'>
                        {formatNum(item.prompt_tokens)}
                      </TableCell>
                      <TableCell className='whitespace-nowrap text-right tabular-nums'>
                        {formatNum(item.completion_tokens)}
                      </TableCell>
                      <TableCell className='whitespace-nowrap text-right tabular-nums'>
                        {formatNum(item.cached_tokens)}
                      </TableCell>
                      <TableCell className='whitespace-nowrap text-right tabular-nums font-medium'>
                        {formatNum(item.total_tokens)}
                      </TableCell>
                      <TableCell className='whitespace-nowrap text-right tabular-nums'>
                        {formatNum(item.request_count)}
                      </TableCell>
                      <TableCell className='whitespace-nowrap text-right tabular-nums'>
                        {formatQuota(item.quota)}
                      </TableCell>
                    </TableRow>
                  )
                )}
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
