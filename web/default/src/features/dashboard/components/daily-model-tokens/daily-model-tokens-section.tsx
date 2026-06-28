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
import { Hash, Loader2, ArrowLeftRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { getRollingDateRange } from '@/lib/time'
import { useAutoRefresh } from '@/features/dashboard/hooks/use-auto-refresh'
import { VCHART_OPTION } from '@/lib/vchart'
import { useTheme } from '@/context/theme-provider'
import { useAuthStore } from '@/stores/auth-store'
import { ROLE } from '@/lib/roles'
import { formatQuotaWithCurrency } from '@/lib/currency'
import { formatCompactNumber, formatNumber } from '@/lib/format'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  getDailyModelTokenData,
  getSelfDailyModelTokenData,
} from '@/features/dashboard/api'
import { TIME_RANGE_PRESETS } from '@/features/dashboard/constants'
import { processDailyModelTokensChartData } from '@/features/dashboard/lib'
import type {
  DailyTokensFilters,
  ProcessedDailyModelTokensChartData,
  TokenMetricType,
} from '@/features/dashboard/types'

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
  const topUserLimit = props.filters.topUserLimit
  const onFiltersChange = props.onFiltersChange
  const [metricType, setMetricType] = useState<TokenMetricType>('total')
  const [currentPage, setCurrentPage] = useState(1)

  const locale = i18n.resolvedLanguage || 'en'

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
    queryKey: ['dashboard', 'daily-model-tokens', timeRange, isAdmin],
    queryFn: () =>
      isAdmin
        ? getDailyModelTokenData(timeRange)
        : getSelfDailyModelTokenData(timeRange),
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
    [dailyModelTokenData, isLoading, t, metricType, topUserLimit, compactMode, locale]
  )

  const tableData = useMemo(() => dailyModelTokenData ?? [], [dailyModelTokenData])
  const totalPages = Math.ceil(tableData.length / PAGE_SIZE)
  const paginatedData = useMemo(
    () => tableData.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE),
    [tableData, currentPage]
  )

  const formatQuota = (quota: number) => formatQuotaWithCurrency(quota)
  const formatNum = (value: number) =>
    compactMode ? formatCompactNumber(value, locale) : formatNumber(value)

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
              <TabsTrigger key={preset.days} value={String(preset.days)} className='px-2.5 text-xs'>
                {t(preset.label)}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>

        <Tabs
          value={metricType}
          onValueChange={(value) => setMetricType(value as TokenMetricType)}
          className='shrink-0'
        >
          <TabsList>
            {TOKEN_METRIC_OPTIONS.map((opt) => (
              <TabsTrigger key={opt.value} value={opt.value} className='px-2.5 text-xs'>
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
              <TabsTrigger key={limit} value={String(limit)} className='px-2.5 text-xs'>
                {t('Top {{count}}', { count: limit })}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>

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

        {isLoading && <Loader2 className='text-muted-foreground size-4 animate-spin' />}
      </div>

      {/* Charts */}
      <div className='grid gap-3'>
        {MODEL_CHARTS.map((chart) => {
          const spec = chartData[chart.specKey]
          return (
            <div key={chart.value} className='overflow-hidden rounded-lg border'>
              <div className='flex w-full items-center gap-2 border-b px-3 py-2 sm:px-5 sm:py-3'>
                <Hash className='text-muted-foreground/60 size-4' />
                <div className='text-sm font-semibold'>{t(chart.labelKey)}</div>
              </div>
              <div className='h-[300px] p-1.5 sm:h-96 sm:p-2'>
                {isLoading ? (
                  <Skeleton className='h-full w-full' />
                ) : (
                  themeReady && spec && (
                    <VChart
                      key={`daily-model-tokens-${chart.value}-${topUserLimit}-${metricType}-${resolvedTheme}-${compactMode}`}
                      spec={{ ...spec, theme: resolvedTheme === 'dark' ? 'dark' : 'light', background: 'transparent' }}
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
          <div className='text-sm font-semibold'>{t('Token Usage per Model per Day')}</div>
        </div>
        <div className='overflow-x-auto'>
          {isLoading ? (
            <div className='p-4'><Skeleton className='h-64 w-full' /></div>
          ) : tableData.length === 0 ? (
            <div className='text-muted-foreground flex items-center justify-center p-8 text-sm'>
              {t('No data available')}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className='whitespace-nowrap'>{t('Model')}</TableHead>
                  <TableHead className='whitespace-nowrap'>{t('Date')}</TableHead>
                  <TableHead className='whitespace-nowrap text-right'>{t('Prompt Tokens')}</TableHead>
                  <TableHead className='whitespace-nowrap text-right'>{t('Completion Tokens')}</TableHead>
                  <TableHead className='whitespace-nowrap text-right'>{t('Total Tokens')}</TableHead>
                  <TableHead className='whitespace-nowrap text-right'>{t('Request Count')}</TableHead>
                  <TableHead className='whitespace-nowrap text-right'>{t('Quota')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {paginatedData.map((item, idx) => (
                  <TableRow key={`${item.model_name}-${item.date}-${idx}`}>
                    <TableCell className='whitespace-nowrap font-medium'>{item.model_name}</TableCell>
                    <TableCell className='whitespace-nowrap'>{item.date}</TableCell>
                    <TableCell className='whitespace-nowrap text-right tabular-nums'>{formatNum(item.prompt_tokens)}</TableCell>
                    <TableCell className='whitespace-nowrap text-right tabular-nums'>{formatNum(item.completion_tokens)}</TableCell>
                    <TableCell className='whitespace-nowrap text-right tabular-nums font-medium'>{formatNum(item.total_tokens)}</TableCell>
                    <TableCell className='whitespace-nowrap text-right tabular-nums'>{formatNum(item.request_count)}</TableCell>
                    <TableCell className='whitespace-nowrap text-right tabular-nums'>{formatQuota(item.quota)}</TableCell>
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
              <Tabs value={String(currentPage)} onValueChange={(v) => setCurrentPage(Number(v))}>
                <TabsList>
                  {Array.from({ length: totalPages }, (_, i) => i + 1).map((page) => (
                    <TabsTrigger key={page} value={String(page)} className='px-2.5 text-xs'>{page}</TabsTrigger>
                  ))}
                </TabsList>
              </Tabs>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
