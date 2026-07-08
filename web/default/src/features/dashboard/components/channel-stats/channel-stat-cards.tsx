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
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { ArrowDownToLine, ArrowUpFromLine, Database, Hash } from 'lucide-react'
import { formatCompactNumber, formatNumber } from '@/lib/format'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import type { ChannelModelStatsItem } from '@/features/dashboard/types'

interface ChannelStatCardsProps {
  data: ChannelModelStatsItem[]
  loading?: boolean
  compact?: boolean
  channelName?: string
}

function formatStatNumber(value: number, locale: Intl.LocalesArgument, compact: boolean) {
  return compact ? formatCompactNumber(value, locale) : formatNumber(value)
}

export function ChannelStatCards(props: ChannelStatCardsProps) {
  const { t, i18n } = useTranslation()
  const locale = i18n.resolvedLanguage || 'en'
  const compact = props.compact ?? true
  const data = props.data ?? []
  const subtitle = props.channelName
    ? t('Across {{channel}}', { channel: props.channelName })
    : t('Across all channels')

  const stats = useMemo(() => {
    return data.reduce(
      (acc, item) => ({
        promptTokens: acc.promptTokens + (item.prompt_tokens || 0),
        completionTokens: acc.completionTokens + (item.completion_tokens || 0),
        cachedTokens: acc.cachedTokens + (item.cached_tokens || 0),
        totalTokens: acc.totalTokens + (item.total_tokens || 0),
      }),
      { promptTokens: 0, completionTokens: 0, cachedTokens: 0, totalTokens: 0 }
    )
  }, [data])

  const items = [
    {
      key: 'prompt',
      title: t('Input Tokens'),
      value: formatStatNumber(stats.promptTokens, locale, compact),
      fullValue: formatNumber(stats.promptTokens, locale),
      icon: ArrowDownToLine,
    },
    {
      key: 'completion',
      title: t('Output Tokens'),
      value: formatStatNumber(stats.completionTokens, locale, compact),
      fullValue: formatNumber(stats.completionTokens, locale),
      icon: ArrowUpFromLine,
    },
    {
      key: 'cached',
      title: t('Cached Tokens'),
      value: formatStatNumber(stats.cachedTokens, locale, compact),
      fullValue: formatNumber(stats.cachedTokens, locale),
      icon: Database,
    },
    {
      key: 'total',
      title: t('Total Tokens'),
      value: formatStatNumber(stats.totalTokens, locale, compact),
      fullValue: formatNumber(stats.totalTokens, locale),
      icon: Hash,
    },
  ]

  return (
    <div className='overflow-hidden rounded-lg border'>
      <div className='divide-border/60 grid min-w-0 grid-cols-2 divide-x sm:grid-cols-4'>
        {items.map((it, idx) => {
          const Icon = it.icon
          return (
            <div
              key={it.key}
              className={cn(
                'min-w-0 px-3 py-2.5 sm:px-5 sm:py-4',
                idx === items.length - 1 &&
                  items.length % 2 !== 0 &&
                  'col-span-2 sm:col-span-1'
              )}
            >
              <div className='flex min-w-0 items-center gap-2'>
                <Icon className='text-muted-foreground/60 size-3.5 shrink-0' />
                <div className='text-muted-foreground truncate text-xs font-medium tracking-wider uppercase'>
                  {it.title}
                </div>
              </div>

              {props.loading ? (
                <div className='mt-2 flex flex-col gap-1.5'>
                  <Skeleton className='h-7 w-20' />
                  <Skeleton className='h-3.5 w-28' />
                </div>
              ) : (
                <>
                  <div
                    className='text-foreground mt-1.5 max-w-full truncate font-mono text-lg font-bold tracking-tight tabular-nums sm:mt-2 sm:text-2xl'
                    title={it.fullValue}
                  >
                    {it.value}
                  </div>
                  <div className='text-muted-foreground/60 mt-1 hidden text-xs md:block'>
                    {subtitle}
                  </div>
                </>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
