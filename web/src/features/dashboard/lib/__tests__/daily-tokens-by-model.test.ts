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
import { describe, expect, test } from 'vitest'

import { processDailyTokensChartData } from '../charts'
import type { DailyTokenDataItem } from '../../types'

function tokenRow(
  partial: Pick<
    DailyTokenDataItem,
    'user_id' | 'username' | 'model_name' | 'date' | 'total_tokens'
  > &
    Partial<DailyTokenDataItem>
): DailyTokenDataItem {
  return {
    prompt_tokens: 0,
    completion_tokens: 0,
    cached_tokens: 0,
    request_count: 1,
    quota: 0,
    ...partial,
  }
}

describe('daily token user+model chart processing', () => {
  test('aggregates each user token share by model across dates', () => {
    const result = processDailyTokensChartData(
      [
        tokenRow({
          user_id: 1,
          username: 'alice',
          model_name: 'gpt-a',
          date: '2026-08-12',
          prompt_tokens: 60,
          completion_tokens: 40,
          total_tokens: 100,
        }),
        tokenRow({
          user_id: 1,
          username: 'alice',
          model_name: 'gpt-a',
          date: '2026-08-13',
          total_tokens: 20,
        }),
        tokenRow({
          user_id: 1,
          username: 'alice',
          model_name: 'gpt-b',
          date: '2026-08-13',
          total_tokens: 30,
        }),
        tokenRow({
          user_id: 2,
          username: 'bob',
          model_name: 'gpt-a',
          date: '2026-08-13',
          total_tokens: 70,
        }),
      ],
      undefined,
      'total',
      10,
      false,
      'en-US'
    )

    expect(result.spec_tokens_rank.data[0].values).toEqual([
      { User: 'alice', Tokens: 150 },
      { User: 'bob', Tokens: 70 },
    ])
    expect(result.spec_tokens_by_model.data[0].values).toEqual([
      { User: 'alice', Model: 'gpt-a', Tokens: 120 },
      { User: 'alice', Model: 'gpt-b', Tokens: 30 },
      { User: 'bob', Model: 'gpt-a', Tokens: 70 },
      { User: 'bob', Model: 'gpt-b', Tokens: 0 },
    ])
  })

  test('keeps user ranking intact when rows are already split by model', () => {
    const result = processDailyTokensChartData(
      [
        tokenRow({
          user_id: 1,
          username: 'alice',
          model_name: 'gpt-a',
          date: '2026-08-12',
          total_tokens: 100,
        }),
        tokenRow({
          user_id: 1,
          username: 'alice',
          model_name: 'gpt-b',
          date: '2026-08-13',
          total_tokens: 20,
        }),
        tokenRow({
          user_id: 2,
          username: 'bob',
          model_name: 'gpt-a',
          date: '2026-08-13',
          total_tokens: 70,
        }),
      ],
      undefined,
      'total',
      2,
      false,
      'en-US'
    )

    expect(result.spec_tokens_rank.data[0].values).toEqual([
      { User: 'alice', Tokens: 120 },
      { User: 'bob', Tokens: 70 },
    ])
  })
})
