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
import { Hash, Loader2, ArrowLeftRight, ArrowUpDown } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toIntlLocale } from '@/i18n/languages'
import { getRollingDateRange } from '@/lib/time'
import { useAutoRefresh } from '@/features/dashboard/hooks/use-auto-refresh'
import { useAuthStore } from '@/stores/auth-store'
import { ROLE } from '@/lib/roles'
import { formatQuotaWithCurrency } from '@/lib/currency'
import { formatCompactNumber, formatNumber } from '@/lib/format'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  getChannelModelStats,
  getSelfChannelModelStats,
} from '@/features/dashboard/api'
import { TIME_RANGE_PRESETS } from '@/features/dashboard/constants'
import { ChannelStatCards } from './channel-stat-cards'
import type {
  ChannelModelStatsItem,
  ChannelStatsFilters,
} from '@/features/dashboard/types'

const PAGE_SIZE = 20

const TOP_LIMIT_OPTIONS = [5, 10, 20, 50]

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

const STRING_SORT_FIELDS: Set<SortField> = new Set(['channel_name', 'model_name'])

export function ChannelStatsSection(props: ChannelStatsSectionProps) {
  const { t, i18n } = useTranslation()
  const { refetchInterval } = useAutoRefresh()
  const userRole = useAuthStore((state) => state.auth.user?.role)
  const isAdmin = Boolean(userRole && userRole >= ROLE.ADMIN)

  const [compactMode, setCompactMode] = useState(true)
  const [selectedChannelId, setSelectedChannelId] = useState<number | null>(null)
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
    queryKey: ['dashboard', 'channel-stats', timeRange, isAdmin, props.includeCache],
    queryFn: () =>
      isAdmin
        ? getChannelModelStats({ ...timeRange, include_cache: props.includeCache })
        : getSelfChannelModelStats({ ...timeRange, include_cache: props.includeCache }),
    select: (res) => (res.success ? res.data : []),
    staleTime: 60_000,
    refetchInterval: refetchInterval || undefined,
  })

  // Unique channel list for the channel filter dropdown
  const channelOptions = useMemo(() => {
    const seen = new Map<number, string>()
    for (const item of (channelStatsData ?? []) as ChannelModelStatsItem[]) {
      if (!seen.has(item.channel_id)) {
        seen.set(item.channel_id, item.channel_name || String(item.channel_id))
      }
    }
    return Array.from(seen.entries()).map(([id, name]) => ({ id, name }))
  }, [channelStatsData])

  // Data for the stat cards: filtered by selected channel, or all
  const statsCardData = useMemo(() => {
    const raw = (channelStatsData ?? []) as ChannelModelStatsItem[]
    if (selectedChannelId === null) return raw
    return raw.filter((item) => item.channel_id === selectedChannelId)
  }, [channelStatsData, selectedChannelId])

  const selectedChannelName = useMemo(() => {
    if (selectedChannelId === null) return undefined
    return channelOptions.find((c) => c.id === selectedChannelId)?.name
  }, [channelOptions, selectedChannelId])

  // Sort data
  const sortedData = useMemo(() => {
    const raw = (channelStatsData ?? []) as ChannelModelStatsItem[]
    // For non-admin users (who only see their own data), don't apply topLimit
    // Admin users can adjust the limit to focus on top channels
    const effectiveLimit = isAdmin ? topLimit : Math.max(raw.length, topLimit)
    const topItems = raw.slice(0, effectiveLimit)
    const sorted = [...topItems].sort((a, b) => {
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
    return sorted
  }, [channelStatsData, topLimit, sortField, sortDirection, isAdmin])

  const totalPages = Math.ceil(sortedData.length / PAGE_SIZE)
  const paginatedData = useMemo(
    () => sortedData.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE),
    [sortedData, currentPage]
  )

  const formatQuota = (quota: number) => formatQuotaWithCurrency(quota)

  const formatNum = (value: number) =>
    compactMode
      ? formatCompactNumber(value, locale)
      : formatNumber(value)

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
        className={`inline-block size-3 ml-1 ${
          sortDirection === 'asc' ? 'rotate-180' : ''
        }`}
      />
    )
  }

  return (
    <div className='space-y-3'>
      <ChannelStatCards
        data={statsCardData}
        loading={isLoading}
        compact={compactMode}
        channelName={selectedChannelName}
      />

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

        {/* Show top limit control for all users — non-admin users may still want to filter */}
        <Tabs
          value={String(topLimit)}
          onValueChange={(value) => handleTopLimitChange(Number(value))}
          className='shrink-0'
        >
          <TabsList>
            <span className='text-muted-foreground px-2 text-xs font-medium whitespace-nowrap'>
              {t('Top Channels')}
            </span>
            {TOP_LIMIT_OPTIONS.map((limit) => (
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

        <Select
          value={selectedChannelId === null ? 'all' : String(selectedChannelId)}
          onValueChange={(value) =>
            setSelectedChannelId(value === 'all' ? null : Number(value))
          }
        >
          <SelectTrigger
            className='shrink-0 h-7 w-auto gap-1 text-xs'
            aria-label={t('Filter by channel')}
          >
            <SelectValue placeholder={t('All channels')} />
          </SelectTrigger>
          <SelectContent alignItemWithTrigger={false}>
            <SelectGroup>
              <SelectItem value='all'>{t('All channels')}</SelectItem>
              {channelOptions.map((ch) => (
                <SelectItem key={ch.id} value={String(ch.id)}>
                  {ch.name}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>

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

      {/* Data Table */}
      <div className='overflow-hidden rounded-lg border'>
        <div className='flex w-full items-center gap-2 border-b px-3 py-2 sm:px-5 sm:py-3'>
          <Hash className='text-muted-foreground/60 size-4' />
          <div className='text-sm font-semibold'>
            {t('Channel Statistics')}
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
                      className='inline-flex items-center hover:text-foreground'
                      onClick={() => handleSort('channel_name')}
                    >
                      {t('Channel Name')}
                      {sortIcon('channel_name')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap'>
                    <button
                      className='inline-flex items-center hover:text-foreground'
                      onClick={() => handleSort('model_name')}
                    >
                      {t('Model Name')}
                      {sortIcon('model_name')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    <button
                      className='inline-flex items-center hover:text-foreground'
                      onClick={() => handleSort('request_count')}
                    >
                      {t('Request Count')}
                      {sortIcon('request_count')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    <button
                      className='inline-flex items-center hover:text-foreground'
                      onClick={() => handleSort('prompt_tokens')}
                    >
                      {t('Input Tokens')}
                      {sortIcon('prompt_tokens')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    <button
                      className='inline-flex items-center hover:text-foreground'
                      onClick={() => handleSort('completion_tokens')}
                    >
                      {t('Output Tokens')}
                      {sortIcon('completion_tokens')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    <button
                      className='inline-flex items-center hover:text-foreground'
                      onClick={() => handleSort('cached_tokens')}
                    >
                      {t('Cached Tokens')}
                      {sortIcon('cached_tokens')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    <button
                      className='inline-flex items-center hover:text-foreground'
                      onClick={() => handleSort('avg_first_byte_ms')}
                    >
                      {t('Avg First Byte')}
                      {sortIcon('avg_first_byte_ms')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    <button
                      className='inline-flex items-center hover:text-foreground'
                      onClick={() => handleSort('avg_speed_tok_per_s')}
                    >
                      {t('Avg Speed')}
                      {sortIcon('avg_speed_tok_per_s')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    <button
                      className='inline-flex items-center hover:text-foreground'
                      onClick={() => handleSort('cache_hit_ratio')}
                    >
                      {t('Cache Hit Ratio')}
                      {sortIcon('cache_hit_ratio')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    <button
                      className='inline-flex items-center hover:text-foreground'
                      onClick={() => handleSort('success_rate')}
                    >
                      {t('Success Rate')}
                      {sortIcon('success_rate')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    <button
                      className='inline-flex items-center hover:text-foreground'
                      onClick={() => handleSort('total_tokens')}
                    >
                      {t('Total Tokens')}
                      {sortIcon('total_tokens')}
                    </button>
                  </TableHead>
                  <TableHead className='whitespace-nowrap text-right'>
                    <button
                      className='inline-flex items-center hover:text-foreground'
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
                  <TableRow key={`${item.channel_id}-${item.model_name}-${idx}`}>
                    <TableCell className='whitespace-nowrap font-medium'>
                      {item.channel_name || `#${item.channel_id}`}
                    </TableCell>
                    <TableCell className='whitespace-nowrap'>
                      {item.model_name}
                    </TableCell>
                    <TableCell className='whitespace-nowrap text-right tabular-nums'>
                      {formatNum(item.request_count)}
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
                    <TableCell className='whitespace-nowrap text-right tabular-nums'>
                      {formatRatio(item.cache_hit_ratio)}
                    </TableCell>
                    <TableCell className='whitespace-nowrap text-right tabular-nums font-medium'>
                      {formatRatio(item.success_rate)}
                    </TableCell>
                    <TableCell className='whitespace-nowrap text-right tabular-nums'>
                      {formatNum(item.total_tokens)}
                    </TableCell>
                    <TableCell className='whitespace-nowrap text-right tabular-nums'>
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
              {t('{{total}} records', { total: sortedData.length })}
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
