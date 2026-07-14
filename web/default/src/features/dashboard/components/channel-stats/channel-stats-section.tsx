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
import { useMemo, useState, useCallback } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  ArrowLeftRight,
  ArrowUpDown,
  Hash,
  Loader2,
  Search,
  X,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
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
import {
  getChannelModelStats,
  getSelfChannelModelStats,
} from '@/features/dashboard/api'
import { TIME_RANGE_PRESETS } from '@/features/dashboard/constants'
import { useAutoRefresh } from '@/features/dashboard/hooks/use-auto-refresh'
import type {
  ChannelModelStatsItem,
  ChannelStatsFilters,
} from '@/features/dashboard/types'
import { toIntlLocale } from '@/i18n/languages'
import { formatQuotaWithCurrency } from '@/lib/currency'
import { formatCompactNumber, formatNumber } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { getRollingDateRange } from '@/lib/time'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

import { ChannelStatCards } from './channel-stat-cards'

const PAGE_SIZE = 20
const TOP_LIMIT_OPTIONS = [5, 10, 20, 50, 0] as const

interface ChannelStatsSectionProps {
  filters: ChannelStatsFilters
  onFiltersChange: (filters: ChannelStatsFilters) => void
  includeCache?: boolean
}

type SortField =
  | 'channel_name'
  | 'model_name'
  | 'request_count'
  | 'prompt_tokens'
  | 'completion_tokens'
  | 'cached_tokens'
  | 'avg_first_byte_ms'
  | 'avg_speed_tok_per_s'
  | 'cache_hit_ratio'
  | 'success_rate'
  | 'total_tokens'
  | 'quota'

type SortDirection = 'asc' | 'desc'

const STRING_SORT_FIELDS: Set<SortField> = new Set([
  'channel_name',
  'model_name',
])

function RatioBar(props: {
  value: number
  max: number
  className?: string
  label: string
}) {
  const ratio =
    props.max > 0 ? Math.max(0, Math.min(props.value / props.max, 1)) : 0
  return (
    <div className='flex min-w-[88px] flex-col items-end gap-1'>
      <span className='tabular-nums'>{props.label}</span>
      <div className='bg-muted h-1 w-full max-w-[96px] overflow-hidden rounded-full'>
        <div
          className={cn('h-full rounded-full transition-all', props.className)}
          style={{ width: `${ratio * 100}%` }}
        />
      </div>
    </div>
  )
}

