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
import { OpsStatsGrid } from '@/features/dashboard/components/ops-stats-grid'
import { DashboardTimeRangeBar } from '@/features/dashboard/components/ui/dashboard-time-range-bar'
import { TIME_GRANULARITY_OPTIONS } from '@/features/dashboard/constants'
import { useAutoRefresh } from '@/features/dashboard/hooks/use-auto-refresh'
import { useOpsDashboardStats } from '@/features/dashboard/hooks/use-ops-dashboard-stats'
import {
  buildTimeWindow,
  formatTokens,
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
import { formatQuota } from '@/lib/format'
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

  const selectedUsername = props.filters.selectedUsername ?? null
  const setSelectedUsername = useCallback(
    (username: string | null) => {
      onFiltersChange({ ...props.filters, selectedUsername: username })
    },
    [onFiltersChange, props.filters]
  )

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
  }, [selectedUsername, setSelectedUsername, usernames])

  const scopedUserData = useMemo(() => {
    const data = userData ?? []
    if (!selectedUsername) return data
    return data.filter((item) => item.username === selectedUsername)
  }, [selectedUsername, userData])

  const chartData = useMemo(
    () =>
      processUserChartData(
        isLoading ? [] : scopedUserData,
        timeGranularity,
        t,
        selectedUsername ? 1 : topUserLimit
      ),
    [
      scopedUserData,
      isLoading,
      timeGranularity,
      t,
      topUserLimit,
      selectedUsername,
    ]
  )
  const { stats: opsStats, loading: opsLoading } = useOpsDashboardStats({
    username: selectedUsername ?? undefined,
    refetchInterval: refetchInterval || false,
  })

  const modelBreakdown = useMemo(() => {
    const byModel = new Map<
      string,
      {
        modelName: string
        count: number
        promptTokens: number
        completionTokens: number
        cacheReadTokens: number
        quota: number
      }
    >()
    for (const item of scopedUserData) {
      const modelName = (item.model_name || '').trim() || 'unknown'
      const prev = byModel.get(modelName) || {
        modelName,
        count: 0,
        promptTokens: 0,
        completionTokens: 0,
        cacheReadTokens: 0,
        quota: 0,
      }
      prev.count += Number(item.count) || 0
      prev.promptTokens += Number(item.prompt_tokens) || 0
      prev.completionTokens += Number(item.completion_tokens) || 0
      prev.cacheReadTokens += Number(item.cache_read_tokens) || 0
      prev.quota += Number(item.quota) || 0
      byModel.set(modelName, prev)
    }
    return [...byModel.values()].sort(
      (a, b) => b.quota - a.quota || b.count - a.count
    )
  }, [scopedUserData])

  const dataFingerprint = useMemo(() => {
    let quotaSum = 0
    for (const item of scopedUserData) quotaSum += item.quota ?? 0
    return `${scopedUserData.length}-${quotaSum}-${selectedUsername ?? 'all'}`
  }, [scopedUserData, selectedUsername])

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

      <OpsStatsGrid loading={isLoading || opsLoading} stats={opsStats} />

      {!selectedUsername && usernames.length > 0 ? (
        <div className='overflow-hidden rounded-lg border'>
          <div className='border-b px-3 py-2 text-sm font-semibold sm:px-5'>
            {t('User Consumption Ranking')}
          </div>
          <div className='divide-y'>
            {[...usernames]
              .map((name) => {
                const rows = (userData ?? []).filter((i) => i.username === name)
                const quota = rows.reduce(
                  (sum, row) => sum + (Number(row.quota) || 0),
                  0
                )
                return { name, quota }
              })
              .sort((a, b) => b.quota - a.quota)
              .slice(0, topUserLimit)
              .map((row, index) => (
                <button
                  key={row.name}
                  type='button'
                  className='hover:bg-muted/40 flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm sm:px-5'
                  onClick={() => setSelectedUsername(row.name)}
                >
                  <span className='truncate'>
                    <span className='text-muted-foreground mr-2 tabular-nums'>
                      #{index + 1}
                    </span>
                    {row.name}
                  </span>
                  <span className='shrink-0 tabular-nums'>
                    {formatQuota(row.quota)}
                  </span>
                </button>
              ))}
          </div>
        </div>
      ) : null}

      {selectedUsername && modelBreakdown.length > 0 ? (
        <div className='overflow-hidden rounded-lg border'>
          <div className='border-b px-3 py-2 text-sm font-semibold sm:px-5'>
            {t('Model breakdown')}
          </div>
          <div className='overflow-x-auto'>
            <table className='w-full text-left text-sm'>
              <thead className='bg-muted/40 text-muted-foreground'>
                <tr>
                  <th className='px-3 py-2 font-medium'>{t('Model')}</th>
                  <th className='px-3 py-2 font-medium'>{t('Requests')}</th>
                  <th className='px-3 py-2 font-medium'>{t('Input TOKEN')}</th>
                  <th className='px-3 py-2 font-medium'>{t('Output TOKEN')}</th>
                  <th className='px-3 py-2 font-medium'>{t('Cache TOKEN')}</th>
                  <th className='px-3 py-2 font-medium'>{t('Quota')}</th>
                </tr>
              </thead>
              <tbody>
                {modelBreakdown.map((row) => (
                  <tr key={row.modelName} className='border-t'>
                    <td className='px-3 py-2 font-medium'>{row.modelName}</td>
                    <td className='px-3 py-2 tabular-nums'>
                      {formatTokens(row.count)}
                    </td>
                    <td className='px-3 py-2 tabular-nums'>
                      {formatTokens(
                        row.promptTokens + row.cacheReadTokens
                      )}
                    </td>
                    <td className='px-3 py-2 tabular-nums'>
                      {formatTokens(row.completionTokens)}
                    </td>
                    <td className='px-3 py-2 tabular-nums'>
                      {formatTokens(row.cacheReadTokens)}
                    </td>
                    <td className='px-3 py-2 tabular-nums'>
                      {formatQuota(row.quota)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : null}

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
