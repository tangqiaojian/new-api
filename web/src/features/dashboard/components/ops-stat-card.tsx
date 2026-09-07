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
import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'

import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

export type OpsIconTone =
  | 'emerald'
  | 'blue'
  | 'green'
  | 'purple'
  | 'amber'
  | 'indigo'
  | 'violet'
  | 'rose'

const TONE_CLASSES: Record<OpsIconTone, string> = {
  emerald:
    'bg-emerald-100 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-400',
  blue: 'bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400',
  green: 'bg-green-100 text-green-600 dark:bg-green-900/30 dark:text-green-400',
  purple:
    'bg-purple-100 text-purple-600 dark:bg-purple-900/30 dark:text-purple-400',
  amber: 'bg-amber-100 text-amber-600 dark:bg-amber-900/30 dark:text-amber-400',
  indigo:
    'bg-indigo-100 text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-400',
  violet:
    'bg-violet-100 text-violet-600 dark:bg-violet-900/30 dark:text-violet-400',
  rose: 'bg-rose-100 text-rose-600 dark:bg-rose-900/30 dark:text-rose-400',
}

interface OpsStatCardProps {
  title: string
  value: ReactNode
  subtitle?: ReactNode
  icon: LucideIcon
  tone: OpsIconTone
  loading?: boolean
  className?: string
}

/** Sub2API-style compact card: icon square left, title / value / subtitle right. */
export function OpsStatCard(props: OpsStatCardProps) {
  const Icon = props.icon
  return (
    <div
      className={cn(
        'bg-card text-card-foreground flex items-start gap-3 rounded-xl border p-4 shadow-sm',
        props.className
      )}
    >
      <div
        className={cn(
          'flex size-9 shrink-0 items-center justify-center rounded-lg sm:size-10',
          TONE_CLASSES[props.tone]
        )}
      >
        <Icon className='size-4 sm:size-5' aria-hidden='true' />
      </div>
      <div className='min-w-0 flex-1'>
        <div className='text-muted-foreground text-xs leading-none'>
          {props.title}
        </div>
        {props.loading ? (
          <Skeleton className='mt-1.5 h-5 w-24' />
        ) : (
          <div className='mt-1.5 text-xl leading-tight font-semibold tracking-tight tabular-nums'>
            {props.value}
          </div>
        )}
        {(() => {
          if (props.loading) {
            return <Skeleton className='mt-1.5 h-3 w-32' />
          }
          if (props.subtitle) {
            return (
              <div className='text-muted-foreground mt-1 text-xs leading-snug'>
                {props.subtitle}
              </div>
            )
          }
          return null
        })()}
      </div>
    </div>
  )
}
