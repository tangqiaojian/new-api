import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

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

    assert.equal(summary.totalTokens, 525)
    assert.equal(summary.promptTokens, 350)
    assert.equal(summary.completionTokens, 175)
    assert.equal(summary.requestCount, 6)
    assert.equal(summary.models.length, 2)
    assert.equal(summary.models[0].modelName, 'claude-sonnet')
    assert.equal(summary.models[0].totalTokens, 300)
    assert.equal(summary.models[1].modelName, 'gpt-4o')
    assert.equal(summary.models[1].totalTokens, 225)
    assert.ok(Math.abs(summary.models[0].share - 300 / 525) < 1e-9)
    assert.ok(Math.abs(summary.models[1].share - 225 / 525) < 1e-9)
  })

  it('returns empty summary for empty input', () => {
    const summary = aggregateTodayModelTokens([])
    assert.equal(summary.totalTokens, 0)
    assert.deepEqual(summary.models, [])
  })
})
