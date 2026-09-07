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
  if (Number.isNaN(result) || !Number.isFinite(result)) return 0
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
      promptTokens: acc.promptTokens + (Number(item.prompt_tokens) || 0),
      completionTokens:
        acc.completionTokens + (Number(item.completion_tokens) || 0),
      cacheReadTokens:
        acc.cacheReadTokens + (Number(item.cache_read_tokens) || 0),
      cacheWriteTokens:
        acc.cacheWriteTokens + (Number(item.cache_write_tokens) || 0),
      successCount: acc.successCount + (Number(item.success_count) || 0),
      errorCount: acc.errorCount + (Number(item.error_count) || 0),
    }),
    {
      totalQuota: 0,
      totalCount: 0,
      totalTokens: 0,
      promptTokens: 0,
      completionTokens: 0,
      cacheReadTokens: 0,
      cacheWriteTokens: 0,
      successCount: 0,
      errorCount: 0,
    }
  )
}

/** Input tokens for KPI cards: prompt + cache reads (subtitle: includes cache hits). */
export function kpiInputTokens(stats: {
  promptTokens: number
  cacheReadTokens: number
}): number {
  return stats.promptTokens + stats.cacheReadTokens
}

export function kpiSuccessRate(stats: {
  successCount: number
  totalCount: number
}): number {
  return safeDivide(stats.successCount, stats.totalCount, 4)
}

export function kpiCacheHitRate(stats: {
  promptTokens: number
  cacheReadTokens: number
}): number {
  return safeDivide(
    stats.cacheReadTokens,
    stats.promptTokens + stats.cacheReadTokens,
    4
  )
}

/** Format token counts: B / M / K with 1 decimal (Sub2API style). */
export function formatTokens(value: number): string {
  const abs = Math.abs(value)
  if (abs >= 1e9) return `${(value / 1e9).toFixed(1)}B`
  if (abs >= 1e6) return `${(value / 1e6).toFixed(1)}M`
  if (abs >= 1e3) return `${(value / 1e3).toFixed(1)}K`
  return String(Math.round(value))
}

/** @deprecated Prefer formatTokens — kept for older KPI helpers. */
export function formatKpiTokenCount(value: number): string {
  return formatTokens(value)
}

export function formatKpiPercent(rate: number): string {
  return `${(rate * 100).toFixed(2)}%`
}

export function formatDurationSeconds(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return '0ms'
  if (seconds >= 1) return `${seconds.toFixed(2)}s`
  return `${Math.round(seconds * 1000)}ms`
}

export type TokenSplit = {
  promptTokens: number
  completionTokens: number
  cacheReadTokens: number
}

export type OpsBucketStats = {
  requests: number
  quota: number
  promptTokens: number
  completionTokens: number
  cacheReadTokens: number
  cacheWriteTokens: number
  tokenUsed: number
}

export type OpsDashboardStats = {
  balanceQuota: number
  usedQuota: number
  apiKeysTotal: number
  apiKeysActive: number
  today: OpsBucketStats
  lifetime: OpsBucketStats
  rpm: number
  tpm: number
  avgUseTimeSec: number
}

export function emptyOpsBucket(): OpsBucketStats {
  return {
    requests: 0,
    quota: 0,
    promptTokens: 0,
    completionTokens: 0,
    cacheReadTokens: 0,
    cacheWriteTokens: 0,
    tokenUsed: 0,
  }
}

export function bucketFromDashboardStats(
  stats: ReturnType<typeof calculateDashboardStats> | null
): OpsBucketStats {
  if (!stats) return emptyOpsBucket()
  return {
    requests: stats.totalCount,
    quota: stats.totalQuota,
    promptTokens: stats.promptTokens,
    completionTokens: stats.completionTokens,
    cacheReadTokens: stats.cacheReadTokens,
    cacheWriteTokens: stats.cacheWriteTokens,
    tokenUsed: stats.totalTokens,
  }
}

export function formatTokenSplitLine(
  t: (key: string) => string,
  split: TokenSplit
): string {
  return `${t('Input')}: ${formatTokens(split.promptTokens)} / ${t('Output')}: ${formatTokens(split.completionTokens)} / ${t('Cache')}: ${formatTokens(split.cacheReadTokens)}`
}

export function quotaProgressTone(percent: number): 'green' | 'amber' | 'red' {
  if (percent >= 95) return 'red'
  if (percent >= 75) return 'amber'
  return 'green'
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

  const models = [...byModel.values()].sort(
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
