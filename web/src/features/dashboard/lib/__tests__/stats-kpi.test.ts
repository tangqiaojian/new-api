import { describe, expect, it } from 'vitest'

import {
  calculateDashboardStats,
  formatKpiPercent,
  formatKpiTokenCount,
  formatTokens,
  formatTokenSplitLine,
  kpiCacheHitRate,
  kpiInputTokens,
  kpiSuccessRate,
} from '../stats'

describe('calculateDashboardStats token split', () => {
  it('sums prompt completion cache and success fields', () => {
    const stats = calculateDashboardStats([
      {
        created_at: 1,
        quota: 10,
        count: 2,
        token_used: 150,
        prompt_tokens: 100,
        completion_tokens: 50,
        cache_read_tokens: 40,
        cache_write_tokens: 5,
        success_count: 2,
        error_count: 0,
      },
      {
        created_at: 2,
        quota: 5,
        count: 1,
        token_used: 30,
        prompt_tokens: 20,
        completion_tokens: 10,
        cache_read_tokens: 8,
        cache_write_tokens: 1,
        success_count: 0,
        error_count: 1,
      },
    ])
    expect(stats.promptTokens).toBe(120)
    expect(stats.completionTokens).toBe(60)
    expect(stats.cacheReadTokens).toBe(48)
    expect(stats.cacheWriteTokens).toBe(6)
    expect(stats.successCount).toBe(2)
    expect(stats.errorCount).toBe(1)
    expect(stats.totalCount).toBe(3)
  })
})

describe('KPI helpers', () => {
  it('computes input tokens and rates from the reference formulas', () => {
    const stats = {
      promptTokens: 100,
      cacheReadTokens: 50,
      completionTokens: 20,
      successCount: 99,
      totalCount: 100,
    }
    expect(kpiInputTokens(stats)).toBe(150)
    expect(kpiSuccessRate(stats)).toBe(0.99)
    expect(kpiCacheHitRate(stats)).toBe(0.3333)
  })

  it('formats B/M/K with one decimal via formatTokens', () => {
    expect(formatTokens(1_500_000_000)).toBe('1.5B')
    expect(formatTokens(2_300_000)).toBe('2.3M')
    expect(formatTokens(4_200)).toBe('4.2K')
    expect(formatTokens(42)).toBe('42')
    expect(formatKpiTokenCount(42)).toBe('42')
    expect(formatKpiPercent(0.9941)).toBe('99.41%')
  })

  it('builds Sub2API token split subtitle in one line', () => {
    const line = formatTokenSplitLine((k) => k, {
      promptTokens: 1200,
      completionTokens: 340,
      cacheReadTokens: 80,
    })
    expect(line).toBe('Input: 1.2K / Output: 340 / Cache: 80')
  })
})
