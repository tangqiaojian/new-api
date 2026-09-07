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
import { useQueries, useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'

import { getUserQuotaDates } from '@/features/dashboard/api'
import {
  bucketFromDashboardStats,
  calculateDashboardStats,
  emptyOpsBucket,
  type OpsDashboardStats,
} from '@/features/dashboard/lib/stats'
import { getApiKeys } from '@/features/keys/api'
import { getLogStats, getUserLogStats } from '@/features/usage-logs/api'
import dayjs from '@/lib/dayjs'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

export type OpsStatsScope = {
  /** Optional username filter (admin viewing one user). */
  username?: string
  includeCache?: boolean
  refetchInterval?: number | false
}

function todayRangeUnix() {
  const start = dayjs().tz().startOf('day').unix()
  const end = dayjs().tz().unix() + 3600
  return { start_timestamp: start, end_timestamp: end }
}

function lifetimeRangeUnix() {
  // Cap at ~90 days for export tables; user.used_quota still shows account totals.
  const end = dayjs().tz().unix() + 3600
  const start = end - 90 * 24 * 3600
  return { start_timestamp: start, end_timestamp: end }
}

export function useOpsDashboardStats(scope: OpsStatsScope = {}) {
  const user = useAuthStore((s) => s.auth.user)
  const isAdmin = !!(user?.role && user.role >= ROLE.ADMIN)
  const today = useMemo(() => todayRangeUnix(), [])
  const lifetime = useMemo(() => lifetimeRangeUnix(), [])
  const username = scope.username?.trim() || undefined

  const quotaQueries = useQueries({
    queries: [
      {
        queryKey: [
          'ops-stats',
          'quota',
          'today',
          today,
          username,
          isAdmin,
          scope.includeCache,
        ],
        queryFn: () =>
          getUserQuotaDates(
            {
              ...today,
              username,
              include_cache: scope.includeCache,
            },
            isAdmin
          ),
        staleTime: 30_000,
        refetchInterval: scope.refetchInterval || undefined,
      },
      {
        queryKey: [
          'ops-stats',
          'quota',
          'lifetime',
          lifetime,
          username,
          isAdmin,
          scope.includeCache,
        ],
        queryFn: () =>
          getUserQuotaDates(
            {
              ...lifetime,
              username,
              include_cache: scope.includeCache,
            },
            isAdmin
          ),
        staleTime: 60_000,
        refetchInterval: scope.refetchInterval || undefined,
      },
    ],
  })

  const keysQuery = useQuery({
    queryKey: ['ops-stats', 'api-keys', username ?? 'self'],
    queryFn: () => getApiKeys({ p: 1, size: 100 }),
    staleTime: 60_000,
    enabled: !username || username === user?.username,
  })

  const logStatQuery = useQuery({
    queryKey: ['ops-stats', 'log-stat', username, isAdmin],
    queryFn: async () => {
      const params = {
        type: 2,
        start_timestamp: lifetime.start_timestamp,
        end_timestamp: lifetime.end_timestamp,
        ...(username ? { username } : {}),
      }
      const res = isAdmin
        ? await getLogStats(params)
        : await getUserLogStats(params)
      return res.success ? res.data : null
    },
    staleTime: 15_000,
    refetchInterval: scope.refetchInterval || undefined,
  })

  const loading =
    quotaQueries.some((q) => q.isLoading) ||
    keysQuery.isLoading ||
    logStatQuery.isLoading

  const stats: OpsDashboardStats | null = useMemo(() => {
    const todayData = quotaQueries[0]?.data?.data ?? []
    const lifetimeData = quotaQueries[1]?.data?.data ?? []
    const todayBucket = bucketFromDashboardStats(
      todayData.length ? calculateDashboardStats(todayData) : null
    )
    const lifetimeBucket = bucketFromDashboardStats(
      lifetimeData.length ? calculateDashboardStats(lifetimeData) : null
    )

    const keys = keysQuery.data?.data?.items ?? []
    const apiKeysTotal = keysQuery.data?.data?.total ?? keys.length
    const apiKeysActive = keys.filter((k) => k.status === 1).length

    // Prefer account-level lifetime counters (Sub2API "Total"); fall back to
    // the 90-day quota aggregation window when viewing another user / no fields.
    let lifetimeFinal = emptyOpsBucket()
    if (lifetimeBucket.requests || lifetimeBucket.quota) {
      lifetimeFinal = { ...lifetimeBucket }
    } else if (todayBucket.requests) {
      lifetimeFinal = { ...todayBucket }
    }
    const accountRequests = Number(user?.request_count)
    if (
      (!username || username === user?.username) &&
      Number.isFinite(accountRequests) &&
      accountRequests > 0
    ) {
      lifetimeFinal = {
        ...lifetimeFinal,
        requests: accountRequests,
      }
    }
    const accountUsed = Number(user?.used_quota)
    if (
      (!username || username === user?.username) &&
      Number.isFinite(accountUsed) &&
      accountUsed > 0
    ) {
      lifetimeFinal = {
        ...lifetimeFinal,
        quota: accountUsed,
        // No separate lifetime standard without full-history scan; keep actual.
        standardQuota: Math.max(lifetimeFinal.standardQuota, accountUsed),
      }
    }

    return {
      balanceQuota: user?.quota ?? 0,
      usedQuota: user?.used_quota ?? 0,
      apiKeysTotal,
      apiKeysActive,
      today: todayBucket,
      lifetime: lifetimeFinal,
      rpm: logStatQuery.data?.rpm ?? 0,
      tpm: logStatQuery.data?.tpm ?? 0,
      avgUseTimeSec: Number(logStatQuery.data?.avg_use_time ?? 0),
    }
  }, [
    keysQuery.data,
    logStatQuery.data,
    quotaQueries,
    user?.quota,
    user?.request_count,
    user?.used_quota,
    user?.username,
    username,
  ])

  return { stats, loading }
}
