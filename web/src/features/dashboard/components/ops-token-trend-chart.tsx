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
import { VChart } from '@visactor/react-vchart'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { useTheme } from '@/context/theme-provider'
import type { QuotaDataItem } from '@/features/dashboard/types'
import dayjs from '@/lib/dayjs'
import { VCHART_OPTION } from '@/lib/vchart'

interface OpsTokenTrendChartProps {
  data: QuotaDataItem[]
  loading?: boolean
}

/** Stacked/ overlay Token trend: Input / Output / Cache (Sub2API style). */
export function OpsTokenTrendChart(props: OpsTokenTrendChartProps) {
  const { t } = useTranslation()
  const { resolvedTheme } = useTheme()

  const values = useMemo(() => {
    const byHour = new Map<
      number,
      { prompt: number; completion: number; cache: number }
    >()
    for (const item of props.data) {
      const hour = Number(item.created_at) || 0
      const prev = byHour.get(hour) || { prompt: 0, completion: 0, cache: 0 }
      prev.prompt += Number(item.prompt_tokens) || 0
      prev.completion += Number(item.completion_tokens) || 0
      prev.cache += Number(item.cache_read_tokens) || 0
      byHour.set(hour, prev)
    }
    const rows: { Time: string; Kind: string; Tokens: number }[] = []
    const sorted = [...byHour.entries()].sort((a, b) => a[0] - b[0])
    for (const [ts, bag] of sorted) {
      const label = dayjs.unix(ts).tz().format('MM-DD HH:mm')
      rows.push({ Time: label, Kind: t('Input'), Tokens: bag.prompt })
      rows.push({ Time: label, Kind: t('Output'), Tokens: bag.completion })
      rows.push({ Time: label, Kind: t('Cache'), Tokens: bag.cache })
    }
    return rows
  }, [props.data, t])

  if (props.loading) {
    return (
      <div className='bg-card h-64 animate-pulse rounded-xl border shadow-sm' />
    )
  }
  if (values.length === 0) {
    return null
  }

  return (
    <div className='bg-card overflow-hidden rounded-xl border shadow-sm'>
      <div className='border-b px-4 py-2 text-sm font-semibold'>
        {t('Token Trend')}
      </div>
      <div className='h-72 p-2'>
        <VChart
          key={`token-trend-${resolvedTheme}-${values.length}`}
          spec={{
            type: 'area',
            data: [{ id: 'tokenTrend', values }],
            xField: 'Time',
            yField: 'Tokens',
            seriesField: 'Kind',
            stack: false,
            legends: { visible: true, position: 'top' },
            theme: resolvedTheme === 'dark' ? 'dark' : 'light',
            background: 'transparent',
          }}
          option={VCHART_OPTION}
        />
      </div>
    </div>
  )
}
