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
import {
  createContext,
  useContext,
  useState,
  useCallback,
  useEffect,
  useRef,
} from 'react'

export const AUTO_REFRESH_OPTIONS = [
  { value: 15_000, labelKey: 'Every 15s' },
  { value: 30_000, labelKey: 'Every 30s' },
  { value: 60_000, labelKey: 'Every 60s' },
] as const

export type AutoRefreshInterval = (typeof AUTO_REFRESH_OPTIONS)[number]['value']

const STORAGE_KEY_INTERVAL = 'dashboard-auto-refresh-interval'
const STORAGE_KEY_ENABLED = 'dashboard-auto-refresh-enabled'

function loadSavedInterval(): AutoRefreshInterval {
  try {
    const saved = localStorage.getItem(STORAGE_KEY_INTERVAL)
    if (saved != null) {
      const parsed = Number(saved)
      if (AUTO_REFRESH_OPTIONS.some((opt) => opt.value === parsed)) {
        return parsed as AutoRefreshInterval
      }
    }
  } catch {
    // ignore
  }
  return 30_000
}

function loadSavedEnabled(): boolean {
  try {
    const saved = localStorage.getItem(STORAGE_KEY_ENABLED)
    if (saved != null) {
      return saved === 'true'
    }
  } catch {
    // ignore
  }
  return false
}

export interface AutoRefreshContextValue {
  /** Effective refetch interval (0 when disabled). */
  refetchInterval: number
  /** Selected interval (ignored when disabled). */
  selectedInterval: AutoRefreshInterval
  setSelectedInterval: (interval: AutoRefreshInterval) => void
  autoRefreshEnabled: boolean
  setAutoRefreshEnabled: (enabled: boolean) => void
  /** Seconds remaining until next auto-refresh (0 when disabled or idle). */
  countdown: number
  /** Reset the countdown timer (call on manual refresh). */
  resetCountdown: () => void
}

const AutoRefreshContext = createContext<AutoRefreshContextValue | null>(null)

/** Refetch interval for dashboard data queries. */
export function useAutoRefresh() {
  const ctx = useContext(AutoRefreshContext)
  if (ctx == null) {
    throw new Error('useAutoRefresh must be used within AutoRefreshProvider')
  }
  return { refetchInterval: ctx.refetchInterval }
}

/** Full auto-refresh controls (switch, interval, countdown). */
export function useAutoRefreshControls(): AutoRefreshContextValue {
  const ctx = useContext(AutoRefreshContext)
  if (ctx == null) {
    throw new Error(
      'useAutoRefreshControls must be used within AutoRefreshProvider'
    )
  }
  return ctx
}

export { AutoRefreshContext }

/**
 * Live countdown hook. Recalculates `countdown` seconds each tick.
 * Resets whenever `interval` changes, `enabled` toggles, or `resetKey` increments.
 *
 * Uses a single epoch (`lastResetRef`) so the displayed countdown stays aligned
 * with the wall-clock interval that React Query's `refetchInterval` follows
 * after the same enable/reset moment.
 */
function useAutoRefreshCountdown(
  interval: number,
  enabled: boolean,
  resetKey: number
): number {
  const [countdown, setCountdown] = useState(() =>
    enabled ? Math.floor(interval / 1000) : 0
  )
  const lastResetRef = useRef(Date.now())
  const intervalSec = Math.floor(interval / 1000)

  // Reset countdown when interval, enabled, or resetKey changes
  useEffect(() => {
    lastResetRef.current = Date.now()
    setCountdown(enabled ? intervalSec : 0)
  }, [interval, enabled, intervalSec, resetKey])

  // Tick every 250ms; cycle the epoch when a full interval elapses so the
  // next period matches a fresh React Query poll window.
  useEffect(() => {
    if (!enabled) return

    const timer = setInterval(() => {
      const elapsedMs = Date.now() - lastResetRef.current
      if (elapsedMs >= interval) {
        // Align to interval boundaries to reduce drift vs refetchInterval.
        const cycles = Math.floor(elapsedMs / interval)
        lastResetRef.current += cycles * interval
        setCountdown(intervalSec)
        return
      }
      const remaining = Math.max(
        0,
        intervalSec - Math.floor(elapsedMs / 1000)
      )
      setCountdown(remaining)
    }, 250)

    return () => clearInterval(timer)
  }, [enabled, interval, intervalSec])

  return countdown
}

/** Create auto-refresh state for the dashboard provider (call once per page). */
export function useAutoRefreshState(): AutoRefreshContextValue {
  const [autoRefreshEnabled, setAutoRefreshEnabledState] =
    useState<boolean>(loadSavedEnabled)
  const [selectedInterval, setSelectedIntervalState] =
    useState<AutoRefreshInterval>(loadSavedInterval)
  const [resetKey, setResetKey] = useState(0)

  const setAutoRefreshEnabled = useCallback((enabled: boolean) => {
    setAutoRefreshEnabledState(enabled)
    try {
      localStorage.setItem(STORAGE_KEY_ENABLED, String(enabled))
    } catch {
      // ignore
    }
  }, [])

  const setSelectedInterval = useCallback((interval: AutoRefreshInterval) => {
    setSelectedIntervalState(interval)
    try {
      localStorage.setItem(STORAGE_KEY_INTERVAL, String(interval))
    } catch {
      // ignore
    }
  }, [])

  const resetCountdown = useCallback(() => {
    setResetKey((k) => k + 1)
  }, [])

  const refetchInterval = autoRefreshEnabled ? selectedInterval : 0
  const countdown = useAutoRefreshCountdown(
    selectedInterval,
    autoRefreshEnabled,
    resetKey
  )

  return {
    refetchInterval,
    selectedInterval,
    setSelectedInterval,
    autoRefreshEnabled,
    setAutoRefreshEnabled,
    countdown,
    resetCountdown,
  }
}
