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
import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Activity,
  Zap,
  Timer,
  Database,
  TrendingUp,
  AlertCircle,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAutoRefresh } from '@/features/dashboard/hooks/use-auto-refresh'
import { formatCompactNumber, formatPercent } from '@/lib/format'
import { formatQuotaWithCurrency } from '@/lib/currency'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { StatCard } from '@/features/dashboard/components/ui/stat-card'
import { PanelWrapper } from '@/features/dashboard/components/ui/panel-wrapper'
import { getChannelStats } from '@/features/dashboard/api'
import { TIME_RANGE_PRESETS } from '@/features/dashboard/constants'
import type { DailyTokensFilters } from '@/features/dashboard/types'

type ChannelStatsSectionProps = {
  filters: DailyTokensFilters
  onFiltersChange: (filters: DailyTokensFilters) => void
}

export function ChannelStatsSection({
  filters,
  onFiltersChange,
}: ChannelStatsSectionProps) {
  const { t } = useTranslation()
  const autoRefresh = useAutoRefresh()
  const [includeCache, setIncludeCache] = useState(true)

  const { startTimestamp, endTimestamp } = useMemo(() => {
    const now = Math.floor(Date.now() / 1000)
    const days = filters.selectedRange ?? 7
    return {
      startTimestamp: now - days * 86400,
      endTimestamp: now,
    }
  }, [filters.selectedRange])

  const { data, isLoading, isError } = useQuery({
    queryKey: [
      'dashboard',
      'channel-stats',
      startTimestamp,
      endTimestamp,
      includeCache,
    ],
    queryFn: () =>
      getChannelStats({
        start_timestamp: startTimestamp,
        end_timestamp: endTimestamp,
        include_cache: includeCache,
      }),
    refetchInterval: autoRefresh.refetchInterval || false,
    staleTime: 30 * 1000,
  })

  const summary = data?.data
  const topChannels = summary?.top_channels ?? []
  const allChannels = summary?.all_channels ?? []

  return (
    <div className='space-y-3 sm:space-y-4'>
      {/* 时间范围选择 + 缓存开关 */}
      <div className='flex flex-wrap items-center justify-between gap-1.5 sm:gap-2'>
        <div className='flex flex-wrap items-center gap-1.5 sm:gap-2'>
          {TIME_RANGE_PRESETS.map((preset) => (
            <button
              key={preset.days}
              onClick={() =>
                onFiltersChange({ ...filters, selectedRange: preset.days })
              }
              className={`rounded-md px-2.5 py-1 text-xs font-medium transition-colors ${
                filters.selectedRange === preset.days
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-muted hover:bg-muted/80 text-muted-foreground'
              }`}
            >
              {t(preset.label)}
            </button>
          ))}
        </div>
        <div className='flex items-center gap-2'>
          <Switch
            id='channel-stats-include-cache'
            checked={includeCache}
            onCheckedChange={setIncludeCache}
          />
          <Label
            htmlFor='channel-stats-include-cache'
            className='text-muted-foreground cursor-pointer text-xs font-normal'
          >
            {t('Include cache')}
          </Label>
        </div>
      </div>

      {/* 汇总卡片 */}
      <div className='grid grid-cols-2 gap-2 sm:gap-3 lg:grid-cols-3'>
        <StatCard
          title={t('Success Rate')}
          value={summary ? formatPercent(summary.overall_success_rate) : '—'}
          description={t('Across all channels')}
          icon={Activity}
          tone='teal'
          loading={isLoading}
          error={isError}
        />
        <StatCard
          title={t('Avg Speed')}
          value={
            summary
              ? `${formatCompactNumber(summary.avg_use_time)}ms`
              : '—'
          }
          description={t('Across all channels')}
          icon={Zap}
          tone='rose'
          loading={isLoading}
          error={isError}
        />
        <StatCard
          title={t('Avg First Byte')}
          value={
            summary
              ? `${formatCompactNumber(summary.avg_first_byte)}ms`
              : '—'
          }
          description={t('Across all channels')}
          icon={Timer}
          tone='teal'
          loading={isLoading}
          error={isError}
        />
        <StatCard
          title={t('Cache Hit Ratio')}
          value={
            summary ? formatPercent(summary.overall_cache_hit_ratio) : '—'
          }
          description={t('Cached Tokens')}
          icon={Database}
          tone='teal'
          loading={isLoading}
          error={isError}
        />
        <StatCard
          title={t('Total Tokens')}
          value={
            summary ? formatCompactNumber(summary.total_tokens) : '—'
          }
          description={`${summary ? formatCompactNumber(summary.total_cached_tokens) : 0} ${t('Cached Tokens')}`}
          icon={TrendingUp}
          tone='gray'
          loading={isLoading}
          error={isError}
        />
        <StatCard
          title={t('Total Requests')}
          value={
            summary ? formatCompactNumber(summary.total_requests) : '—'
          }
          description={
            summary
              ? `${formatCompactNumber(summary.total_errors)} ${t('Error')}`
              : ''
          }
          icon={AlertCircle}
          tone={summary && summary.total_errors > 0 ? 'rose' : 'gray'}
          loading={isLoading}
          error={isError}
        />
      </div>

      {/* Top 渠道商表格 */}
      <PanelWrapper title={t('Top Channels')}>
        {isLoading ? (
          <Skeleton className='h-64 w-full' />
        ) : (
          <ChannelTable channels={topChannels} />
        )}
      </PanelWrapper>

      {/* 全部渠道商表格 */}
      <PanelWrapper title={t('All channels')}>
        {isLoading ? (
          <Skeleton className='h-64 w-full' />
        ) : (
          <ChannelTable channels={allChannels} />
        )}
      </PanelWrapper>
    </div>
  )
}

