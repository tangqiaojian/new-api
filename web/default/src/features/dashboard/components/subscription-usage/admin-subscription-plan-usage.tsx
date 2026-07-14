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
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AlertCircle, Loader2, Search, Users } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Link } from '@tanstack/react-router'

import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Progress } from '@/components/ui/progress'
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
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useDebounce } from '@/hooks'
import { formatQuotaWithCurrency } from '@/lib/currency'
import {
  formatChineseNumber,
  formatCompactNumber,
  formatTimestampToDate,
} from '@/lib/format'
import { cn } from '@/lib/utils'
import { useAutoRefresh } from '@/features/dashboard/hooks/use-auto-refresh'
import { adminListAllSubscriptions } from '@/features/subscriptions/api'
import type { AdminUserSubscriptionItem } from '@/features/subscriptions/types'

const PAGE_SIZE = 20

type StatusFilter = 'active' | 'all' | 'expired' | 'cancelled'

function getProgressColor(percentage: number): string {
  if (percentage >= 90) return '[&_[data-slot=progress-indicator]]:bg-rose-500'
  if (percentage >= 70) return '[&_[data-slot=progress-indicator]]:bg-amber-500'
  return '[&_[data-slot=progress-indicator]]:bg-emerald-500'
}

function QuotaUsageCell({
  used,
  total,
  unlimitedLabel,
}: {
  used: number
  total: number
  unlimitedLabel: string
}) {
  const { t } = useTranslation()
  if (!total || total <= 0) {
    return <span className='text-muted-foreground text-xs'>{unlimitedLabel}</span>
  }
  const percentage = Math.min((used / total) * 100, 100)
  return (
    <Tooltip>
      <TooltipTrigger render={<div className='w-[150px] cursor-help space-y-1' />}>
        <div className='flex justify-between text-xs'>
          <span className='font-medium tabular-nums'>
            {formatQuotaWithCurrency(used)}
          </span>
          <span className='text-muted-foreground tabular-nums'>
            {formatQuotaWithCurrency(total)}
          </span>
        </div>
        <Progress
          value={percentage}
          className={cn('h-1.5', getProgressColor(percentage))}
        />
      </TooltipTrigger>
      <TooltipContent>
        <div className='space-y-1 text-xs'>
          <div>
            {t('Used:')} {formatQuotaWithCurrency(used)}
          </div>
          <div>
            {t('Total:')} {formatQuotaWithCurrency(total)}
          </div>
          <div>
            {t('Percentage:')} {percentage.toFixed(1)}%
          </div>
        </div>
      </TooltipContent>
    </Tooltip>
  )
}

function TokenUsageCell({
  used,
  total,
  unlimitedLabel,
}: {
  used: number
  total: number
  unlimitedLabel: string
}) {
  const { t } = useTranslation()
  if (!total || total <= 0) {
    return <span className='text-muted-foreground text-xs'>{unlimitedLabel}</span>
  }
  const percentage = Math.min((used / total) * 100, 100)
  return (
    <Tooltip>
      <TooltipTrigger render={<div className='w-[150px] cursor-help space-y-1' />}>
        <div className='flex justify-between text-xs'>
          <span className='font-medium tabular-nums'>
            {formatCompactNumber(used)}
          </span>
          <span className='text-muted-foreground tabular-nums'>
            {formatCompactNumber(total)}
          </span>
        </div>
        <Progress
          value={percentage}
          className={cn('h-1.5', getProgressColor(percentage))}
        />
      </TooltipTrigger>
      <TooltipContent>
        <div className='space-y-1 text-xs'>
          <div>
            {t('Used:')} {formatCompactNumber(used)}（{formatChineseNumber(used)}）
          </div>
          <div>
            {t('Total:')} {formatCompactNumber(total)}（
            {formatChineseNumber(total)}）
          </div>
          <div>
            {t('Percentage:')} {percentage.toFixed(1)}%
          </div>
        </div>
      </TooltipContent>
    </Tooltip>
  )
}

function StatusCell({ status }: { status: string }) {
  const { t } = useTranslation()
  const normalized = (status || '').toLowerCase()
  if (normalized === 'active') {
    return (
      <StatusBadge
        label={t('Active')}
        variant='success'
        copyable={false}
        className='-ml-1.5'
      />
    )
  }
  if (normalized === 'expired') {
    return (
      <StatusBadge
        label={t('Expired')}
        variant='neutral'
        copyable={false}
        className='-ml-1.5'
      />
    )
  }
  if (normalized === 'cancelled' || normalized === 'canceled') {
    return (
      <StatusBadge
        label={t('Cancelled')}
        variant='danger'
        copyable={false}
        className='-ml-1.5'
      />
    )
  }
  return (
    <StatusBadge
      label={status || '-'}
      variant='warning'
      copyable={false}
      className='-ml-1.5'
    />
  )
}