export function ChannelStatsSection(props: ChannelStatsSectionProps) {
  const { t, i18n } = useTranslation()
  const { refetchInterval } = useAutoRefresh()
  const userRole = useAuthStore((state) => state.auth.user?.role)
  const isAdmin = Boolean(userRole && userRole >= ROLE.ADMIN)

  const [compactMode, setCompactMode] = useState(true)
  const [selectedChannelId, setSelectedChannelId] = useState<number | null>(
    null
  )
  // Draft input vs applied keyword so users can type and click Search.
  const [searchInput, setSearchInput] = useState('')
  const [appliedQuery, setAppliedQuery] = useState('')
  const selectedRange = props.filters.selectedRange
  const topLimit = props.filters.topLimit
  const onFiltersChange = props.onFiltersChange
  const [currentPage, setCurrentPage] = useState(1)
  const [sortField, setSortField] = useState<SortField>('request_count')
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc')

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

  const handleTopLimitChange = useCallback(
    (limit: number) => {
      onFiltersChange({ ...props.filters, topLimit: limit })
      setCurrentPage(1)
    },
    [onFiltersChange, props.filters]
  )

  const { data: channelStatsData, isLoading } = useQuery({
    queryKey: [
      'dashboard',
      'channel-stats',
      timeRange,
      isAdmin,
      props.includeCache,
    ],
    queryFn: () =>
      isAdmin
        ? getChannelModelStats({
            ...timeRange,
            include_cache: props.includeCache,
          })
        : getSelfChannelModelStats({
            ...timeRange,
            include_cache: props.includeCache,
          }),
    select: (res) => (res.success ? res.data : []),
    staleTime: 60_000,
    refetchInterval: refetchInterval || undefined,
  })

  const channelOptions = useMemo(() => {
    const seen = new Map<number, string>()
    for (const item of (channelStatsData ?? []) as ChannelModelStatsItem[]) {
      if (!seen.has(item.channel_id)) {
        const name = (item.channel_name || '').trim()
        seen.set(
          item.channel_id,
          name || `${t('Channel')} #${item.channel_id}`
        )
      }
    }
    return Array.from(seen.entries())
      .map(([id, name]) => ({ id, name }))
      .sort((a, b) => a.name.localeCompare(b.name))
  }, [channelStatsData, t])

  // Base UI Select needs items=[{value,label}] so SelectValue shows labels, not raw ids.
  const channelSelectItems = useMemo(
    () => [
      { value: 'all', label: t('All channels') },
      ...channelOptions.map((ch) => ({
        value: String(ch.id),
        label: ch.name,
      })),
    ],
    [channelOptions, t]
  )

  const topLimitSelectItems = useMemo(
    () =>
      TOP_LIMIT_OPTIONS.map((limit) => ({
        value: String(limit),
        label:
          limit === 0
            ? t('Show all')
            : t('Top {{count}}', { count: limit }),
      })),
    [t]
  )

  const statsCardData = useMemo(() => {
    const raw = (channelStatsData ?? []) as ChannelModelStatsItem[]
    if (selectedChannelId === null) return raw
    return raw.filter((item) => Number(item.channel_id) === selectedChannelId)
  }, [channelStatsData, selectedChannelId])

  const selectedChannelName = useMemo(() => {
    if (selectedChannelId === null) return undefined
    return channelOptions.find((c) => c.id === selectedChannelId)?.name
  }, [channelOptions, selectedChannelId])

  const applySearch = useCallback(() => {
    setAppliedQuery(searchInput.trim())
    setCurrentPage(1)
  }, [searchInput])

  const filteredData = useMemo(() => {
    let raw = (channelStatsData ?? []) as ChannelModelStatsItem[]
    if (selectedChannelId !== null) {
      raw = raw.filter((item) => Number(item.channel_id) === selectedChannelId)
    }
    const query = appliedQuery.trim().toLowerCase()
    if (query) {
      raw = raw.filter((item) => {
        const model = (item.model_name || '').toLowerCase()
        const channel = (item.channel_name || '').toLowerCase()
        const channelId = String(item.channel_id)
        return (
          model.includes(query) ||
          channel.includes(query) ||
          channelId.includes(query)
        )
      })
    }
    return raw
  }, [channelStatsData, selectedChannelId, appliedQuery])

  const sortedData = useMemo(() => {
    const sorted = [...filteredData].sort((a, b) => {
      const aVal = a[sortField]
      const bVal = b[sortField]
      if (STRING_SORT_FIELDS.has(sortField)) {
        const sa = String(aVal ?? '')
        const sb = String(bVal ?? '')
        const cmp = sa.localeCompare(sb)
        return sortDirection === 'asc' ? cmp : -cmp
      }
      const cmp = Number(aVal ?? 0) - Number(bVal ?? 0)
      return sortDirection === 'asc' ? cmp : -cmp
    })

    // topLimit is applied after filter/sort for more intuitive UX.
    // 0 means show all rows.
    if (topLimit > 0) {
      return sorted.slice(0, topLimit)
    }
    return sorted
  }, [filteredData, topLimit, sortField, sortDirection])

  const maxRequestCount = useMemo(
    () =>
      sortedData.reduce(
        (max, item) => Math.max(max, Number(item.request_count) || 0),
        0
      ),
    [sortedData]
  )
  const maxTotalTokens = useMemo(
    () =>
      sortedData.reduce(
        (max, item) => Math.max(max, Number(item.total_tokens) || 0),
        0
      ),
    [sortedData]
  )
  const maxQuota = useMemo(
    () =>
      sortedData.reduce(
        (max, item) => Math.max(max, Number(item.quota) || 0),
        0
      ),
    [sortedData]
  )

  const totalPages = Math.max(1, Math.ceil(sortedData.length / PAGE_SIZE))
  const safePage = Math.min(currentPage, totalPages)
  const paginatedData = useMemo(
    () =>
      sortedData.slice((safePage - 1) * PAGE_SIZE, safePage * PAGE_SIZE),
    [sortedData, safePage]
  )

  const formatQuota = (quota: number) => formatQuotaWithCurrency(quota)
  const formatNum = (value: number) =>
    compactMode ? formatCompactNumber(value, locale) : formatNumber(value)

  const formatRatio = (value: number) => {
    if (value === 0) return '0%'
    return `${(value * 100).toFixed(1)}%`
  }

  const formatFloat = (value: number) => {
    if (value === 0) return '0'
    return value.toFixed(1)
  }

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection((prev) => (prev === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortField(field)
      setSortDirection('desc')
    }
    setCurrentPage(1)
  }

  const sortIcon = (field: SortField) => {
    if (sortField !== field) return null
    return (
      <ArrowUpDown
        className={`ml-1 inline-block size-3 ${
          sortDirection === 'asc' ? 'rotate-180' : ''
        }`}
      />
    )
  }

  const hasActiveFilters =
    selectedChannelId !== null ||
    appliedQuery.trim().length > 0 ||
    searchInput.trim().length > 0 ||
    topLimit !== 20

  const clearFilters = () => {
    setSelectedChannelId(null)
    setSearchInput('')
    setAppliedQuery('')
    if (topLimit !== 20) {
      onFiltersChange({ ...props.filters, topLimit: 20 })
    }
    setCurrentPage(1)
  }

  return (
    <div className='space-y-3'>
      <ChannelStatCards
        data={statsCardData}
        loading={isLoading}
        compact={compactMode}
        channelName={selectedChannelName}
      />

      <div className='bg-card flex flex-col gap-2 rounded-lg border p-2 sm:flex-row sm:flex-wrap sm:items-center sm:gap-2 sm:p-3'>
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

        <Select
          items={channelSelectItems}
          value={selectedChannelId === null ? 'all' : String(selectedChannelId)}
          onValueChange={(value) => {
            const next =
              value == null || value === 'all' ? null : Number(value)
            setSelectedChannelId(
              next != null && Number.isFinite(next) ? next : null
            )
            setCurrentPage(1)
          }}
        >
          <SelectTrigger
            className='h-8 w-full min-w-[160px] shrink-0 text-xs sm:w-[200px]'
            aria-label={t('Filter by channel')}
          >
            <SelectValue placeholder={t('All channels')} />
          </SelectTrigger>
          <SelectContent alignItemWithTrigger={false}>
            <SelectGroup>
              {channelSelectItems.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>

        <div className='flex min-w-0 flex-1 items-center gap-1.5 sm:max-w-[320px]'>
          <div className='relative min-w-0 flex-1'>
            <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-2 size-3.5 -translate-y-1/2' />
            <Input
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault()
                  applySearch()
                }
              }}
              placeholder={t('Search channel or model')}
              className='h-8 pl-7 text-xs'
            />
          </div>
          <Button
            type='button'
            variant='secondary'
            size='sm'
            className='h-8 shrink-0 gap-1 px-2.5 text-xs'
            onClick={applySearch}
          >
            <Search className='h-3 w-3' />
            {t('Search')}
          </Button>
        </div>

        <Select
          items={topLimitSelectItems}
          value={String(topLimit)}
          onValueChange={(value) => handleTopLimitChange(Number(value))}
        >
          <SelectTrigger
            className='h-8 w-full shrink-0 text-xs sm:w-[140px]'
            aria-label={t('Top Channels')}
          >
            <SelectValue />
          </SelectTrigger>
          <SelectContent alignItemWithTrigger={false}>
            <SelectGroup>
              {topLimitSelectItems.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>

        <Button
          variant='outline'
          size='sm'
          className='h-8 shrink-0 gap-1 px-2 text-xs'
          onClick={() => setCompactMode(!compactMode)}
          title={t('Number Format')}
        >
          <ArrowLeftRight className='h-3 w-3' />
          {compactMode ? t('Compact') : t('Precise')}
        </Button>

        {hasActiveFilters && (
          <Button
            variant='ghost'
            size='sm'
            className='h-8 shrink-0 gap-1 px-2 text-xs'
            onClick={clearFilters}
          >
            <X className='h-3 w-3' />
            {t('Clear filters')}
          </Button>
        )}

        {isLoading && (
          <Loader2 className='text-muted-foreground size-4 animate-spin' />
        )}
      </div>

      <div className='overflow-hidden rounded-lg border'>
        <div className='flex w-full items-center justify-between gap-2 border-b px-3 py-2 sm:px-5 sm:py-3'>
          <div className='flex min-w-0 items-center gap-2'>
            <Hash className='text-muted-foreground/60 size-4' />
            <div className='truncate text-sm font-semibold'>
              {t('Channel Statistics')}
            </div>
          </div>
          <div className='text-muted-foreground shrink-0 text-xs tabular-nums'>
            {t('{{total}} records', { total: sortedData.length })}
          </div>
        </div>

        <div className='overflow-x-auto'>
          {isLoading ? (
            <div className='p-4'>
              <Skeleton className='h-64 w-full' />
            </div>
          ) : sortedData.length === 0 ? (
            <div className='text-muted-foreground flex items-center justify-center p-8 text-sm'>
              {t('No data available')}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className='whitespace-nowrap'>
                    <button
                      className='hover:text-foreground inline-flex items-center'
                      onClick={() => handleSort('channel_name')}
                    >
                      {t('Channel Name')}
                      {sortIcon('channel_name')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap'>
                    <button
                      className='hover:text-foreground inline-flex items-center'
                      onClick={() => handleSort('model_name')}
                    >
                      {t('Model Name')}
                      {sortIcon('model_name')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    <button
                      className='hover:text-foreground inline-flex items-center'
                      onClick={() => handleSort('request_count')}
                    >
                      {t('Request Count')}
                      {sortIcon('request_count')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    <button
                      className='hover:text-foreground inline-flex items-center'
                      onClick={() => handleSort('prompt_tokens')}
                    >
                      {t('Input Tokens')}
                      {sortIcon('prompt_tokens')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    <button
                      className='hover:text-foreground inline-flex items-center'
                      onClick={() => handleSort('completion_tokens')}
                    >
                      {t('Output Tokens')}
                      {sortIcon('completion_tokens')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    <button
                      className='hover:text-foreground inline-flex items-center'
                      onClick={() => handleSort('cached_tokens')}
                    >
                      {t('Cached Tokens')}
                      {sortIcon('cached_tokens')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    <button
                      className='hover:text-foreground inline-flex items-center'
                      onClick={() => handleSort('avg_first_byte_ms')}
                    >
                      {t('Avg First Byte')}
                      {sortIcon('avg_first_byte_ms')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    <button
                      className='hover:text-foreground inline-flex items-center'
                      onClick={() => handleSort('avg_speed_tok_per_s')}
                    >
                      {t('Avg Speed')}
                      {sortIcon('avg_speed_tok_per_s')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    <button
                      className='hover:text-foreground inline-flex items-center'
                      onClick={() => handleSort('cache_hit_ratio')}
                    >
                      {t('Cache Hit Ratio')}
                      {sortIcon('cache_hit_ratio')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    <button
                      className='hover:text-foreground inline-flex items-center'
                      onClick={() => handleSort('success_rate')}
                    >
                      {t('Success Rate')}
                      {sortIcon('success_rate')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    <button
                      className='hover:text-foreground inline-flex items-center'
                      onClick={() => handleSort('total_tokens')}
                    >
                      {t('Total Tokens')}
                      {sortIcon('total_tokens')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    <button
                      className='hover:text-foreground inline-flex items-center'
                      onClick={() => handleSort('quota')}
                    >
                      {t('Quota')}
                      {sortIcon('quota')}
                    </button>
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {paginatedData.map((item, idx) => (
                  <TableRow
                    key={`${item.channel_id}-${item.model_name}-${idx}`}
                  >
                    <TableCell className='whitespace-nowrap font-medium'>
                      {item.channel_name || `#${item.channel_id}`}
                    </TableCell>
                    <TableCell className='whitespace-nowrap'>
                      {item.model_name}
                    </TableCell>
                    <TableCell className='whitespace-nowrap text-right'>
                      <div className='flex justify-end'>
                        <RatioBar
                          value={item.request_count}
                          max={maxRequestCount}
                          className='bg-info'
                          label={formatNum(item.request_count)}
                        />
                      </div>
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
                    <TableCell className='whitespace-nowrap text-right tabular-nums'>
                      {formatFloat(item.avg_first_byte_ms)} ms
                    </TableCell>
                    <TableCell className='whitespace-nowrap text-right tabular-nums'>
                      {formatFloat(item.avg_speed_tok_per_s)} tok/s
                    </TableCell>
                    <TableCell className='whitespace-nowrap text-right'>
                      <div className='flex justify-end'>
                        <RatioBar
                          value={item.cache_hit_ratio}
                          max={1}
                          className='bg-chart-4'
                          label={formatRatio(item.cache_hit_ratio)}
                        />
                      </div>
                    </TableCell>
                    <TableCell className='whitespace-nowrap text-right font-medium'>
                      <div className='flex justify-end'>
                        <RatioBar
                          value={item.success_rate}
                          max={1}
                          className={
                            item.success_rate >= 0.95
                              ? 'bg-success'
                              : item.success_rate >= 0.8
                                ? 'bg-warning'
                                : 'bg-destructive'
                          }
                          label={formatRatio(item.success_rate)}
                        />
                      </div>
                    </TableCell>
                    <TableCell className='whitespace-nowrap text-right'>
                      <div className='flex justify-end'>
                        <RatioBar
                          value={item.total_tokens}
                          max={maxTotalTokens}
                          className='bg-chart-2'
                          label={formatNum(item.total_tokens)}
                        />
                      </div>
                    </TableCell>
                    <TableCell className='whitespace-nowrap text-right'>
                      <div className='flex justify-end'>
                        <RatioBar
                          value={item.quota}
                          max={maxQuota}
                          className='bg-success'
                          label={formatQuota(item.quota)}
                        />
                      </div>
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
              {t('Page {{current}} of {{total}}', {
                current: safePage,
                total: totalPages,
              })}
            </div>
            <div className='flex items-center gap-1'>
              <Tabs
                value={String(safePage)}
                onValueChange={(v) => setCurrentPage(Number(v))}
              >
                <TabsList>
                  {Array.from({ length: totalPages }, (_, i) => i + 1)
                    .slice(0, 12)
                    .map((page) => (
                      <TabsTrigger
                        key={page}
                        value={String(page)}
                        className='px-2.5 text-xs'
                      >
                        {page}
                      </TabsTrigger>
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
