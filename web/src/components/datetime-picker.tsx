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
import { ChevronDownIcon } from 'lucide-react'
import * as React from 'react'
import { enUS, fr, ja, ru, vi, zhCN, zhTW } from 'react-day-picker/locale'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import { Label } from '@/components/ui/label'
import {
  NativeSelect,
  NativeSelectOption,
} from '@/components/ui/native-select'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'

const HOUR_OPTIONS = Array.from({ length: 24 }, (_, hour) =>
  hour.toString().padStart(2, '0')
)
const MINUTE_OPTIONS = Array.from({ length: 60 }, (_, minute) =>
  minute.toString().padStart(2, '0')
)

const calendarLocales = {
  en: enUS,
  zh: zhCN,
  zhCN: zhCN,
  'zh-CN': zhCN,
  zhTW: zhTW,
  'zh-TW': zhTW,
  fr,
  ru,
  ja,
  vi,
} as const

interface DateTimePickerProps {
  value?: Date
  onChange?: (date: Date | undefined) => void
  placeholder?: string
  className?: string
}

export function DateTimePicker({
  value,
  onChange,
  placeholder,
  className,
}: DateTimePickerProps) {
  const { t, i18n } = useTranslation()
  const placeholderText = placeholder ?? t('Select date')
  const calendarLocale =
    calendarLocales[i18n.language as keyof typeof calendarLocales] ?? enUS
  const currentYear = new Date().getFullYear()
  const [open, setOpen] = React.useState(false)
  const [date, setDate] = React.useState<Date | undefined>(value)
  const [month, setMonth] = React.useState<Date | undefined>(value)
  const [time, setTime] = React.useState<string>('00:00')

  React.useEffect(() => {
    setDate(value)
    setMonth(value)
    if (value) {
      const hours = value.getHours().toString().padStart(2, '0')
      const minutes = value.getMinutes().toString().padStart(2, '0')
      setTime(`${hours}:${minutes}`)
    }
  }, [value])

  const hourId = React.useId()
  const minuteId = React.useId()
  const [rawHours, rawMinutes] = time.split(':')
  const hours = rawHours || '00'
  const minutes = rawMinutes || '00'
  const triggerLabel = date
    ? dayjs(date).format('YYYY-MM-DD HH:mm')
    : placeholderText

  const applyDateAndTime = (
    selectedDate: Date,
    nextHours: number,
    nextMinutes: number
  ) => {
    const newDate = new Date(selectedDate)
    newDate.setHours(nextHours, nextMinutes, 0, 0)
    setDate(newDate)
    setMonth(newDate)
    onChange?.(newDate)
  }

  const handleDateSelect = (selectedDate: Date | undefined) => {
    if (selectedDate) {
      applyDateAndTime(selectedDate, Number(hours), Number(minutes))
      return
    }
    setDate(undefined)
    setMonth(undefined)
    onChange?.(undefined)
  }

  const handleTimePartChange = (
    part: 'hour' | 'minute',
    nextValue: string
  ) => {
    const nextHours = part === 'hour' ? nextValue : hours
    const nextMinutes = part === 'minute' ? nextValue : minutes
    setTime(`${nextHours}:${nextMinutes}`)
    if (date) {
      applyDateAndTime(date, Number(nextHours), Number(nextMinutes))
    }
  }

  const handleClear = () => {
    setDate(undefined)
    setMonth(undefined)
    setTime('00:00')
    onChange?.(undefined)
  }

  return (
    <div className={cn('flex gap-2', className)}>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger
          render={
            <Button
              variant='outline'
              className={cn(
                'flex-1 justify-between font-normal',
                !date && 'text-muted-foreground'
              )}
            />
          }
        >
          {triggerLabel}
          <ChevronDownIcon className='h-4 w-4 opacity-50' />
        </PopoverTrigger>
        <PopoverContent className='w-auto overflow-hidden p-0' align='start'>
          <Calendar
            mode='single'
            required
            selected={date}
            month={month}
            onMonthChange={setMonth}
            captionLayout='dropdown'
            onSelect={handleDateSelect}
            locale={calendarLocale}
            startMonth={new Date(currentYear - 100, 0)}
            endMonth={new Date(currentYear + 100, 11)}
          />
          <div className='flex items-end gap-2 border-t p-3'>
            <div className='flex min-w-0 flex-1 flex-col gap-1'>
              <Label htmlFor={hourId}>{t('Hour')}</Label>
              <NativeSelect
                id={hourId}
                aria-label={t('Hour')}
                value={hours}
                onChange={(event) =>
                  handleTimePartChange('hour', event.target.value)
                }
              >
                {HOUR_OPTIONS.map((hour) => (
                  <NativeSelectOption key={hour} value={hour}>
                    {hour}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </div>
            <div className='flex min-w-0 flex-1 flex-col gap-1'>
              <Label htmlFor={minuteId}>{t('Minute')}</Label>
              <NativeSelect
                id={minuteId}
                aria-label={t('Minute')}
                value={minutes}
                onChange={(event) =>
                  handleTimePartChange('minute', event.target.value)
                }
              >
                {MINUTE_OPTIONS.map((minute) => (
                  <NativeSelectOption key={minute} value={minute}>
                    {minute}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </div>
            <Button
              type='button'
              variant='outline'
              className='shrink-0'
              onClick={() => setOpen(false)}
            >
              {t('Done')}
            </Button>
          </div>
        </PopoverContent>
      </Popover>
      {date && (
        <Button
          type='button'
          variant='outline'
          size='icon'
          onClick={handleClear}
          className='shrink-0'
          aria-label={t('Clear')}
        >
          <span aria-hidden='true'>✕</span>
        </Button>
      )}
    </div>
  )
}
