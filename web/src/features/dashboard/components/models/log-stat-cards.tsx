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
import { useQuery } from '@tanstack/react-query'
import { useEffect, useMemo } from 'react'

import { getUserQuotaDates } from '@/features/dashboard/api'
import { OpsGroupCards } from '@/features/dashboard/components/ops-group-cards'
import { OpsStatsGrid } from '@/features/dashboard/components/ops-stats-grid'
import { useAutoRefresh } from '@/features/dashboard/hooks/use-auto-refresh'
import { useOpsDashboardStats } from '@/features/dashboard/hooks/use-ops-dashboard-stats'
import {
  buildQueryParams,
  getDefaultDays,
} from '@/features/dashboard/lib'
import type {
  QuotaDataItem,
  DashboardFilters,
} from '@/features/dashboard/types'
import { computeTimeRange } from '@/lib/time'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

interface LogStatCardsProps {
  filters?: DashboardFilters
  onDataUpdate?: (data: QuotaDataItem[], loading: boolean) => void
  includeCache?: boolean
}

export function LogStatCards(props: LogStatCardsProps) {
  const user = useAuthStore((state) => state.auth.user)
  const isAdmin = !!(user?.role && user.role >= ROLE.ADMIN)
  const { refetchInterval } = useAutoRefresh()
  const username = props.filters?.username?.trim() || undefined

  const timeRange = useMemo(
    () =>
      computeTimeRange(
        getDefaultDays(props.filters?.time_granularity),
        props.filters?.start_timestamp,
        props.filters?.end_timestamp
      ),
    [
      props.filters?.end_timestamp,
      props.filters?.start_timestamp,
      props.filters?.time_granularity,
    ]
  )
  const queryParams = useMemo(
    () => buildQueryParams(timeRange, props.filters),
    [props.filters, timeRange]
  )

  const { stats: opsStats, loading: opsLoading } = useOpsDashboardStats({
    username,
    includeCache: props.includeCache,
    refetchInterval: refetchInterval || false,
  })

  const quotaQuery = useQuery({
    queryKey: [
      'dashboard',
      'models',
      'log-stats',
      queryParams,
      isAdmin,
      props.includeCache,
    ],
    queryFn: () =>
      getUserQuotaDates(
        { ...queryParams, include_cache: props.includeCache },
        isAdmin
      ),
    staleTime: 60_000,
    refetchInterval: refetchInterval || undefined,
  })

  const data = useMemo(
    () => quotaQuery.data?.data ?? [],
    [quotaQuery.data?.data]
  )
  const onDataUpdate = props.onDataUpdate

  useEffect(() => {
    onDataUpdate?.(data, quotaQuery.isLoading)
  }, [data, onDataUpdate, quotaQuery.isLoading])

  const groupCards = useMemo(() => {
    const byModel = new Map<
      string,
      { name: string; todayCost: number; requests: number; tokens: number }
    >()
    for (const item of data) {
      const name = (item.model_name || '').trim() || 'unknown'
      const prev = byModel.get(name) || {
        name,
        todayCost: 0,
        requests: 0,
        tokens: 0,
      }
      prev.todayCost += Number(item.quota) || 0
      prev.requests += Number(item.count) || 0
      prev.tokens +=
        (Number(item.prompt_tokens) || 0) +
        (Number(item.completion_tokens) || 0) +
        (Number(item.cache_read_tokens) || 0)
      byModel.set(name, prev)
    }
    return [...byModel.values()]
      .sort((a, b) => b.todayCost - a.todayCost)
      .slice(0, 8)
  }, [data])

  return (
    <div className='space-y-3'>
      <OpsStatsGrid loading={opsLoading} stats={opsStats} />
      <OpsGroupCards loading={quotaQuery.isLoading} items={groupCards} />
    </div>
  )
}
