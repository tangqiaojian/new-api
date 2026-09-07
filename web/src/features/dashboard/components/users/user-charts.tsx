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
import { Users, Loader2, X } from 'lucide-react'
import { useEffect, useMemo, useState, useRef, useCallback } from 'react'
import { useTranslation } from 'react-i18next'

import { IconBadge } from '@/components/ui/icon-badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useTheme } from '@/context/theme-provider'
import { getUserQuotaDataByUsers } from '@/features/dashboard/api'
import { UsageKpiGrid } from '@/features/dashboard/components/usage-kpi-grid'
import { DashboardTimeRangeBar } from '@/features/dashboard/components/ui/dashboard-time-range-bar'
import { TIME_GRANULARITY_OPTIONS } from '@/features/dashboard/constants'
import { useAutoRefresh } from '@/features/dashboard/hooks/use-auto-refresh'
import {
  buildTimeWindow,
  calculateDashboardStats,
  getDefaultDays,
  processUserChartData,
  resolveUnixTimeRange,
  saveGranularity,
} from '@/features/dashboard/lib'
import type {
  DashboardTimeWindow,
  ProcessedUserChartData,
  UserChartsFilters,
} from '@/features/dashboard/types'
import type { TimeGranularity } from '@/lib/time'
import { VCHART_OPTION } from '@/lib/vchart'

let themeManagerPromise: Promise<
  (typeof import('@visactor/vchart'))['ThemeManager']
> | null = null

const USER_CHARTS: {
  value: string
  labelKey: string
  specKey: keyof ProcessedUserChartData
}[] = [
  {
    value: 'rank',
    labelKey: 'User Consumption Ranking',
    specKey: 'spec_user_rank',
  },
  {
    value: 'trend',
    labelKey: 'User Consumption Trend',
    specKey: 'spec_user_trend',
  },
]

const TOP_USER_LIMIT_OPTIONS = [5, 10, 20, 50]

interface UserChartsProps {
  filters: UserChartsFilters
  onFiltersChange: (filters: UserChartsFilters) => void
}