function ChannelTable({
  channels,
}: {
  channels: Array<{
    channel_id: number
    channel_name: string
    request_count: number
    success_rate: number
    avg_use_time: number
    avg_first_byte: number
    cache_hit_ratio: number
    cached_tokens: number
    total_tokens: number
    used_quota: number
  }>
}) {
  const { t } = useTranslation()

  if (channels.length === 0) {
    return (
      <div className='text-muted-foreground py-8 text-center text-sm'>
        {t('No data')}
      </div>
    )
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('Channel Name')}</TableHead>
          <TableHead className='text-right'>{t('Success Rate')}</TableHead>
          <TableHead className='text-right'>{t('Avg Speed')}</TableHead>
          <TableHead className='text-right'>{t('Avg First Byte')}</TableHead>
          <TableHead className='text-right'>{t('Cache Hit Ratio')}</TableHead>
          <TableHead className='text-right'>{t('Cached Tokens')}</TableHead>
          <TableHead className='text-right'>{t('Total Tokens')}</TableHead>
          <TableHead className='text-right'>{t('Total Requests')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {channels.map((ch) => (
          <TableRow key={ch.channel_id}>
            <TableCell className='font-medium'>{ch.channel_name}</TableCell>
            <TableCell className='text-right tabular-nums'>
              {formatPercent(ch.success_rate)}
            </TableCell>
            <TableCell className='text-right tabular-nums'>
              {formatCompactNumber(ch.avg_use_time)}ms
            </TableCell>
            <TableCell className='text-right tabular-nums'>
              {formatCompactNumber(ch.avg_first_byte)}ms
            </TableCell>
            <TableCell className='text-right tabular-nums'>
              {formatPercent(ch.cache_hit_ratio)}
            </TableCell>
            <TableCell className='text-right tabular-nums'>
              {formatCompactNumber(ch.cached_tokens)}
            </TableCell>
            <TableCell className='text-right tabular-nums'>
              {formatCompactNumber(ch.total_tokens)}
            </TableCell>
            <TableCell className='text-right tabular-nums'>
              {formatCompactNumber(ch.request_count)}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
