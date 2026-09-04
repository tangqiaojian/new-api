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

import {
  mergeChannelSelectItems,
  resolveChannelSelectChange,
} from './channel-filters'

describe('channel stats filter persistence', () => {
  test('keeps the selected channel in the select items after a time-range rebuild', () => {
    const items = mergeChannelSelectItems(
      [{ id: 2, name: 'beta' }],
      9,
      'alpha',
      'All channels'
    )

    expect(items).toEqual([
      { value: 'all', label: 'All channels' },
      { value: '2', label: 'beta' },
      { value: '9', label: 'alpha' },
    ])
  })

  test('ignores a spurious all-reset while options are loading or empty', () => {
    expect(
      resolveChannelSelectChange({
        nextValue: 'all',
        currentChannelId: 9,
        optionIds: [],
        isLoading: true,
      })
    ).toEqual({ type: 'ignore' })

    expect(
      resolveChannelSelectChange({
        nextValue: null,
        currentChannelId: 9,
        optionIds: [9],
        isLoading: false,
      })
    ).toEqual({ type: 'ignore' })
  })

  test('ignores an all-reset when the current channel is missing from rebuilt options', () => {
    expect(
      resolveChannelSelectChange({
        nextValue: 'all',
        currentChannelId: 9,
        optionIds: [2, 3],
        isLoading: false,
      })
    ).toEqual({ type: 'ignore' })
  })

  test('accepts an explicit all-channels choice when the current channel is still listed', () => {
    expect(
      resolveChannelSelectChange({
        nextValue: 'all',
        currentChannelId: 9,
        optionIds: [2, 9],
        isLoading: false,
      })
    ).toEqual({ type: 'set', channelId: null })
  })

  test('accepts a concrete channel id', () => {
    expect(
      resolveChannelSelectChange({
        nextValue: '12',
        currentChannelId: 9,
        optionIds: [12],
        isLoading: false,
      })
    ).toEqual({ type: 'set', channelId: 12 })
  })
})
