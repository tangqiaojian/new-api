import { useQuery } from '@tanstack/react-query'
import { VChart } from '@visactor/react-vchart'
import { Hash, Loader2, ArrowLeftRight } from 'lucide-react'
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
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
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
  getDailyModelTokenData,
  getSelfDailyModelTokenData,
} from '@/features/dashboard/api'
import { DashboardTimeRangeBar } from '@/features/dashboard/components/ui/dashboard-time-range-bar'
import { useAutoRefresh } from '@/features/dashboard/hooks/use-auto-refresh'
import {
  processDailyModelTokensChartData,
  resolveUnixTimeRange,
} from '@/features/dashboard/lib'
import type {
  DailyTokensFilters,
  DashboardTimeWindow,
  ProcessedDailyModelTokensChartData,
  TokenMetricType,
} from '@/features/dashboard/types'
import { toIntlLocale } from '@/i18n/languages'
import { formatQuotaWithCurrency } from '@/lib/currency'
import { formatCompactNumber, formatNumber } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { VCHART_OPTION } from '@/lib/vchart'
import { useAuthStore } from '@/stores/auth-store'

let themeManagerPromise: Promise<
  (typeof import('@visactor/vchart'))['ThemeManager']
> | null = null

const MODEL_CHARTS: {
  value: string
  labelKey: string
  specKey: keyof ProcessedDailyModelTokensChartData
}[] = [
  {
    value: 'trend',
    labelKey: 'Daily Model Token Usage Trend',
    specKey: 'spec_model_trend',
  },
  {
    value: 'rank',
    labelKey: 'Model Token Ranking',
    specKey: 'spec_model_rank',
  },
  {
    value: 'request-count',
    labelKey: 'Model Request Count Ranking',
    specKey: 'spec_model_request_count',
  },
  {
    value: 'pie',
    labelKey: 'Model Token Distribution',
    specKey: 'spec_model_pie',
  },
]

const TOP_MODEL_LIMIT_OPTIONS = [5, 10, 20, 50]

const TOKEN_METRIC_OPTIONS: {
  value: TokenMetricType
  labelKey: string
}[] = [
  { value: 'total', labelKey: 'Total Tokens' },
  { value: 'prompt', labelKey: 'Prompt Tokens' },
  { value: 'completion', labelKey: 'Completion Tokens' },
]

const PAGE_SIZE = 20

interface DailyModelTokensSectionProps {
  filters: DailyTokensFilters
  onFiltersChange: (filters: DailyTokensFilters) => void
  includeCache?: boolean
}

