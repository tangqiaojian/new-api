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
import { useTranslation } from 'react-i18next'

import {
  formatTokens,
  quotaProgressTone,
} from '@/features/dashboard/lib/stats'
import { formatQuota, formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'

export type OpsGroupCardItem = {
  name: string
  /** Primary header amount (Sub2API total_actual_cost). */
  totalCost?: number
  todayCost: number
  requests: number
  tokens: number
  /** 0–100 usage percent when a limit exists */
  limitPercent?: number
  resetAt?: number | null
}

interface OpsGroupCardsProps {
  items: OpsGroupCardItem[]
  loading?: boolean
  className?: string
}

function progressBarClass(percent: number): string {
  const tone = quotaProgressTone(percent)
  if (tone === 'red') return 'bg-red-500'
  if (tone === 'amber') return 'bg-amber-500'
  return 'bg-green-500'
}

/** Row-3 Sub2API group/channel mini cards with optional quota progress. */
export function OpsGroupCards(props: OpsGroupCardsProps) {
  const { t } = useTranslation()
  if (!props.loading && props.items.length === 0) {
    return null
  }

  return (
    <div className={cn('grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4', props.className)}>
      {(props.loading ? Array.from({ length: 4 }) : props.items).map(
        (item, index) => {
          const card = item as OpsGroupCardItem | undefined
          return (
            <div
              key={card?.name ?? `skeleton-${index}`}
              className='bg-card rounded-xl border p-4 shadow-sm'
            >
              <div className='flex items-start justify-between gap-2'>
                <div className='truncate text-sm font-semibold'>
                  {card?.name ?? '—'}
                </div>
                <div className='shrink-0 text-sm font-semibold tabular-nums'>
                  {formatQuota(card?.totalCost ?? card?.todayCost ?? 0)}
                </div>
              </div>
              <div className='mt-3 grid grid-cols-3 gap-2 text-xs'>
                <div>
                  <div className='text-muted-foreground'>{t('Today Cost')}</div>
                  <div className='mt-0.5 font-medium tabular-nums'>
                    {formatQuota(card?.todayCost ?? 0)}
                  </div>
                </div>
                <div>
                  <div className='text-muted-foreground'>{t('Requests')}</div>
                  <div className='mt-0.5 font-medium tabular-nums'>
                    {(card?.requests ?? 0).toLocaleString()}
                  </div>
                </div>
                <div>
                  <div className='text-muted-foreground'>{t('Tokens')}</div>
                  <div className='mt-0.5 font-medium tabular-nums'>
                    {formatTokens(card?.tokens ?? 0)}
                  </div>
                </div>
              </div>
              {card && card.limitPercent != null ? (
                <div className='mt-3'>
                  <div className='bg-muted h-1.5 overflow-hidden rounded-full'>
                    <div
                      className={cn(
                        'h-full rounded-full',
                        progressBarClass(card.limitPercent)
                      )}
                      style={{
                        width: `${Math.min(100, Math.max(0, card.limitPercent))}%`,
                      }}
                    />
                  </div>
                  <div className='text-muted-foreground mt-1 text-[10px]'>
                    {card.resetAt && card.resetAt > 0
                      ? `${t('Resets at')} ${formatTimestampToDate(card.resetAt)}`
                      : t('Never reset')}
                  </div>
                </div>
              ) : null}
            </div>
          )
        }
      )}
    </div>
  )
}
