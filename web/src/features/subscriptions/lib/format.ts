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
import type { TFunction } from 'i18next'

import dayjs from '@/lib/dayjs'

import type { SubscriptionPlan } from '../types'

export function formatDuration(
  plan: Partial<SubscriptionPlan>,
  t: TFunction
): string {
  const unit = plan?.duration_unit || 'month'
  const value = plan?.duration_value || 1
  const unitLabels: Record<string, string> = {
    year: t('years'),
    month: t('months'),
    day: t('days'),
    hour: t('hours'),
    custom: t('Custom (seconds)'),
  }
  if (unit === 'custom') {
    const seconds = plan?.custom_seconds || 0
    if (seconds >= 86400) return `${Math.floor(seconds / 86400)} ${t('days')}`
    if (seconds >= 3600) return `${Math.floor(seconds / 3600)} ${t('hours')}`
    return `${seconds} ${t('seconds')}`
  }
  return `${value} ${unitLabels[unit] || unit}`
}

export function formatResetPeriod(
  plan: Partial<SubscriptionPlan>,
  t: TFunction
): string {
  const period = plan?.quota_reset_period || 'never'
  if (period === 'daily') return t('Daily')
  if (period === 'weekly') return t('Weekly')
  if (period === 'monthly') return t('Monthly')
  if (period === 'custom') {
    const seconds = Number(plan?.quota_reset_custom_seconds || 0)
    if (seconds >= 86400) return `${Math.floor(seconds / 86400)} ${t('days')}`
    if (seconds >= 3600) return `${Math.floor(seconds / 3600)} ${t('hours')}`
    if (seconds >= 60) return `${Math.floor(seconds / 60)} ${t('minutes')}`
    return `${seconds} ${t('seconds')}`
  }
  return t('No Reset')
}

export function formatPlanType(
  planType: SubscriptionPlan['plan_type'] | string | undefined,
  t: TFunction
): string {
  if (planType === 'channel') return t('Channel')
  if (planType === 'both') return t('User and channel')
  return t('User')
}

export function formatResetSummary(
  plan: Partial<SubscriptionPlan>,
  t: TFunction
): string {
  const period = formatResetPeriod(plan, t)
  if ((plan.quota_reset_period || 'never') === 'never') {
    return period
  }
  if (!plan.quota_reset_anchor) {
    return period
  }
  const anchor = formatTimestamp(plan.quota_reset_anchor)
  const timezone = plan.quota_reset_timezone || t('Browser timezone')
  return `${period} · ${anchor} (${timezone})`
}

export function formatTimestamp(ts: number): string {
  if (!ts) return '-'
  return dayjs(ts * 1000).format('YYYY-MM-DD HH:mm:ss')
}

export function isChannelPoolRoutable(pool: {
  status: string
  amount_total: number
  amount_used: number
  tokens_total: number
  tokens_used: number
  start_time: number
  end_time: number
}): boolean {
  if (pool.status !== 'active') return false
  const now = Math.floor(Date.now() / 1000)
  if (pool.start_time > now || pool.end_time <= now) return false
  if (pool.amount_total > 0 && pool.amount_used >= pool.amount_total) {
    return false
  }
  if (pool.tokens_total > 0 && pool.tokens_used >= pool.tokens_total) {
    return false
  }
  return true
}