/**
 * Admin dashboard panel: each user's subscription plan usage
 * (quota amount + token amount), not log-based traffic charts.
 */
export function AdminSubscriptionPlanUsage() {
  const { t } = useTranslation()
  const { refetchInterval } = useAutoRefresh()
  const [page, setPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('active')
  const [searchInput, setSearchInput] = useState('')
  const [isComposing, setIsComposing] = useState(false)
  const debouncedSearch = useDebounce(searchInput, 400)
  const usernameFilter = isComposing ? undefined : debouncedSearch || undefined

  useEffect(() => {
    setPage(1)
  }, [debouncedSearch, statusFilter])

  const queryParams = useMemo(
    () => ({
      p: page,
      size: PAGE_SIZE,
      username: usernameFilter,
      status: statusFilter === 'all' ? undefined : statusFilter,
    }),
    [page, usernameFilter, statusFilter]
  )

  const { data, isLoading, isError, isFetching } = useQuery({
    queryKey: ['dashboard', 'admin-subscription-plan-usage', queryParams],
    queryFn: async () => {
      const res = await adminListAllSubscriptions(queryParams)
      if (!res?.success) {
        return { items: [] as AdminUserSubscriptionItem[], total: 0 }
      }
      return {
        items: res.data?.items ?? [],
        total: res.data?.total ?? 0,
      }
    },
    staleTime: 30_000,
    refetchInterval: refetchInterval || undefined,
    placeholderData: (prev) => prev,
  })

  const items = data?.items ?? []
  const total = data?.total ?? 0
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const summary = useMemo(() => {
    let highUsage = 0
    let quotaExhausted = 0
    let tokenExhausted = 0
    for (const row of items) {
      const amountTotal = row.amount_total || 0
      const tokensTotal = row.tokens_total || 0
      const amountPct =
        amountTotal > 0
          ? ((row.amount_used || 0) / amountTotal) * 100
          : 0
      const tokenPct =
        tokensTotal > 0
          ? ((row.tokens_used || 0) / tokensTotal) * 100
          : 0
      if (amountPct >= 80 || tokenPct >= 80) highUsage += 1
      if (amountTotal > 0 && (row.amount_used || 0) >= amountTotal) {
        quotaExhausted += 1
      }
      if (tokensTotal > 0 && (row.tokens_used || 0) >= tokensTotal) {
        tokenExhausted += 1
      }
    }
    return { highUsage, quotaExhausted, tokenExhausted }
  }, [items])

  const handleStatusChange = useCallback((value: string) => {
    setStatusFilter(value as StatusFilter)
  }, [])

  return (
    <div className='space-y-3'>
      <div className='flex flex-wrap items-center gap-2'>
        <Tabs value={statusFilter} onValueChange={handleStatusChange}>
          <TabsList>
            <TabsTrigger value='active' className='px-2.5 text-xs'>
              {t('Active')}
            </TabsTrigger>
            <TabsTrigger value='all' className='px-2.5 text-xs'>
              {t('All')}
            </TabsTrigger>
            <TabsTrigger value='expired' className='px-2.5 text-xs'>
              {t('Expired')}
            </TabsTrigger>
            <TabsTrigger value='cancelled' className='px-2.5 text-xs'>
              {t('Cancelled')}
            </TabsTrigger>
          </TabsList>
        </Tabs>

        <div className='relative'>
          <Search className='text-muted-foreground absolute top-1/2 left-2 size-3.5 -translate-y-1/2' />
          <Input
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            onCompositionStart={() => setIsComposing(true)}
            onCompositionEnd={(e) => {
              setIsComposing(false)
              setSearchInput(e.currentTarget.value)
            }}
            placeholder={t('Search by username')}
            className='h-8 w-48 pl-7 text-xs sm:w-56'
            aria-label={t('Username')}
          />
        </div>

        {(isLoading || isFetching) && (
          <Loader2 className='text-muted-foreground size-4 animate-spin' />
        )}

        <div className='text-muted-foreground ml-auto text-xs'>
          <Link
            to='/subscriptions'
            className='hover:text-foreground underline-offset-2 hover:underline'
          >
            {t('Manage Subscriptions')}
          </Link>
        </div>
      </div>

      <div className='overflow-hidden rounded-lg border'>
        <div className='flex items-center justify-between gap-2 border-b px-3 py-2 sm:px-4'>
          <div className='flex items-center gap-2 text-sm font-semibold'>
            <Users className='text-muted-foreground size-4' />
            {t('User subscription plan usage')}
          </div>
          <div className='text-muted-foreground text-xs'>
            {t('{{count}} subscriptions', { count: total })}
          </div>
        </div>
        <div className='divide-border/60 grid min-w-0 grid-cols-2 divide-x sm:grid-cols-4'>
          {(
            [
              {
                key: 'total',
                title: t('Subscriptions'),
                value: total,
              },
              {
                key: 'high',
                title: t('High usage (≥80%)'),
                value: summary.highUsage,
                hint: t('On this page'),
              },
              {
                key: 'quota_out',
                title: t('Quota exhausted'),
                value: summary.quotaExhausted,
                hint: t('On this page'),
              },
              {
                key: 'token_out',
                title: t('Token exhausted'),
                value: summary.tokenExhausted,
                hint: t('On this page'),
              },
            ] as const
          ).map((card) => (
            <div key={card.key} className='min-w-0 px-3 py-3 sm:px-4'>
              <div className='text-muted-foreground text-[11px] font-medium tracking-wide uppercase sm:text-xs'>
                {card.title}
              </div>
              {isLoading && items.length === 0 ? (
                <Skeleton className='mt-2 h-7 w-16' />
              ) : (
                <div className='mt-1 font-mono text-lg font-bold tabular-nums sm:text-2xl'>
                  {card.value}
                </div>
              )}
              {'hint' in card && card.hint ? (
                <div className='text-muted-foreground mt-0.5 text-[10px]'>
                  {card.hint}
                </div>
              ) : null}
            </div>
          ))}
        </div>
      </div>

      {isError && (
        <div className='border-destructive/30 bg-destructive/5 text-destructive flex items-center gap-2 rounded-lg border px-4 py-3 text-sm'>
          <AlertCircle className='size-4 shrink-0' />
          {t('Failed to load subscription usage data')}
        </div>
      )}

      <div className='overflow-hidden rounded-lg border'>
        <div className='overflow-x-auto'>
          {isLoading && items.length === 0 ? (
            <div className='p-4'>
              <Skeleton className='h-64 w-full' />
            </div>
          ) : items.length === 0 ? (
            <div className='text-muted-foreground flex items-center justify-center p-8 text-sm'>
              {t('No user subscriptions found')}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className='whitespace-nowrap'>
                    {t('Username')}
                  </TableHead>
                  <TableHead className='whitespace-nowrap'>{t('Plan')}</TableHead>
                  <TableHead className='whitespace-nowrap'>
                    {t('Status')}
                  </TableHead>
                  <TableHead className='whitespace-nowrap'>
                    {t('Quota Usage')}
                  </TableHead>
                  <TableHead className='whitespace-nowrap'>
                    {t('Token Usage')}
                  </TableHead>
                  <TableHead className='whitespace-nowrap'>
                    {t('Next Reset')}
                  </TableHead>
                  <TableHead className='whitespace-nowrap'>
                    {t('End Time')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell className='font-medium whitespace-nowrap'>
                      {row.username || `#${row.user_id}`}
                    </TableCell>
                    <TableCell className='text-muted-foreground max-w-[180px] truncate'>
                      {row.plan_title || `Plan #${row.plan_id}`}
                    </TableCell>
                    <TableCell>
                      <StatusCell status={row.status} />
                    </TableCell>
                    <TableCell>
                      <QuotaUsageCell
                        used={row.amount_used || 0}
                        total={row.amount_total || 0}
                        unlimitedLabel={t('Unlimited')}
                      />
                    </TableCell>
                    <TableCell>
                      <TokenUsageCell
                        used={row.tokens_used || 0}
                        total={row.tokens_total || 0}
                        unlimitedLabel={t('Unlimited')}
                      />
                    </TableCell>
                    <TableCell className='text-muted-foreground space-y-0.5 text-xs whitespace-nowrap'>
                      <div>
                        {t('Quota')}: {formatTimestampToDate(row.next_reset_time)}
                      </div>
                      <div>
                        {t('Token')}:{' '}
                        {formatTimestampToDate(row.token_next_reset_time)}
                      </div>
                    </TableCell>
                    <TableCell className='text-muted-foreground whitespace-nowrap'>
                      {formatTimestampToDate(row.end_time)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </div>

        {pageCount > 1 && (
          <div className='flex items-center justify-between gap-2 border-t px-3 py-2 sm:px-4'>
            <div className='text-muted-foreground text-xs'>
              {t('Page {{current}} of {{total}}', {
                current: page,
                total: pageCount,
              })}
            </div>
            <div className='flex items-center gap-1.5'>
              <Button
                variant='outline'
                size='sm'
                className='h-7 px-2 text-xs'
                disabled={page <= 1}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
              >
                {t('Previous')}
              </Button>
              <Button
                variant='outline'
                size='sm'
                className='h-7 px-2 text-xs'
                disabled={page >= pageCount}
                onClick={() => setPage((p) => Math.min(pageCount, p + 1))}
              >
                {t('Next')}
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
