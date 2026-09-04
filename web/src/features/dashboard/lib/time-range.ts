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
import { TIME_RANGE_PRESETS } from '@/features/dashboard/constants'
import type { DashboardTimeWindow } from '@/features/dashboard/types'
import { dateToUnixTimestamp, getRollingDateRange } from '@/lib/time'

export type { DashboardTimeWindow }

const MS_PER_DAY = 86_400_000

export function buildTimeWindow(days: number): DashboardTimeWindow {
  const { start, end } = getRollingDateRange(days)
  return {
    selectedRange: days,
    start_timestamp: start,
    end_timestamp: end,
  }
}

export function detectQuickRangeDays(
  start?: Date,
  end?: Date
): number | null {
  if (!start || !end) return null
  const days = Math.round((end.getTime() - start.getTime()) / MS_PER_DAY)
  return TIME_RANGE_PRESETS.some((preset) => preset.days === days) ? days : null
}

export function applyCustomTimeBound(
  current: DashboardTimeWindow,
  field: 'start_timestamp' | 'end_timestamp',
  value: Date | undefined
): DashboardTimeWindow {
  const next: DashboardTimeWindow = {
    ...current,
    [field]: value,
  }
  const matched = detectQuickRangeDays(next.start_timestamp, next.end_timestamp)
  return {
    ...next,
    selectedRange: matched ?? 0,
  }
}

export function resolveUnixTimeRange(
  window: DashboardTimeWindow
): { start_timestamp: number; end_timestamp: number } {
  let start = window.start_timestamp
  let end = window.end_timestamp

  if (!start || !end) {
    const rolling = getRollingDateRange(
      window.selectedRange > 0 ? window.selectedRange : 1
    )
    start = start ?? rolling.start
    end = end ?? rolling.end
  }

  if (start.getTime() > end.getTime()) {
    const swapped = start
    start = end
    end = swapped
  }

  return {
    start_timestamp: dateToUnixTimestamp(start),
    end_timestamp: dateToUnixTimestamp(end),
  }
}
