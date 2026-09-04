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
export interface ChannelOption {
  id: number
  name: string
}

export interface ChannelSelectItem {
  value: string
  label: string
}

export type ChannelSelectResolution =
  | { type: 'ignore' }
  | { type: 'set'; channelId: number | null }

export function mergeChannelSelectItems(
  options: ChannelOption[],
  selectedId: number | null,
  selectedFallbackName: string,
  allLabel: string
): ChannelSelectItem[] {
  const items: ChannelSelectItem[] = [{ value: 'all', label: allLabel }]
  const seen = new Set<number>()

  for (const option of options) {
    items.push({ value: String(option.id), label: option.name })
    seen.add(option.id)
  }

  if (selectedId !== null && !seen.has(selectedId)) {
    items.push({
      value: String(selectedId),
      label: selectedFallbackName,
    })
  }

  return items
}

export function resolveChannelSelectChange(args: {
  nextValue: string | null | undefined
  currentChannelId: number | null
  optionIds: number[]
  isLoading: boolean
}): ChannelSelectResolution {
  if (args.nextValue == null || args.nextValue === '') {
    return { type: 'ignore' }
  }

  if (args.nextValue === 'all') {
    const currentMissingFromOptions =
      args.currentChannelId !== null &&
      !args.optionIds.includes(args.currentChannelId)
    if (
      args.currentChannelId !== null &&
      (args.isLoading ||
        args.optionIds.length === 0 ||
        currentMissingFromOptions)
    ) {
      return { type: 'ignore' }
    }
    return { type: 'set', channelId: null }
  }

  const nextId = Number(args.nextValue)
  if (!Number.isFinite(nextId)) return { type: 'ignore' }
  return { type: 'set', channelId: nextId }
}
