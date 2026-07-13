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
import { type ColumnDef } from '@tanstack/react-table'
import { Plus, RotateCcw } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { formatQuotaWithCurrency } from '@/lib/currency'
import {
  formatCompactNumber,
  formatTimestampToDate,
} from '@/lib/format'
import { cn } from '@/lib/utils'

import type { AdminUserSubscriptionItem } from '../types'

function getProgressColor(percentage: number): string {
  if (percentage >= 90) return '[&_[data-slot=progress-indicator]]:bg-rose-500'
  if (percentage >= 70) return '[&_[data-slot=progress-indicator]]:bg-amber-500'
  return '[&_[data-slot=progress-indicator]]:bg-emerald-500'
}

interface UseUserSubscriptionsColumnsProps {
  onAddQuota: (row: AdminUserSubscriptionItem) => void
  onReset: (row: AdminUserSubscriptionItem) => void
}

export function useUserSubscriptionsColumns({
  onAddQuota,
  onReset,
}: UseUserSubscriptionsColumnsProps): ColumnDef<AdminUserSubscriptionItem>[] {
  const { t } = useTranslation()

  return useMemo(
    (): ColumnDef<AdminUserSubscriptionItem>[] => [
      {
        accessorKey: 'username',
        id: 'username',
        header: t('Username'),
        cell: ({ row }) => (
          <span className='font-medium'>{row.original.username}</span>
        ),
        size: 140,
      },
      {
        accessorKey: 'plan_title',
        id: 'plan_title',
        header: t('Plan'),
        cell: ({ row }) => (
          <span className='text-muted-foreground truncate'>
            {row.original.plan_title || '-'}
          </span>
        ),
        size: 160,
      },
      {
        accessorKey: 'status',
        id: 'status',
        header: t('Status'),
        cell: ({ row }) => {
          const status = (row.original.status || '').toLowerCase()
          if (status === 'active') {
            return (
              <StatusBadge
                label={t('Active')}
                variant='success'
                copyable={false}
                className='-ml-1.5'
              />
            )
          }
          if (status === 'expired') {
            return (
              <StatusBadge
                label={t('Expired')}
                variant='neutral'
                copyable={false}
                className='-ml-1.5'
              />
            )
          }
          if (status === 'cancelled' || status === 'canceled') {
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
              label={row.original.status || '-'}
              variant='warning'
              copyable={false}
              className='-ml-1.5'
            />
          )
        },
        size: 100,
      },
      {
        id: 'quota_usage',
        header: t('Quota Usage'),
        cell: ({ row }) => {
          const { amount_total, amount_used } = row.original
          if (!amount_total || amount_total <= 0) {
            return (
              <span className='text-muted-foreground'>{t('Unlimited')}</span>
            )
          }
          const used = amount_used || 0
          const percentage = Math.min((used / amount_total) * 100, 100)
          return (
            <Tooltip>
              <TooltipTrigger
                render={<div className='w-[140px] cursor-help space-y-1' />}
              >
                <div className='flex justify-between text-xs'>
                  <span className='font-medium tabular-nums'>
                    {formatQuotaWithCurrency(used)}
                  </span>
                  <span className='text-muted-foreground tabular-nums'>
                    {formatQuotaWithCurrency(amount_total)}
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
                    {t('Total:')} {formatQuotaWithCurrency(amount_total)}
                  </div>
                  <div>
                    {t('Percentage:')} {percentage.toFixed(1)}%
                  </div>
                </div>
              </TooltipContent>
            </Tooltip>
          )
        },
        size: 170,
      },
      {
        id: 'token_usage',
        header: t('Token Usage'),
        cell: ({ row }) => {
          const { tokens_total, tokens_used } = row.original
          if (!tokens_total || tokens_total <= 0) {
            return (
              <span className='text-muted-foreground'>{t('Unlimited')}</span>
            )
          }
          const used = tokens_used || 0
          const percentage = Math.min((used / tokens_total) * 100, 100)
          return (
            <Tooltip>
              <TooltipTrigger
                render={<div className='w-[140px] cursor-help space-y-1' />}
              >
                <div className='flex justify-between text-xs'>
                  <span className='font-medium tabular-nums'>
                    {formatCompactNumber(used)}
                  </span>
                  <span className='text-muted-foreground tabular-nums'>
                    {formatCompactNumber(tokens_total)}
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
                    {t('Used:')} {formatCompactNumber(used)}
                  </div>
                  <div>
                    {t('Total:')} {formatCompactNumber(tokens_total)}
                  </div>
                  <div>
                    {t('Percentage:')} {percentage.toFixed(1)}%
                  </div>
                </div>
              </TooltipContent>
            </Tooltip>
          )
        },
        size: 170,
      },
      {
        id: 'next_reset',
        header: t('Next Reset'),
        cell: ({ row }) => {
          const { next_reset_time, token_next_reset_time } = row.original
          return (
            <div className='text-muted-foreground space-y-0.5 text-xs'>
              <div>
                {t('Quota')}: {formatTimestampToDate(next_reset_time)}
              </div>
              <div>
                {t('Token')}: {formatTimestampToDate(token_next_reset_time)}
              </div>
            </div>
          )
        },
        size: 180,
      },
      {
        id: 'end_time',
        header: t('End Time'),
        cell: ({ row }) => (
          <span className='text-muted-foreground'>
            {formatTimestampToDate(row.original.end_time)}
          </span>
        ),
        size: 160,
      },
      {
        id: 'actions',
        header: () => t('Actions'),
        cell: ({ row }) => (
          <div className='flex items-center gap-1.5'>
            <Button
              variant='outline'
              size='xs'
              onClick={() => onAddQuota(row.original)}
            >
              <Plus className='size-3.5' />
              {t('Add Quota')}
            </Button>
            <Button
              variant='outline'
              size='xs'
              onClick={() => onReset(row.original)}
            >
              <RotateCcw className='size-3.5' />
              {t('Reset Usage')}
            </Button>
          </div>
        ),
        meta: { pinned: 'right' as const },
        size: 180,
      },
    ],
    [t, onAddQuota, onReset]
  )
}
