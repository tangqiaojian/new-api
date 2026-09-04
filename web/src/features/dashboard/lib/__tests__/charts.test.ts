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
import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

import {
  processDailyModelTokensChartData,
  processDailyTokensChartData,
} from '../charts'

describe('daily token chart processing', () => {
  it('aggregates token totals per user for the ranking chart', () => {
    const result = processDailyTokensChartData(
      [
        {
          user_id: 1,
          username: 'alice',
          model_name: 'gpt-a',
          date: '2026-08-12',
          prompt_tokens: 60,
          completion_tokens: 40,
          total_tokens: 100,
          cached_tokens: 10,
          request_count: 2,
          quota: 1,
        },
        {
          user_id: 1,
          username: 'alice',
          model_name: 'gpt-a',
          date: '2026-08-13',
          prompt_tokens: 10,
          completion_tokens: 10,
          total_tokens: 20,
          cached_tokens: 0,
          request_count: 1,
          quota: 1,
        },
        {
          user_id: 2,
          username: 'bob',
          model_name: 'gpt-b',
          date: '2026-08-13',
          prompt_tokens: 50,
          completion_tokens: 20,
          total_tokens: 70,
          cached_tokens: 5,
          request_count: 3,
          quota: 1,
        },
      ],
      undefined,
      'total',
      2,
      false,
      'en-US'
    )

    assert.deepEqual(result.spec_tokens_rank.data[0].values, [
      { User: 'alice', Tokens: 120 },
      { User: 'bob', Tokens: 70 },
    ])
  })

  it('aggregates and sorts model request counts independently of token rank', () => {
    const result = processDailyModelTokensChartData(
      [
        {
          model_name: 'gpt-4o',
          date: '2026-08-13',
          prompt_tokens: 150,
          completion_tokens: 50,
          total_tokens: 200,
          cached_tokens: 0,
          request_count: 2,
          quota: 1,
        },
        {
          model_name: 'gpt-4o',
          date: '2026-08-12',
          prompt_tokens: 50,
          completion_tokens: 50,
          total_tokens: 100,
          cached_tokens: 0,
          request_count: 2,
          quota: 1,
        },
        {
          model_name: 'claude-sonnet',
          date: '2026-08-13',
          prompt_tokens: 100,
          completion_tokens: 50,
          total_tokens: 150,
          cached_tokens: 0,
          request_count: 5,
          quota: 1,
        },
      ],
      undefined,
      'total',
      2,
      false,
      'en-US'
    )

    assert.deepEqual(result.spec_model_request_count.data[0].values, [
      { Model: 'claude-sonnet', Requests: 5 },
      { Model: 'gpt-4o', Requests: 4 },
    ])
  })
})