export function UserCharts(props: UserChartsProps) {
  const { t } = useTranslation()
  const { resolvedTheme } = useTheme()
  const { refetchInterval } = useAutoRefresh()
  const [themeReady, setThemeReady] = useState(false)
  const themeManagerRef = useRef<
    (typeof import('@visactor/vchart'))['ThemeManager'] | null
  >(null)

  // The selection is owned by the dashboard parent so it persists across
  // sub-section switches; the rolling window is derived from the chosen range.
  const timeGranularity = props.filters.timeGranularity
  const selectedRange = props.filters.selectedRange
  const startTimestamp = props.filters.start_timestamp
  const endTimestamp = props.filters.end_timestamp
  const topUserLimit = props.filters.topUserLimit
  const onFiltersChange = props.onFiltersChange

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

  const handleGranularityChange = useCallback(
    (g: TimeGranularity) => {
      saveGranularity(g)
      onFiltersChange({
        ...props.filters,
        timeGranularity: g,
        ...buildTimeWindow(getDefaultDays(g)),
      })
    },
    [onFiltersChange, props.filters]
  )

  const handleTopUserLimitChange = useCallback(
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

  const [selectedUsername, setSelectedUsername] = useState<string | null>(null)

  const { data: userData, isLoading } = useQuery({
    queryKey: ['dashboard', 'user-quota', timeRange],
    queryFn: () => getUserQuotaDataByUsers(timeRange),
    select: (res) => (res.success ? res.data : []),
    staleTime: 60_000,
    refetchInterval: refetchInterval || undefined,
  })

  const usernames = useMemo(() => {
    const names = new Set<string>()
    for (const item of userData ?? []) {
      const name = (item.username || '').trim()
      if (name) names.add(name)
    }
    return [...names].sort((a, b) => a.localeCompare(b))
  }, [userData])

  useEffect(() => {
    if (selectedUsername && !usernames.includes(selectedUsername)) {
      setSelectedUsername(null)
    }
  }, [selectedUsername, usernames])

  const chartData = useMemo(
    () =>
      processUserChartData(
        isLoading ? [] : (userData ?? []),
        timeGranularity,
        t,
        topUserLimit
      ),
    [userData, isLoading, timeGranularity, t, topUserLimit]
  )
  const kpiStats = useMemo(() => {
    const data = userData ?? []
    const scoped = selectedUsername
      ? data.filter((item) => item.username === selectedUsername)
      : data
    if (scoped.length === 0) return null
    const stats = calculateDashboardStats(scoped)
    return {
      totalCount: stats.totalCount,
      promptTokens: stats.promptTokens,
      completionTokens: stats.completionTokens,
      cacheReadTokens: stats.cacheReadTokens,
      successCount: stats.successCount,
    }
  }, [selectedUsername, userData])
  const dataFingerprint = useMemo(() => {
    const items = userData ?? []
    let quotaSum = 0
    for (const item of items) quotaSum += item.quota ?? 0
    return `${items.length}-${quotaSum}-${selectedUsername ?? 'all'}`
  }, [selectedUsername, userData])

  return (
    <div className='space-y-3'>
      <div className='flex flex-wrap items-center gap-2'>
        <div className='text-sm font-medium'>
          {selectedUsername
            ? t('User detail: {{username}}', { username: selectedUsername })
            : t('All users')}
        </div>
        <Select
          items={[
            { value: '__all__', label: t('All users') },
            ...usernames.map((name) => ({ value: name, label: name })),
          ]}
          value={selectedUsername ?? '__all__'}
          onValueChange={(value) =>
            setSelectedUsername(
              !value || value === '__all__' ? null : String(value)
            )
          }
        >
          <SelectTrigger className='w-[200px]' aria-label={t('Select user')}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value='__all__'>{t('All users')}</SelectItem>
            {usernames.map((name) => (
              <SelectItem key={name} value={name}>
                {name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {selectedUsername ? (
          <Button
            type='button'
            variant='ghost'
            size='sm'
            onClick={() => setSelectedUsername(null)}
          >
            <X className='mr-1 size-4' />
            {t('Clear')}
          </Button>
        ) : null}
      </div>

      <UsageKpiGrid loading={isLoading} stats={kpiStats} />

      {!isLoading && (userData?.length ?? 0) === 0 ? (
        <div className='text-muted-foreground rounded-lg border border-dashed px-4 py-10 text-center text-sm'>
          {t('No usage data in the selected time range')}
        </div>
      ) : null}

      <div className='flex items-center gap-1.5 overflow-x-auto pb-1 sm:gap-2'>
        <DashboardTimeRangeBar
          value={props.filters}
          onChange={handleTimeWindowChange}
        />

        <Tabs
          value={timeGranularity}
          onValueChange={(value) =>
            handleGranularityChange(value as TimeGranularity)
          }
          className='shrink-0'
        >
          <TabsList>
            {TIME_GRANULARITY_OPTIONS.map((opt) => (
              <TabsTrigger
                key={opt.value}
                value={opt.value}
                className='px-2.5 text-xs'
              >
                {t(opt.label)}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>

        <Tabs
          value={String(topUserLimit)}
          onValueChange={(value) => handleTopUserLimitChange(Number(value))}
          className='shrink-0'
        >
          <TabsList>
            <span className='text-muted-foreground px-2 text-xs font-medium whitespace-nowrap'>
              {t('Top Users')}
            </span>
            {TOP_USER_LIMIT_OPTIONS.map((limit) => (
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

        {isLoading && (
          <Loader2 className='text-muted-foreground size-4 animate-spin' />
        )}
      </div>

      <div className='grid gap-3'>
        {USER_CHARTS.map((chart) => {
          const spec = chartData[chart.specKey]

          return (
            <div
              key={chart.value}
              className='overflow-hidden rounded-lg border'
            >
              <div className='flex w-full items-center gap-2 border-b px-3 py-2 sm:px-5 sm:py-3'>
                <IconBadge tone='info' size='sm'>
                  <Users />
                </IconBadge>
                <div className='text-sm font-semibold'>{t(chart.labelKey)}</div>
              </div>

              <div className='h-[300px] p-1.5 sm:h-96 sm:p-2'>
                {isLoading ? (
                  <Skeleton className='h-full w-full' />
                ) : (
                  themeReady &&
                  spec && (
                    <VChart
                      key={`user-${chart.value}-${topUserLimit}-${resolvedTheme}-${dataFingerprint}`}
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
    </div>
  )
}