export function DailyModelTokensSection(props: DailyModelTokensSectionProps) {
  const { t, i18n } = useTranslation()
  const { resolvedTheme } = useTheme()
  const { refetchInterval } = useAutoRefresh()
  const [themeReady, setThemeReady] = useState(false)
  const themeManagerRef = useRef<
    (typeof import('@visactor/vchart'))['ThemeManager'] | null
  >(null)
  const userRole = useAuthStore((state) => state.auth.user?.role)
  const isAdmin = Boolean(userRole && userRole >= ROLE.ADMIN)

  const [compactMode, setCompactMode] = useState(true)
  const selectedRange = props.filters.selectedRange
  const startTimestamp = props.filters.start_timestamp
  const endTimestamp = props.filters.end_timestamp
  const topUserLimit = props.filters.topUserLimit
  const onFiltersChange = props.onFiltersChange
  const [metricType, setMetricType] = useState<TokenMetricType>('total')
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
    },
    [onFiltersChange, props.filters]
  )

  const handleTopModelLimitChange = useCallback(
    (limit: number) => {
      onFiltersChange({ ...props.filters, topUserLimit: limit })
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

  const { data: dailyModelTokenData, isLoading } = useQuery({
    queryKey: [
      'dashboard',
      'daily-model-tokens',
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

  const chartData = useMemo(
    () =>
      processDailyModelTokensChartData(
        isLoading ? [] : (dailyModelTokenData ?? []),
        t,
        metricType,
        topUserLimit,
        compactMode,
        locale
      ),
    [
      dailyModelTokenData,
      isLoading,
      t,
      metricType,
      topUserLimit,
      compactMode,
      locale,
    ]
  )

  // Data fingerprint so VChart remounts when the underlying data changes
  // (react-vchart does not reliably re-render on spec prop changes alone).
  const dataFingerprint = useMemo(() => {
    const items = dailyModelTokenData ?? []
    let sum = 0
    for (const item of items) sum += item.total_tokens
    return `${items.length}-${sum}`
  }, [dailyModelTokenData])

  const tableData = useMemo(
    () => dailyModelTokenData ?? [],
    [dailyModelTokenData]
  )
  const totalPages = Math.ceil(tableData.length / PAGE_SIZE)
  const paginatedData = useMemo(
    () =>
      tableData.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE),
    [tableData, currentPage]
  )

  const formatQuota = (quota: number) => formatQuotaWithCurrency(quota)
  const formatNum = (value: number) =>
    compactMode ? formatCompactNumber(value, locale) : formatNumber(value)

  return (
    <div className='space-y-3'>
      {/* Filter controls */}
      <div className='flex items-center gap-1.5 overflow-x-auto pb-1 sm:gap-2'>
        <DashboardTimeRangeBar
          value={props.filters}
          onChange={handleTimeWindowChange}
        />

        <Tabs
          value={metricType}
          onValueChange={(value) => setMetricType(value as TokenMetricType)}
          className='shrink-0'
        >
          <TabsList>
            {TOKEN_METRIC_OPTIONS.map((opt) => (
              <TabsTrigger
                key={opt.value}
                value={opt.value}
                className='px-2.5 text-xs'
              >
                {t(opt.labelKey)}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>

        <Tabs
          value={String(topUserLimit)}
          onValueChange={(value) => handleTopModelLimitChange(Number(value))}
          className='shrink-0'
        >
          <TabsList>
            <span className='text-muted-foreground px-2 text-xs font-medium whitespace-nowrap'>
              {t('Top Users')}
            </span>
            {TOP_MODEL_LIMIT_OPTIONS.map((limit) => (
              <TabsTrigger
                key={limit}
                value={String(limit)}
                className='px-2.5 text-xs'
              >
                {t('Top {{count}}', { count: limit })}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>

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

      {/* Charts */}
      <div className='grid gap-3'>
        {MODEL_CHARTS.map((chart) => {
          const spec = chartData[chart.specKey]
          return (
            <div
              key={chart.value}
              className='overflow-hidden rounded-lg border'
            >
              <div className='flex w-full items-center gap-2 border-b px-3 py-2 sm:px-5 sm:py-3'>
                <Hash className='text-muted-foreground/60 size-4' />
                <div className='text-sm font-semibold'>{t(chart.labelKey)}</div>
              </div>
              <div className='h-[300px] p-1.5 sm:h-96 sm:p-2'>
                {isLoading ? (
                  <Skeleton className='h-full w-full' />
                ) : (
                  themeReady &&
                  spec && (
                    <VChart
                      key={`daily-model-tokens-${chart.value}-${topUserLimit}-${metricType}-${resolvedTheme}-${compactMode}-${dataFingerprint}`}
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
        })}
      </div>

      {/* Data Table */}
      <div className='overflow-hidden rounded-lg border'>
        <div className='flex w-full items-center gap-2 border-b px-3 py-2 sm:px-5 sm:py-3'>
          <Hash className='text-muted-foreground/60 size-4' />
          <div className='text-sm font-semibold'>
            {t('Token Usage per Model per Day')}
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
              {t('No data available')}
            </div>
          )}
          {!isLoading && tableData.length > 0 && (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className='whitespace-nowrap'>
                    {t('Model')}
                  </TableHead>
                  <TableHead className='whitespace-nowrap'>
                    {t('Date')}
                  </TableHead>
                  <TableHead className='text-right whitespace-nowrap'>
                    {t('Prompt Tokens')}
                  </TableHead>
                  <TableHead className='text-right whitespace-nowrap'>
                    {t('Completion Tokens')}
                  </TableHead>
                  <TableHead className='text-right whitespace-nowrap'>
                    {t('Total Tokens')}
                  </TableHead>
                  <TableHead className='text-right whitespace-nowrap'>
                    {t('Request Count')}
                  </TableHead>
                  <TableHead className='text-right whitespace-nowrap'>
                    {t('Quota')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {paginatedData.map((item) => (
                  <TableRow key={`${item.model_name}-${item.date}`}>
                    <TableCell className='font-medium whitespace-nowrap'>
                      {item.model_name}
                    </TableCell>
                    <TableCell className='whitespace-nowrap'>
                      {item.date}
                    </TableCell>
                    <TableCell className='text-right whitespace-nowrap tabular-nums'>
                      {formatNum(item.prompt_tokens)}
                    </TableCell>
                    <TableCell className='text-right whitespace-nowrap tabular-nums'>
                      {formatNum(item.completion_tokens)}
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
