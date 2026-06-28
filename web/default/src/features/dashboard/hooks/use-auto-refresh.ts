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
import { createContext, useContext, useState, useCallback } from 'react'

export const AUTO_REFRESH_OPTIONS = [
  { value: 0, labelKey: 'Off' },
  { value: 15_000, labelKey: 'Every 15s' },
  { value: 30_000, labelKey: 'Every 30s' },
  { value: 60_000, labelKey: 'Every 60s' },
] as const

export type AutoRefreshInterval = (typeof AUTO_REFRESH_OPTIONS)[number]['value']

const STORAGE_KEY = 'dashboard-auto-refresh-interval'

function loadSavedInterval(): AutoRefreshInterval {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved != null) {
      const parsed = Number(saved)
      if (AUTO_REFRESH_OPTIONS.some((opt) => opt.value === parsed)) {
        return parsed as AutoRefreshInterval
      }
    }
  } catch {
    // ignore
  }
  return 0
}

interface AutoRefreshContextValue {
  refetchInterval: AutoRefreshInterval
  setRefetchInterval: (interval: AutoRefreshInterval) => void
}

const AutoRefreshContext = createContext<AutoRefreshContextValue>({
  refetchInterval: 0,
  setRefetchInterval: () => {},
})

export function useAutoRefresh() {
  const ctx = useContext(AutoRefreshContext)
  if (ctx == null) {
    throw new Error('useAutoRefresh must be used within AutoRefreshProvider')
  }
  return ctx
}

export { AutoRefreshContext }

export function useAutoRefreshState() {
  const [refetchInterval, setRefetchIntervalState] =
    useState<AutoRefreshInterval>(loadSavedInterval)

  const setRefetchInterval = useCallback((interval: AutoRefreshInterval) => {
    setRefetchIntervalState(interval)
    try {
      localStorage.setItem(STORAGE_KEY, String(interval))
    } catch {
      // ignore
    }
  }, [])

  return { refetchInterval, setRefetchInterval }
}
