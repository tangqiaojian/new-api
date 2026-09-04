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

import { DateTimePicker } from '@/components/datetime-picker'
import { Button } from '@/components/ui/button'
import { TIME_RANGE_PRESETS } from '@/features/dashboard/constants'
import {
  applyCustomTimeBound,
  buildTimeWindow,
  detectQuickRangeDays,
  type DashboardTimeWindow,
} from '@/features/dashboard/lib/time-range'
import { cn } from '@/lib/utils'

interface DashboardTimeRangeBarProps {
  value: DashboardTimeWindow
  onChange: (next: DashboardTimeWindow) => void
  className?: string
}

export function DashboardTimeRangeBar(props: DashboardTimeRangeBarProps) {
  const { t } = useTranslation()
  const selectedRange =
    detectQuickRangeDays(
      props.value.start_timestamp,
      props.value.end_timestamp
    ) ?? (props.value.selectedRange > 0 ? props.value.selectedRange : null)

  return (
    <div
      className={cn(
        'flex min-w-0 flex-wrap items-center gap-1.5 sm:gap-2',
        props.className
      )}
    >
      <div className='flex flex-wrap items-center gap-1.5'>
        {TIME_RANGE_PRESETS.map((preset) => (
          <Button
            key={preset.days}
            type='button'
            size='sm'
            variant={selectedRange === preset.days ? 'default' : 'outline'}
            className='h-8 px-2.5 text-xs'
            onClick={() => props.onChange(buildTimeWindow(preset.days))}
          >
            {t(preset.label)}
          </Button>
        ))}
      </div>

      <div className='flex min-w-0 flex-wrap items-center gap-1.5'>
        <span className='text-muted-foreground text-xs whitespace-nowrap'>
          {t('Start Time')}
        </span>
        <DateTimePicker
          value={props.value.start_timestamp}
          onChange={(date) =>
            props.onChange(
              applyCustomTimeBound(props.value, 'start_timestamp', date)
            )
          }
          placeholder={t('Select start time')}
          className='h-8 items-center'
        />
        <span className='text-muted-foreground text-xs whitespace-nowrap'>
          {t('End Time')}
        </span>
        <DateTimePicker
          value={props.value.end_timestamp}
          onChange={(date) =>
            props.onChange(
              applyCustomTimeBound(props.value, 'end_timestamp', date)
            )
          }
          placeholder={t('Select end time')}
          className='h-8 items-center'
        />
      </div>
    </div>
  )
}
