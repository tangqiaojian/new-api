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
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { useTheme } from '@/context/theme-provider'
import { formatQuota } from '@/lib/format'
import type { QuotaDataItem } from '@/features/dashboard/types'
import { VCHART_OPTION } from '@/lib/vchart'
import { cn } from '@/lib/utils'

interface OpsDistributionPieProps {
  data: QuotaDataItem[]
  loading?: boolean
  onSelectUser?: (username: string) => void
}

type DistMode = 'model' | 'user'

/** Donut that toggles Model distribution ↔ User consumption ranking (Sub2API). */
export function OpsDistributionPie(props: OpsDistributionPieProps) {
  const { t } = useTranslation()
  const { resolvedTheme } = useTheme()
  const [mode, setMode] = useState<DistMode>('model')

  const values = useMemo(() => {
    const bag = new Map<string, number>()
    for (const item of props.data) {
      const key =
        mode === 'user'
          ? (item.username || '').trim() || t('Unknown')
          : (item.model_name || '').trim() || t('Unknown')
      bag.set(key, (bag.get(key) || 0) + (Number(item.quota) || 0))
    }
    return [...bag.entries()]
      .map(([name, quota]) => ({ name, quota }))
      .sort((a, b) => b.quota - a.quota)
      .slice(0, 12)
  }, [mode, props.data, t])

  if (props.loading) {
    return (
      <div className='bg-card h-72 animate-pulse rounded-xl border shadow-sm' />
    )
  }
  if (values.length === 0) return null

  return (
    <div className='bg-card overflow-hidden rounded-xl border shadow-sm'>
      <div className='flex flex-wrap items-center justify-between gap-2 border-b px-4 py-2'>
        <div className='text-sm font-semibold'>
          {mode === 'model'
            ? t('Model Distribution')
            : t('User Consumption Ranking')}
        </div>
        <div className='bg-muted/60 inline-flex rounded-lg border p-0.5'>
          {(
            [
              ['model', t('Model Distribution')],
              ['user', t('User Consumption Ranking')],
            ] as const
          ).map(([value, label]) => (
            <button
              key={value}
              type='button'
              className={cn(
                'rounded-md px-2.5 py-1 text-xs font-medium',
                mode === value
                  ? 'bg-background text-foreground shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              )}
              onClick={() => setMode(value)}
            >
              {label}
            </button>
          ))}
        </div>
      </div>
      <div className='h-72 p-2'>
        <VChart
          key={`dist-${mode}-${resolvedTheme}-${values.length}`}
          spec={{
            type: 'pie',
            data: [{ id: 'dist', values }],
            valueField: 'quota',
            categoryField: 'name',
            outerRadius: 0.8,
            innerRadius: 0.55,
            legends: { visible: true, orient: 'right' },
            label: {
              visible: true,
              formatMethod: (_: string, data: { name?: string; quota?: number }) =>
                `${data?.name ?? ''}: ${formatQuota(Number(data?.quota) || 0)}`,
            },
            theme: resolvedTheme === 'dark' ? 'dark' : 'light',
            background: 'transparent',
          }}
          option={VCHART_OPTION}
          onReady={(chart: { on: (event: string, cb: (e: { datum?: { name?: string } }) => void) => void }) => {
            if (mode !== 'user' || !props.onSelectUser) return
            chart.on('click', (event) => {
              const name = event?.datum?.name
              if (name) props.onSelectUser?.(name)
            })
          }}
        />
      </div>
    </div>
  )
}
