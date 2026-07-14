import { describe, expect, it } from 'vitest'

import { aggregateTodayModelTokens } from './stats'

describe('aggregateTodayModelTokens', () => {
  it('aggregates rows by model and sorts by total tokens', () => {
    const summary = aggregateTodayModelTokens([
      {
        model_name: 'gpt-4o',
        date: '2026-07-14',
        prompt_tokens: 100,
        completion_tokens: 50,
        total_tokens: 150,
        cached_tokens: 10,
        request_count: 2,
        quota: 1,
      },
      {
        model_name: 'claude-sonnet',
        date: '2026-07-14',
        prompt_tokens: 200,
        completion_tokens: 100,
        total_tokens: 300,
        cached_tokens: 0,
        request_count: 3,
        quota: 2,
      },
      {
        model_name: 'gpt-4o',
        date: '2026-07-14',
        prompt_tokens: 50,
        completion_tokens: 25,
        total_tokens: 75,
        cached_tokens: 5,
        request_count: 1,
        quota: 1,
      },
    ])

    expect(summary.totalTokens).toBe(525)
    expect(summary.promptTokens).toBe(350)
    expect(summary.completionTokens).toBe(175)
    expect(summary.requestCount).toBe(6)
    expect(summary.models).toHaveLength(2)
    expect(summary.models[0].modelName).toBe('claude-sonnet')
    expect(summary.models[0].totalTokens).toBe(300)
    expect(summary.models[1].modelName).toBe('gpt-4o')
    expect(summary.models[1].totalTokens).toBe(225)
    expect(summary.models[0].share).toBeCloseTo(300 / 525)
    expect(summary.models[1].share).toBeCloseTo(225 / 525)
  })

  it('returns empty summary for empty input', () => {
    const summary = aggregateTodayModelTokens([])
    expect(summary.totalTokens).toBe(0)
    expect(summary.models).toEqual([])
  })
})
