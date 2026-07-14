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
import type {
  DailyModelTokenDataItem,
  QuotaDataItem,
} from '@/features/dashboard/types'

/**
 * Safe division: handles NaN and Infinity cases
 */
export function safeDivide(
  value: number,
  divisor: number,
  precision: number = 3
): number {
  const result = value / divisor
  if (isNaN(result) || !isFinite(result)) return 0
  const factor = Math.pow(10, precision)
  return Math.round(result * factor) / factor
}

/**
 * Calculate aggregated statistics from quota data
 */
export function calculateDashboardStats(data: QuotaDataItem[]) {
  return data.reduce(
    (acc, item) => ({
      totalQuota: acc.totalQuota + (Number(item.quota) || 0),
      totalCount: acc.totalCount + (Number(item.count) || 0),
      totalTokens: acc.totalTokens + (Number(item.token_used) || 0),
    }),
    { totalQuota: 0, totalCount: 0, totalTokens: 0 }
  )
}

export type TodayModelTokenRow = {
  modelName: string
  promptTokens: number
  completionTokens: number
  totalTokens: number
  cachedTokens: number
  requestCount: number
  quota: number
  share: number
}

export type TodayModelTokenSummary = {
  promptTokens: number
  completionTokens: number
  totalTokens: number
  cachedTokens: number
  requestCount: number
  quota: number
  models: TodayModelTokenRow[]
}

/**
 * Aggregate daily model token rows into per-model totals with share ratios.
 * Rows without a model name are bucketed as "unknown".
 */
export function aggregateTodayModelTokens(
  data: DailyModelTokenDataItem[]
): TodayModelTokenSummary {
  const byModel = new Map<string, TodayModelTokenRow>()

  for (const item of data) {
    const modelName = (item.model_name || '').trim() || 'unknown'
    const existing = byModel.get(modelName)
    if (existing) {
      existing.promptTokens += Number(item.prompt_tokens) || 0
      existing.completionTokens += Number(item.completion_tokens) || 0
      existing.totalTokens += Number(item.total_tokens) || 0
      existing.cachedTokens += Number(item.cached_tokens) || 0
      existing.requestCount += Number(item.request_count) || 0
      existing.quota += Number(item.quota) || 0
      continue
    }
    byModel.set(modelName, {
      modelName,
      promptTokens: Number(item.prompt_tokens) || 0,
      completionTokens: Number(item.completion_tokens) || 0,
      totalTokens: Number(item.total_tokens) || 0,
      cachedTokens: Number(item.cached_tokens) || 0,
      requestCount: Number(item.request_count) || 0,
      quota: Number(item.quota) || 0,
      share: 0,
    })
  }

  const models = Array.from(byModel.values()).sort(
    (a, b) => b.totalTokens - a.totalTokens || b.requestCount - a.requestCount
  )
  const totals = models.reduce(
    (acc, row) => ({
      promptTokens: acc.promptTokens + row.promptTokens,
      completionTokens: acc.completionTokens + row.completionTokens,
      totalTokens: acc.totalTokens + row.totalTokens,
      cachedTokens: acc.cachedTokens + row.cachedTokens,
      requestCount: acc.requestCount + row.requestCount,
      quota: acc.quota + row.quota,
    }),
    {
      promptTokens: 0,
      completionTokens: 0,
      totalTokens: 0,
      cachedTokens: 0,
      requestCount: 0,
      quota: 0,
    }
  )
  const denom = Math.max(totals.totalTokens, 1)
  for (const row of models) {
    row.share = row.totalTokens / denom
  }

  return { ...totals, models }
}
